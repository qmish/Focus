package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
	"github.com/qmish/focus-api/internal/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func getTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "host=localhost port=5432 user=focus password=focus dbname=focus_test sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Discard,
	})
	if err != nil {
		t.Skipf("Skipping DB-dependent test: %v", err)
	}
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Room{}, &models.RoomParticipant{}, &models.Message{}, &models.MessageReaction{}))
	t.Cleanup(func() {
		db.Exec("DELETE FROM messages")
		db.Exec("DELETE FROM rooms")
		db.Exec("DELETE FROM users WHERE email LIKE '%@test-thread.com'")
	})
	return db
}

func setupHandlerWithDB(t *testing.T) (*MessageHandler, *gorm.DB) {
	t.Helper()
	db := getTestDB(t)
	msgRepo := repository.NewMessageRepository(db)
	userRepo := repository.NewUserRepository(db)
	wsHub := websocket.NewHub(zap.NewNop())
	go wsHub.Run()
	return NewMessageHandler(msgRepo, userRepo, wsHub, nil), db
}

func seedTestUser(t *testing.T, db *gorm.DB, id uuid.UUID) {
	t.Helper()
	now := time.Now()
	db.Create(&models.User{
		ID: id, Name: "Thread Tester", Email: id.String() + "@test-thread.com",
		Roles: models.StringArray{"user"}, IsActive: true,
		CreatedAt: now, UpdatedAt: now,
	})
}

func seedTestRoom(t *testing.T, db *gorm.DB, id, creatorID uuid.UUID) {
	t.Helper()
	now := time.Now()
	db.Create(&models.Room{
		ID: id, Name: "Thread Test Room", Type: "public",
		CreatorID: creatorID, JitsiRoomName: "jitsi-" + id.String(),
		CreatedAt: now, UpdatedAt: now,
	})
}

func withClaimsFor(req *http.Request, userID string) *http.Request {
	return withClaimsForRoles(req, userID, []string{"user"})
}

func withClaimsForRoles(req *http.Request, userID string, roles []string) *http.Request {
	ctx := context.WithValue(req.Context(), auth.ContextKeyUserClaims, &auth.SessionClaims{
		UserID: userID, Email: "test@test-thread.com", Name: "Test", Roles: roles,
	})
	return req.WithContext(ctx)
}

// setupHandlerWithRoomRepo возвращает хендлер с привязанным RoomRepository (для тестов прав удаления).
func setupHandlerWithRoomRepo(t *testing.T) (*MessageHandler, *gorm.DB) {
	t.Helper()
	db := getTestDB(t)
	msgRepo := repository.NewMessageRepository(db)
	userRepo := repository.NewUserRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	wsHub := websocket.NewHub(zap.NewNop())
	go wsHub.Run()
	h := NewMessageHandler(msgRepo, userRepo, wsHub, nil)
	h.SetRoomRepository(roomRepo)
	return h, db
}

// addRoomParticipant добавляет участника в комнату с указанной ролью (для тестов).
func addRoomParticipant(t *testing.T, db *gorm.DB, roomID, userID uuid.UUID, role models.ParticipantRole) {
	t.Helper()
	require.NoError(t, db.Create(models.NewRoomParticipant(roomID, userID, role)).Error)
}

// createMessageInDB создаёт сообщение напрямую в БД (без HTTP), позволяя задать createdAt.
func createMessageInDB(t *testing.T, db *gorm.DB, roomID, userID uuid.UUID, content string, createdAt time.Time) *models.Message {
	t.Helper()
	msg := models.NewMessage(roomID, userID, content, models.MessageTypeText)
	msg.CreatedAt = createdAt
	msg.UpdatedAt = createdAt
	require.NoError(t, db.Create(msg).Error)
	return msg
}

// --- Unit tests (no DB required) ---

func newTestHandler() *MessageHandler {
	wsHub := websocket.NewHub(zap.NewNop())
	go wsHub.Run()
	return NewMessageHandler(nil, nil, wsHub, nil)
}

func TestListMessages_MissingRoomID(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	rr := httptest.NewRecorder()
	handler.ListMessages(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListMessages_InvalidRoomID(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages?room_id=not-uuid", nil)
	rr := httptest.NewRecorder()
	handler.ListMessages(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateMessage_MissingRoomID(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(`{"content":"hi"}`))
	req = withClaimsFor(req, uuid.New().String())
	rr := httptest.NewRecorder()
	handler.CreateMessage(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateMessage_EmptyContent(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(`{"room_id":"`+uuid.New().String()+`","content":"","type":"text"}`))
	req = withClaimsFor(req, uuid.New().String())
	rr := httptest.NewRecorder()
	handler.CreateMessage(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateMessage_Unauthorized(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(`{"room_id":"`+uuid.New().String()+`","content":"hi","type":"text"}`))
	rr := httptest.NewRecorder()
	handler.CreateMessage(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestGetThread_InvalidID(t *testing.T) {
	handler := newTestHandler()
	router := chi.NewRouter()
	router.Get("/messages/{id}/thread", handler.GetThread)
	req := httptest.NewRequest(http.MethodGet, "/messages/not-a-uuid/thread", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestMentionRegex(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"Hello @alice and @bob", []string{"alice", "bob"}},
		{"No mentions here", nil},
		{"@single", []string{"single"}},
		{"@alice @alice duplicate", []string{"alice", "alice"}},
		{"@user123 test", []string{"user123"}},
		{"email@example.com", []string{"example"}},
	}
	for _, tt := range tests {
		matches := mentionRegex.FindAllStringSubmatch(tt.input, -1)
		var got []string
		for _, m := range matches {
			got = append(got, m[1])
		}
		assert.Equal(t, tt.expected, got, "input: %s", tt.input)
	}
}

// --- Integration tests (require test DB, skipped if unavailable) ---

func TestCreateMessage_ThreadReply_Integration(t *testing.T) {
	handler, db := setupHandlerWithDB(t)
	userID := uuid.New()
	roomID := uuid.New()
	seedTestUser(t, db, userID)
	seedTestRoom(t, db, roomID, userID)

	rootBody := `{"room_id":"` + roomID.String() + `","content":"Root message","type":"text"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(rootBody))
	req = withClaimsFor(req, userID.String())
	rr := httptest.NewRecorder()
	handler.CreateMessage(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)

	var rootMsg models.Message
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &rootMsg))
	assert.Nil(t, rootMsg.ThreadRootID)

	replyBody := `{"room_id":"` + roomID.String() + `","content":"Thread reply","type":"text","thread_root_id":"` + rootMsg.ID.String() + `"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(replyBody))
	req2 = withClaimsFor(req2, userID.String())
	rr2 := httptest.NewRecorder()
	handler.CreateMessage(rr2, req2)
	require.Equal(t, http.StatusCreated, rr2.Code)

	var replyMsg models.Message
	require.NoError(t, json.Unmarshal(rr2.Body.Bytes(), &replyMsg))
	assert.NotNil(t, replyMsg.ThreadRootID)
	assert.Equal(t, rootMsg.ID, *replyMsg.ThreadRootID)
}

func TestListMessages_ExcludesThreadReplies_Integration(t *testing.T) {
	handler, db := setupHandlerWithDB(t)
	userID := uuid.New()
	roomID := uuid.New()
	seedTestUser(t, db, userID)
	seedTestRoom(t, db, roomID, userID)

	rootBody := `{"room_id":"` + roomID.String() + `","content":"Root","type":"text"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(rootBody))
	req = withClaimsFor(req, userID.String())
	rr := httptest.NewRecorder()
	handler.CreateMessage(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)
	var rootMsg models.Message
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &rootMsg))

	replyBody := `{"room_id":"` + roomID.String() + `","content":"Reply","type":"text","thread_root_id":"` + rootMsg.ID.String() + `"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(replyBody))
	req2 = withClaimsFor(req2, userID.String())
	rr2 := httptest.NewRecorder()
	handler.CreateMessage(rr2, req2)
	require.Equal(t, http.StatusCreated, rr2.Code)

	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/messages?room_id="+roomID.String(), nil)
	rr3 := httptest.NewRecorder()
	handler.ListMessages(rr3, req3)
	require.Equal(t, http.StatusOK, rr3.Code)

	var listResp struct {
		Data []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr3.Body.Bytes(), &listResp))
	assert.Len(t, listResp.Data, 1, "thread replies must be excluded from main feed")
}

func TestListMessages_ThreadCount_Integration(t *testing.T) {
	handler, db := setupHandlerWithDB(t)
	userID := uuid.New()
	roomID := uuid.New()
	seedTestUser(t, db, userID)
	seedTestRoom(t, db, roomID, userID)

	rootBody := `{"room_id":"` + roomID.String() + `","content":"Root","type":"text"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(rootBody))
	req = withClaimsFor(req, userID.String())
	rr := httptest.NewRecorder()
	handler.CreateMessage(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)
	var rootMsg models.Message
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &rootMsg))

	for i := 0; i < 3; i++ {
		replyBody := `{"room_id":"` + roomID.String() + `","content":"Reply","type":"text","thread_root_id":"` + rootMsg.ID.String() + `"}`
		r := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(replyBody))
		r = withClaimsFor(r, userID.String())
		w := httptest.NewRecorder()
		handler.CreateMessage(w, r)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/messages?room_id="+roomID.String(), nil)
	rr2 := httptest.NewRecorder()
	handler.ListMessages(rr2, req2)
	require.Equal(t, http.StatusOK, rr2.Code)

	var listResp struct {
		Data []struct {
			ID          string `json:"id"`
			ThreadCount int64  `json:"thread_count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr2.Body.Bytes(), &listResp))
	require.Len(t, listResp.Data, 1)
	assert.Equal(t, int64(3), listResp.Data[0].ThreadCount)
}

func TestGetThread_Integration(t *testing.T) {
	handler, db := setupHandlerWithDB(t)
	userID := uuid.New()
	roomID := uuid.New()
	seedTestUser(t, db, userID)
	seedTestRoom(t, db, roomID, userID)

	rootBody := `{"room_id":"` + roomID.String() + `","content":"Root msg","type":"text"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(rootBody))
	req = withClaimsFor(req, userID.String())
	rr := httptest.NewRecorder()
	handler.CreateMessage(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)
	var rootMsg models.Message
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &rootMsg))

	for i := 0; i < 2; i++ {
		replyBody := `{"room_id":"` + roomID.String() + `","content":"Reply","type":"text","thread_root_id":"` + rootMsg.ID.String() + `"}`
		r := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(replyBody))
		r = withClaimsFor(r, userID.String())
		w := httptest.NewRecorder()
		handler.CreateMessage(w, r)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	router := chi.NewRouter()
	router.Get("/messages/{id}/thread", handler.GetThread)
	req2 := httptest.NewRequest(http.MethodGet, "/messages/"+rootMsg.ID.String()+"/thread", nil)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)
	require.Equal(t, http.StatusOK, rr2.Code)

	var threadResp struct {
		Root    json.RawMessage   `json:"root"`
		Replies []json.RawMessage `json:"replies"`
		Total   int64             `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rr2.Body.Bytes(), &threadResp))
	assert.Len(t, threadResp.Replies, 2)
	assert.Equal(t, int64(2), threadResp.Total)
}

func TestGetThread_NotFound_Integration(t *testing.T) {
	handler, _ := setupHandlerWithDB(t)

	router := chi.NewRouter()
	router.Get("/messages/{id}/thread", handler.GetThread)
	req := httptest.NewRequest(http.MethodGet, "/messages/"+uuid.New().String()+"/thread", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// --- Этап 4: UpdateMessage / DeleteMessage ---

// Unit-тесты UpdateMessage (без БД, до обращения к репозиторию)

func TestUpdateMessage_InvalidID(t *testing.T) {
	handler := newTestHandler()
	router := chi.NewRouter()
	router.Put("/messages/{id}", handler.UpdateMessage)

	req := httptest.NewRequest(http.MethodPut, "/messages/not-uuid", strings.NewReader(`{"content":"x"}`))
	req = withClaimsFor(req, uuid.New().String())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateMessage_Unauthorized(t *testing.T) {
	handler := newTestHandler()
	router := chi.NewRouter()
	router.Put("/messages/{id}", handler.UpdateMessage)

	req := httptest.NewRequest(http.MethodPut, "/messages/"+uuid.New().String(), strings.NewReader(`{"content":"x"}`))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestUpdateMessage_EmptyContent(t *testing.T) {
	handler := newTestHandler()
	router := chi.NewRouter()
	router.Put("/messages/{id}", handler.UpdateMessage)

	req := httptest.NewRequest(http.MethodPut, "/messages/"+uuid.New().String(), strings.NewReader(`{"content":""}`))
	req = withClaimsFor(req, uuid.New().String())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateMessage_TooLongContent(t *testing.T) {
	handler := newTestHandler()
	router := chi.NewRouter()
	router.Put("/messages/{id}", handler.UpdateMessage)

	long := strings.Repeat("x", 10001)
	req := httptest.NewRequest(http.MethodPut, "/messages/"+uuid.New().String(),
		strings.NewReader(`{"content":"`+long+`"}`))
	req = withClaimsFor(req, uuid.New().String())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// Unit-тесты DeleteMessage (без БД)

func TestDeleteMessage_InvalidID(t *testing.T) {
	handler := newTestHandler()
	router := chi.NewRouter()
	router.Delete("/messages/{id}", handler.DeleteMessage)

	req := httptest.NewRequest(http.MethodDelete, "/messages/not-uuid", nil)
	req = withClaimsFor(req, uuid.New().String())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDeleteMessage_Unauthorized(t *testing.T) {
	handler := newTestHandler()
	router := chi.NewRouter()
	router.Delete("/messages/{id}", handler.DeleteMessage)

	req := httptest.NewRequest(http.MethodDelete, "/messages/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// Integration-тесты UpdateMessage (требуют PostgreSQL)

func TestUpdateMessage_Success_SetsEditedFields_Integration(t *testing.T) {
	handler, db := setupHandlerWithRoomRepo(t)
	userID := uuid.New()
	roomID := uuid.New()
	seedTestUser(t, db, userID)
	seedTestRoom(t, db, roomID, userID)

	original := createMessageInDB(t, db, roomID, userID, "original content", time.Now())

	router := chi.NewRouter()
	router.Put("/messages/{id}", handler.UpdateMessage)

	body := `{"content":"updated content"}`
	req := httptest.NewRequest(http.MethodPut, "/messages/"+original.ID.String(), strings.NewReader(body))
	req = withClaimsFor(req, userID.String())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var updated models.Message
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &updated))
	assert.Equal(t, "updated content", updated.Content)
	require.NotNil(t, updated.Metadata.Edited)
	assert.True(t, *updated.Metadata.Edited)
	require.NotNil(t, updated.Metadata.EditedAt)
	require.NotNil(t, updated.Metadata.EditedBy)
	assert.Equal(t, userID, *updated.Metadata.EditedBy)
}

func TestUpdateMessage_NotAuthor_Returns403_Integration(t *testing.T) {
	handler, db := setupHandlerWithRoomRepo(t)
	authorID := uuid.New()
	otherID := uuid.New()
	roomID := uuid.New()
	seedTestUser(t, db, authorID)
	seedTestUser(t, db, otherID)
	seedTestRoom(t, db, roomID, authorID)

	original := createMessageInDB(t, db, roomID, authorID, "original", time.Now())

	router := chi.NewRouter()
	router.Put("/messages/{id}", handler.UpdateMessage)

	req := httptest.NewRequest(http.MethodPut, "/messages/"+original.ID.String(), strings.NewReader(`{"content":"hijack"}`))
	req = withClaimsFor(req, otherID.String())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestUpdateMessage_NotFound_Integration(t *testing.T) {
	handler, db := setupHandlerWithRoomRepo(t)
	userID := uuid.New()
	seedTestUser(t, db, userID)

	router := chi.NewRouter()
	router.Put("/messages/{id}", handler.UpdateMessage)

	req := httptest.NewRequest(http.MethodPut, "/messages/"+uuid.New().String(), strings.NewReader(`{"content":"x"}`))
	req = withClaimsFor(req, userID.String())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestUpdateMessage_EditWindowExpired_Returns410_Integration(t *testing.T) {
	handler, db := setupHandlerWithRoomRepo(t)
	handler.SetEditWindow(50 * time.Millisecond)
	userID := uuid.New()
	roomID := uuid.New()
	seedTestUser(t, db, userID)
	seedTestRoom(t, db, roomID, userID)

	old := createMessageInDB(t, db, roomID, userID, "old content", time.Now().Add(-1*time.Hour))

	router := chi.NewRouter()
	router.Put("/messages/{id}", handler.UpdateMessage)

	req := httptest.NewRequest(http.MethodPut, "/messages/"+old.ID.String(), strings.NewReader(`{"content":"too late"}`))
	req = withClaimsFor(req, userID.String())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusGone, rr.Code)
}

func TestUpdateMessage_NoLimitWhenWindowZero_Integration(t *testing.T) {
	handler, db := setupHandlerWithRoomRepo(t)
	handler.SetEditWindow(0)
	userID := uuid.New()
	roomID := uuid.New()
	seedTestUser(t, db, userID)
	seedTestRoom(t, db, roomID, userID)

	old := createMessageInDB(t, db, roomID, userID, "old content", time.Now().Add(-72*time.Hour))

	router := chi.NewRouter()
	router.Put("/messages/{id}", handler.UpdateMessage)

	req := httptest.NewRequest(http.MethodPut, "/messages/"+old.ID.String(), strings.NewReader(`{"content":"still editable"}`))
	req = withClaimsFor(req, userID.String())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// Integration-тесты DeleteMessage (требуют PostgreSQL)

func TestDeleteMessage_Author_Success_Integration(t *testing.T) {
	handler, db := setupHandlerWithRoomRepo(t)
	userID := uuid.New()
	roomID := uuid.New()
	seedTestUser(t, db, userID)
	seedTestRoom(t, db, roomID, userID)

	msg := createMessageInDB(t, db, roomID, userID, "to delete", time.Now())

	router := chi.NewRouter()
	router.Delete("/messages/{id}", handler.DeleteMessage)

	req := httptest.NewRequest(http.MethodDelete, "/messages/"+msg.ID.String(), nil)
	req = withClaimsFor(req, userID.String())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNoContent, rr.Code)

	var reloaded models.Message
	require.NoError(t, db.First(&reloaded, "id = ?", msg.ID).Error)
	assert.True(t, reloaded.IsDeleted)
}

func TestDeleteMessage_GlobalAdmin_DeletesOthers_Integration(t *testing.T) {
	handler, db := setupHandlerWithRoomRepo(t)
	authorID := uuid.New()
	adminID := uuid.New()
	roomID := uuid.New()
	seedTestUser(t, db, authorID)
	seedTestUser(t, db, adminID)
	seedTestRoom(t, db, roomID, authorID)

	msg := createMessageInDB(t, db, roomID, authorID, "any user msg", time.Now())

	router := chi.NewRouter()
	router.Delete("/messages/{id}", handler.DeleteMessage)

	req := httptest.NewRequest(http.MethodDelete, "/messages/"+msg.ID.String(), nil)
	req = withClaimsForRoles(req, adminID.String(), []string{"user", "admin"})
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNoContent, rr.Code)

	var reloaded models.Message
	require.NoError(t, db.First(&reloaded, "id = ?", msg.ID).Error)
	assert.True(t, reloaded.IsDeleted)
}

func TestDeleteMessage_RoomModerator_DeletesOthers_Integration(t *testing.T) {
	handler, db := setupHandlerWithRoomRepo(t)
	authorID := uuid.New()
	moderatorID := uuid.New()
	roomID := uuid.New()
	seedTestUser(t, db, authorID)
	seedTestUser(t, db, moderatorID)
	seedTestRoom(t, db, roomID, authorID)
	addRoomParticipant(t, db, roomID, moderatorID, models.ParticipantRoleModerator)

	msg := createMessageInDB(t, db, roomID, authorID, "to be moderated", time.Now())

	router := chi.NewRouter()
	router.Delete("/messages/{id}", handler.DeleteMessage)

	req := httptest.NewRequest(http.MethodDelete, "/messages/"+msg.ID.String(), nil)
	req = withClaimsFor(req, moderatorID.String())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNoContent, rr.Code)

	var reloaded models.Message
	require.NoError(t, db.First(&reloaded, "id = ?", msg.ID).Error)
	assert.True(t, reloaded.IsDeleted)
}

func TestDeleteMessage_RoomAdmin_DeletesOthers_Integration(t *testing.T) {
	handler, db := setupHandlerWithRoomRepo(t)
	authorID := uuid.New()
	adminInRoomID := uuid.New()
	roomID := uuid.New()
	seedTestUser(t, db, authorID)
	seedTestUser(t, db, adminInRoomID)
	seedTestRoom(t, db, roomID, authorID)
	addRoomParticipant(t, db, roomID, adminInRoomID, models.ParticipantRoleAdmin)

	msg := createMessageInDB(t, db, roomID, authorID, "to be admin-deleted", time.Now())

	router := chi.NewRouter()
	router.Delete("/messages/{id}", handler.DeleteMessage)

	req := httptest.NewRequest(http.MethodDelete, "/messages/"+msg.ID.String(), nil)
	req = withClaimsFor(req, adminInRoomID.String())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestDeleteMessage_RegularUser_Forbidden_Integration(t *testing.T) {
	handler, db := setupHandlerWithRoomRepo(t)
	authorID := uuid.New()
	memberID := uuid.New()
	roomID := uuid.New()
	seedTestUser(t, db, authorID)
	seedTestUser(t, db, memberID)
	seedTestRoom(t, db, roomID, authorID)
	addRoomParticipant(t, db, roomID, memberID, models.ParticipantRoleMember)

	msg := createMessageInDB(t, db, roomID, authorID, "members can't delete this", time.Now())

	router := chi.NewRouter()
	router.Delete("/messages/{id}", handler.DeleteMessage)

	req := httptest.NewRequest(http.MethodDelete, "/messages/"+msg.ID.String(), nil)
	req = withClaimsFor(req, memberID.String())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)

	var reloaded models.Message
	require.NoError(t, db.First(&reloaded, "id = ?", msg.ID).Error)
	assert.False(t, reloaded.IsDeleted, "regular member must not be able to delete others' messages")
}

func TestDeleteMessage_NotFound_Integration(t *testing.T) {
	handler, db := setupHandlerWithRoomRepo(t)
	userID := uuid.New()
	seedTestUser(t, db, userID)

	router := chi.NewRouter()
	router.Delete("/messages/{id}", handler.DeleteMessage)

	req := httptest.NewRequest(http.MethodDelete, "/messages/"+uuid.New().String(), nil)
	req = withClaimsFor(req, userID.String())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDeleteMessage_AlreadyDeleted_Idempotent_Integration(t *testing.T) {
	handler, db := setupHandlerWithRoomRepo(t)
	userID := uuid.New()
	roomID := uuid.New()
	seedTestUser(t, db, userID)
	seedTestRoom(t, db, roomID, userID)

	msg := createMessageInDB(t, db, roomID, userID, "soft-deleted already", time.Now())
	require.NoError(t, db.Model(&models.Message{}).Where("id = ?", msg.ID).Update("is_deleted", true).Error)

	router := chi.NewRouter()
	router.Delete("/messages/{id}", handler.DeleteMessage)

	req := httptest.NewRequest(http.MethodDelete, "/messages/"+msg.ID.String(), nil)
	req = withClaimsFor(req, userID.String())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNoContent, rr.Code)
}
