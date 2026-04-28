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
	wsHub := websocket.NewHub(zap.NewNop())
	go wsHub.Run()
	return NewMessageHandler(msgRepo, wsHub, nil), db
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
	ctx := context.WithValue(req.Context(), auth.ContextKeyUserClaims, &auth.SessionClaims{
		UserID: userID, Email: "test@test-thread.com", Name: "Test", Roles: []string{"user"},
	})
	return req.WithContext(ctx)
}

// --- Unit tests (no DB required) ---

func TestListMessages_MissingRoomID(t *testing.T) {
	wsHub := websocket.NewHub(zap.NewNop())
	go wsHub.Run()
	handler := NewMessageHandler(nil, wsHub, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	rr := httptest.NewRecorder()
	handler.ListMessages(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListMessages_InvalidRoomID(t *testing.T) {
	wsHub := websocket.NewHub(zap.NewNop())
	go wsHub.Run()
	handler := NewMessageHandler(nil, wsHub, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages?room_id=not-uuid", nil)
	rr := httptest.NewRecorder()
	handler.ListMessages(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateMessage_MissingRoomID(t *testing.T) {
	wsHub := websocket.NewHub(zap.NewNop())
	go wsHub.Run()
	handler := NewMessageHandler(nil, wsHub, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(`{"content":"hi"}`))
	req = withClaimsFor(req, uuid.New().String())
	rr := httptest.NewRecorder()
	handler.CreateMessage(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateMessage_EmptyContent(t *testing.T) {
	wsHub := websocket.NewHub(zap.NewNop())
	go wsHub.Run()
	handler := NewMessageHandler(nil, wsHub, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(`{"room_id":"`+uuid.New().String()+`","content":"","type":"text"}`))
	req = withClaimsFor(req, uuid.New().String())
	rr := httptest.NewRecorder()
	handler.CreateMessage(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateMessage_Unauthorized(t *testing.T) {
	wsHub := websocket.NewHub(zap.NewNop())
	go wsHub.Run()
	handler := NewMessageHandler(nil, wsHub, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(`{"room_id":"`+uuid.New().String()+`","content":"hi","type":"text"}`))
	rr := httptest.NewRecorder()
	handler.CreateMessage(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestGetThread_InvalidID(t *testing.T) {
	wsHub := websocket.NewHub(zap.NewNop())
	go wsHub.Run()
	handler := NewMessageHandler(nil, wsHub, nil)

	router := chi.NewRouter()
	router.Get("/messages/{id}/thread", handler.GetThread)

	req := httptest.NewRequest(http.MethodGet, "/messages/not-a-uuid/thread", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
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
