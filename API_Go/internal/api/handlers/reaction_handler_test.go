package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
	"github.com/qmish/focus-api/internal/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newReactionHandler() *ReactionHandler {
	wsHub := websocket.NewHub(zap.NewNop())
	go wsHub.Run()
	return NewReactionHandler(nil, wsHub)
}

func TestAddReaction_InvalidMessageID(t *testing.T) {
	handler := newReactionHandler()
	router := chi.NewRouter()
	router.Post("/messages/{id}/reactions", handler.AddReaction)

	req := httptest.NewRequest(http.MethodPost, "/messages/not-uuid/reactions", strings.NewReader(`{"emoji":"👍"}`))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAddReaction_MissingEmoji(t *testing.T) {
	handler := newReactionHandler()
	router := chi.NewRouter()
	router.Post("/messages/{id}/reactions", handler.AddReaction)

	req := httptest.NewRequest(http.MethodPost, "/messages/"+uuid.New().String()+"/reactions", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAddReaction_Unauthorized(t *testing.T) {
	handler := newReactionHandler()
	router := chi.NewRouter()
	router.Post("/messages/{id}/reactions", handler.AddReaction)

	req := httptest.NewRequest(http.MethodPost, "/messages/"+uuid.New().String()+"/reactions", strings.NewReader(`{"emoji":"👍"}`))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRemoveReaction_InvalidMessageID(t *testing.T) {
	handler := newReactionHandler()
	router := chi.NewRouter()
	router.Delete("/messages/{id}/reactions/{emoji}", handler.RemoveReaction)

	req := httptest.NewRequest(http.MethodDelete, "/messages/not-uuid/reactions/👍", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRemoveReaction_Unauthorized(t *testing.T) {
	handler := newReactionHandler()
	router := chi.NewRouter()
	router.Delete("/messages/{id}/reactions/{emoji}", handler.RemoveReaction)

	req := httptest.NewRequest(http.MethodDelete, "/messages/"+uuid.New().String()+"/reactions/👍", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestListReactions_InvalidMessageID(t *testing.T) {
	handler := newReactionHandler()
	router := chi.NewRouter()
	router.Get("/messages/{id}/reactions", handler.ListReactions)

	req := httptest.NewRequest(http.MethodGet, "/messages/not-uuid/reactions", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAggregateReactions(t *testing.T) {
	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()
	msgID := uuid.New()

	reactions := []models.MessageReaction{
		{ID: uuid.New(), MessageID: msgID, UserID: user1, Emoji: "👍"},
		{ID: uuid.New(), MessageID: msgID, UserID: user2, Emoji: "👍"},
		{ID: uuid.New(), MessageID: msgID, UserID: user3, Emoji: "❤️"},
		{ID: uuid.New(), MessageID: msgID, UserID: user1, Emoji: "❤️"},
	}

	summary := AggregateReactions(reactions)
	require.Len(t, summary, 2)

	assert.Equal(t, "👍", summary[0].Emoji)
	assert.Equal(t, 2, summary[0].Count)
	assert.Len(t, summary[0].UserIDs, 2)

	assert.Equal(t, "❤️", summary[1].Emoji)
	assert.Equal(t, 2, summary[1].Count)
	assert.Len(t, summary[1].UserIDs, 2)
}

func TestAggregateReactions_Empty(t *testing.T) {
	summary := AggregateReactions([]models.MessageReaction{})
	assert.Empty(t, summary)
}

func TestAggregateReactions_SingleEmoji(t *testing.T) {
	msgID := uuid.New()
	reactions := []models.MessageReaction{
		{ID: uuid.New(), MessageID: msgID, UserID: uuid.New(), Emoji: "🔥"},
	}
	summary := AggregateReactions(reactions)
	require.Len(t, summary, 1)
	assert.Equal(t, "🔥", summary[0].Emoji)
	assert.Equal(t, 1, summary[0].Count)
}

func TestAddReaction_Integration(t *testing.T) {
	db := getTestDB(t)
	msgRepo := repository.NewMessageRepository(db)
	wsHub := websocket.NewHub(zap.NewNop())
	go wsHub.Run()
	handler := NewReactionHandler(msgRepo, wsHub)
	msgHandler := NewMessageHandler(msgRepo, repository.NewUserRepository(db), wsHub, nil)

	userID := uuid.New()
	roomID := uuid.New()
	seedTestUser(t, db, userID)
	seedTestRoom(t, db, roomID, userID)

	body := `{"room_id":"` + roomID.String() + `","content":"Test msg","type":"text"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(body))
	req = withClaimsFor(req, userID.String())
	rr := httptest.NewRecorder()
	msgHandler.CreateMessage(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)
	var msg models.Message
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &msg))

	router := chi.NewRouter()
	router.Post("/messages/{id}/reactions", handler.AddReaction)
	router.Get("/messages/{id}/reactions", handler.ListReactions)
	router.Delete("/messages/{id}/reactions/{emoji}", handler.RemoveReaction)

	addReq := httptest.NewRequest(http.MethodPost, "/messages/"+msg.ID.String()+"/reactions", strings.NewReader(`{"emoji":"👍"}`))
	addReq = withClaimsFor(addReq, userID.String())
	addRR := httptest.NewRecorder()
	router.ServeHTTP(addRR, addReq)
	require.Equal(t, http.StatusCreated, addRR.Code)

	listReq := httptest.NewRequest(http.MethodGet, "/messages/"+msg.ID.String()+"/reactions", nil)
	listRR := httptest.NewRecorder()
	router.ServeHTTP(listRR, listReq)
	require.Equal(t, http.StatusOK, listRR.Code)

	var summaries []ReactionSummary
	require.NoError(t, json.Unmarshal(listRR.Body.Bytes(), &summaries))
	require.Len(t, summaries, 1)
	assert.Equal(t, "👍", summaries[0].Emoji)
	assert.Equal(t, 1, summaries[0].Count)

	delReq := httptest.NewRequest(http.MethodDelete, "/messages/"+msg.ID.String()+"/reactions/👍", nil)
	delReq = withClaimsFor(delReq, userID.String())
	delRR := httptest.NewRecorder()
	router.ServeHTTP(delRR, delReq)
	assert.Equal(t, http.StatusNoContent, delRR.Code)

	listReq2 := httptest.NewRequest(http.MethodGet, "/messages/"+msg.ID.String()+"/reactions", nil)
	listRR2 := httptest.NewRecorder()
	router.ServeHTTP(listRR2, listReq2)
	require.Equal(t, http.StatusOK, listRR2.Code)
	var summaries2 []ReactionSummary
	require.NoError(t, json.Unmarshal(listRR2.Body.Bytes(), &summaries2))
	assert.Empty(t, summaries2)
}
