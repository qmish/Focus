package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubProvider — реализация search.SearchProvider, удобная для assert'ов.
type stubProvider struct {
	mu sync.Mutex

	users    []*models.User
	rooms    []*models.Room
	messages []*search.MessageHit
	files    []*search.FileHit
	meetings []*search.MeetingHit

	gotLimitMessages int
	gotRoomID        *uuid.UUID
	gotBefore        *uuid.UUID
}

func (s *stubProvider) SearchUsers(_ context.Context, _ string, _ int) ([]*models.User, error) {
	return s.users, nil
}
func (s *stubProvider) SearchRooms(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*models.Room, error) {
	return s.rooms, nil
}
func (s *stubProvider) SearchMessages(_ context.Context, _ uuid.UUID, _ string, roomID *uuid.UUID, opts search.MessageSearchOpts) ([]*search.MessageHit, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gotLimitMessages = opts.Limit
	if roomID != nil {
		copy := *roomID
		s.gotRoomID = &copy
	}
	if opts.Before != nil {
		copy := *opts.Before
		s.gotBefore = &copy
	}
	return s.messages, nil
}
func (s *stubProvider) SearchFiles(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*search.FileHit, error) {
	return s.files, nil
}
func (s *stubProvider) SearchMeetings(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*search.MeetingHit, error) {
	return s.meetings, nil
}

func newSearchHandler(p search.SearchProvider) *SearchHandler {
	return NewSearchHandler(search.NewService(p))
}

func ctxWithUser(uid uuid.UUID) context.Context {
	return context.WithValue(context.Background(), auth.ContextKeyUserClaims, &auth.SessionClaims{UserID: uid.String()})
}

func TestSearchHandler_Global_Unauthorized(t *testing.T) {
	h := newSearchHandler(&stubProvider{})
	req := httptest.NewRequest(http.MethodGet, "/search?q=hello", nil)
	rr := httptest.NewRecorder()
	h.Global(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestSearchHandler_Global_BadQuery(t *testing.T) {
	h := newSearchHandler(&stubProvider{})
	cases := []string{"", "a", "+", "%20"}
	for _, q := range cases {
		req := httptest.NewRequest(http.MethodGet, "/search?q="+q, nil)
		req = req.WithContext(ctxWithUser(uuid.New()))
		rr := httptest.NewRecorder()
		h.Global(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("q=%q: expected 400, got %d (body=%s)", q, rr.Code, rr.Body.String())
		}
	}
}

func TestSearchHandler_Global_OK_AllTypesByDefault(t *testing.T) {
	stub := &stubProvider{
		users:    []*models.User{{ID: uuid.New(), Name: "Alice"}},
		rooms:    []*models.Room{{ID: uuid.New(), Name: "general"}},
		messages: []*search.MessageHit{{RoomID: uuid.New(), Highlight: "<mark>hi</mark>"}},
		files:    []*search.FileHit{{FileName: "report.pdf"}},
		meetings: []*search.MeetingHit{{Subject: "Standup"}},
	}
	h := newSearchHandler(stub)

	req := httptest.NewRequest(http.MethodGet, "/search?q=report", nil)
	req = req.WithContext(ctxWithUser(uuid.New()))
	rr := httptest.NewRecorder()
	h.Global(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var body GlobalResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	assert.Len(t, body.Users, 1)
	assert.Len(t, body.Rooms, 1)
	assert.Len(t, body.Messages, 1)
	assert.Len(t, body.Files, 1)
	assert.Len(t, body.Meetings, 1)
	assert.Equal(t, "report", body.Query)
}

func TestSearchHandler_Global_TypesFilter(t *testing.T) {
	stub := &stubProvider{
		users:    []*models.User{{ID: uuid.New(), Name: "Alice"}},
		rooms:    []*models.Room{{ID: uuid.New(), Name: "general"}},
		messages: []*search.MessageHit{{RoomID: uuid.New()}},
	}
	h := newSearchHandler(stub)

	req := httptest.NewRequest(http.MethodGet, "/search?q=alice&types=users,messages", nil)
	req = req.WithContext(ctxWithUser(uuid.New()))
	rr := httptest.NewRecorder()
	h.Global(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var body GlobalResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	assert.Len(t, body.Users, 1)
	assert.Len(t, body.Messages, 1)
	assert.Empty(t, body.Rooms, "rooms not requested")
	assert.Empty(t, body.Files)
	assert.Empty(t, body.Meetings)
}

func TestSearchHandler_LocalMessages_Unauthorized(t *testing.T) {
	h := newSearchHandler(&stubProvider{})
	req := httptest.NewRequest(http.MethodGet, "/rooms/"+uuid.NewString()+"/messages/search?q=hi", nil)
	rr := httptest.NewRecorder()
	h.LocalMessages(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestSearchHandler_LocalMessages_BadRoomID(t *testing.T) {
	h := newSearchHandler(&stubProvider{})

	r := chi.NewRouter()
	r.Get("/rooms/{id}/messages/search", h.LocalMessages)

	req := httptest.NewRequest(http.MethodGet, "/rooms/not-a-uuid/messages/search?q=hello", nil)
	req = req.WithContext(ctxWithUser(uuid.New()))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, strings.ToLower(rr.Body.String()), "комнат")
}

func TestSearchHandler_LocalMessages_OK_PassesParamsToProvider(t *testing.T) {
	roomID := uuid.New()
	beforeID := uuid.New()
	hitMsg := &models.Message{ID: uuid.New(), RoomID: roomID, Content: "hello"}
	stub := &stubProvider{
		messages: []*search.MessageHit{
			{Message: hitMsg, RoomID: roomID, RoomName: "general", Highlight: "<mark>hello</mark>"},
		},
	}
	h := newSearchHandler(stub)

	r := chi.NewRouter()
	r.Get("/rooms/{id}/messages/search", h.LocalMessages)

	uid := uuid.New()
	url := "/rooms/" + roomID.String() + "/messages/search?q=hello&before=" + beforeID.String() + "&limit=5"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req = req.WithContext(ctxWithUser(uid))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var body LocalMessagesResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	assert.Len(t, body.Messages, 1)
	assert.Equal(t, "hello", body.Query)
	if assert.NotNil(t, stub.gotRoomID) {
		assert.Equal(t, roomID, *stub.gotRoomID)
	}
	if assert.NotNil(t, stub.gotBefore) {
		assert.Equal(t, beforeID, *stub.gotBefore)
	}
	assert.Equal(t, 5, stub.gotLimitMessages, "limit should be passed through")
}

func TestSearchHandler_LocalMessages_LimitClamped(t *testing.T) {
	roomID := uuid.New()
	stub := &stubProvider{messages: nil}
	h := newSearchHandler(stub)

	r := chi.NewRouter()
	r.Get("/rooms/{id}/messages/search", h.LocalMessages)

	url := "/rooms/" + roomID.String() + "/messages/search?q=hello&limit=999"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req = req.WithContext(ctxWithUser(uuid.New()))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 100, stub.gotLimitMessages, "limit > 100 should clamp to 100")
}

func TestParseScope(t *testing.T) {
	cases := map[string]search.Scope{
		"":                 search.DefaultScope(),
		"users":            {Users: true},
		"users,messages":   {Users: true, Messages: true},
		"unknown":          search.DefaultScope(),
		"chats,people":     {Rooms: true, Users: true},
		"files,meetings":   {Files: true, Meetings: true},
		"  USERS , Rooms ": {Users: true, Rooms: true},
	}
	for raw, want := range cases {
		got := parseScope(raw)
		if got != want {
			t.Errorf("parseScope(%q) = %+v, want %+v", raw, got, want)
		}
	}
}
