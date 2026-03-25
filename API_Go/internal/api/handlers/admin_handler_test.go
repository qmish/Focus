package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/bots"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
	"github.com/qmish/focus-api/internal/webhooks"
	"github.com/stretchr/testify/assert"
)

func TestHasRole(t *testing.T) {
	claims := &auth.SessionClaims{
		Roles: []string{"user", "moderator"},
	}

	assert.True(t, hasRole(claims, "user"))
	assert.True(t, hasRole(claims, "moderator"))
	assert.False(t, hasRole(claims, "admin"))
}

func TestRequireAdmin(t *testing.T) {
	t.Run("user with admin role", func(t *testing.T) {
		claims := &auth.SessionClaims{
			Roles: []string{"admin"},
		}

		ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
		rr := httptest.NewRecorder()

		handler := requireAdmin(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("user without admin role", func(t *testing.T) {
		claims := &auth.SessionClaims{
			Roles: []string{"user"},
		}

		ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
		rr := httptest.NewRecorder()

		handler := requireAdmin(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("no claims in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		handler := requireAdmin(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestAdminHandler(t *testing.T) {
	handler := NewAdminHandler(nil, nil)

	assert.NotNil(t, handler)
}

func TestAdminHandlerListUsersForbidden(t *testing.T) {
	handler := NewAdminHandler(nil, nil)

	// Создаём запрос без admin роли
	claims := &auth.SessionClaims{
		Roles: []string{"user"},
	}

	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/users", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ListUsers(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAdminHandlerGetUserInvalidID(t *testing.T) {
	handler := NewAdminHandler(nil, nil)

	claims := &auth.SessionClaims{
		Roles: []string{"admin"},
	}

	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := addURLParam(
		httptest.NewRequest("GET", "/api/v1/admin/users/invalid-id", nil).WithContext(ctx),
		"id",
		"invalid-id",
	)
	rr := httptest.NewRecorder()

	handler.GetUser(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAdminHandlerGetUserForbidden(t *testing.T) {
	handler := NewAdminHandler(nil, nil)

	claims := &auth.SessionClaims{
		Roles: []string{"user"},
	}

	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := addURLParam(
		httptest.NewRequest("GET", "/api/v1/admin/users/"+uuid.New().String(), nil).WithContext(ctx),
		"id",
		uuid.New().String(),
	)
	rr := httptest.NewRecorder()

	handler.GetUser(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAdminHandlerListConferences(t *testing.T) {
	room1 := models.NewRoom("Meeting A", uuid.New(), models.RoomTypeMeeting)
	room1.CreatedAt = time.Now().Add(-1 * time.Hour)
	room1.UpdatedAt = time.Now()
	room2 := models.NewRoom("General", uuid.New(), models.RoomTypePublic)
	handler := NewAdminHandler(nil, &fakeAdminRoomRepo{
		rooms:              map[uuid.UUID]*models.Room{room1.ID: room1, room2.ID: room2},
		participantByRoom:  map[uuid.UUID]int64{room1.ID: 5},
	})

	claims := &auth.SessionClaims{
		Roles: []string{"admin"},
	}

	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/conferences", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ListConferences(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"room_name":"Meeting A"`)
	assert.NotContains(t, rr.Body.String(), `"room_name":"General"`)
}

func TestAdminHandlerEndConference(t *testing.T) {
	room := models.NewRoom("Meeting A", uuid.New(), models.RoomTypeMeeting)
	repo := &fakeAdminRoomRepo{
		rooms: map[uuid.UUID]*models.Room{room.ID: room},
	}
	handler := NewAdminHandler(nil, repo)

	claims := &auth.SessionClaims{
		Roles: []string{"admin"},
	}

	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := addURLParam(
		httptest.NewRequest("POST", "/api/v1/admin/conferences/"+room.ID.String()+"/end", nil).WithContext(ctx),
		"id",
		room.ID.String(),
	)
	rr := httptest.NewRecorder()

	handler.EndConference(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"ended":true`)
	assert.True(t, repo.deleted[room.ID])
}

func TestAdminHandlerEndConferenceNotFound(t *testing.T) {
	handler := NewAdminHandler(nil, &fakeAdminRoomRepo{
		rooms: map[uuid.UUID]*models.Room{},
	})

	claims := &auth.SessionClaims{
		Roles: []string{"admin"},
	}

	missingID := uuid.New()
	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := addURLParam(
		httptest.NewRequest("POST", "/api/v1/admin/conferences/"+missingID.String()+"/end", nil).WithContext(ctx),
		"id",
		missingID.String(),
	)
	rr := httptest.NewRecorder()

	handler.EndConference(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestAdminHandlerGetStats(t *testing.T) {
	handler := NewAdminHandler(&fakeAdminUserRepo{count: 3}, &fakeAdminRoomRepo{count: 7})

	claims := &auth.SessionClaims{
		Roles: []string{"admin"},
	}

	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/stats", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetStats(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"total":3`)
	assert.Contains(t, rr.Body.String(), `"total":7`)
}

func TestAdminHandlerGetStatsForbidden(t *testing.T) {
	handler := NewAdminHandler(nil, nil)

	claims := &auth.SessionClaims{
		Roles: []string{"user"},
	}

	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/stats", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetStats(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

type fakeAdminUserRepo struct {
	count int64
}

func (f *fakeAdminUserRepo) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	return []*models.User{}, nil
}
func (f *fakeAdminUserRepo) Count(ctx context.Context) (int64, error) {
	return f.count, nil
}
func (f *fakeAdminUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return nil, repository.ErrUserNotFound
}
func (f *fakeAdminUserRepo) Update(ctx context.Context, user *models.User) error {
	return nil
}

type fakeAdminRoomRepo struct {
	rooms             map[uuid.UUID]*models.Room
	deleted           map[uuid.UUID]bool
	count             int64
	participantByRoom map[uuid.UUID]int64
}

type fakeAdminWebhookRepo struct {
	deliveries []*webhooks.WebhookDelivery
	err        error
}

type fakeAdminBotRepo struct {
	events []*bots.BotCommandEvent
	err    error
}

func (f *fakeAdminRoomRepo) List(ctx context.Context, limit, offset int) ([]*models.Room, error) {
	rooms := make([]*models.Room, 0, len(f.rooms))
	for _, room := range f.rooms {
		if room != nil && room.DeletedAt == nil {
			rooms = append(rooms, room)
		}
	}
	return rooms, nil
}
func (f *fakeAdminRoomRepo) Count(ctx context.Context) (int64, error) {
	if f.count != 0 {
		return f.count, nil
	}
	return int64(len(f.rooms)), nil
}
func (f *fakeAdminRoomRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Room, error) {
	room, ok := f.rooms[id]
	if !ok || room == nil || room.DeletedAt != nil {
		return nil, repository.ErrRoomNotFound
	}
	return room, nil
}
func (f *fakeAdminRoomRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if f.deleted == nil {
		f.deleted = map[uuid.UUID]bool{}
	}
	room, ok := f.rooms[id]
	if !ok || room == nil {
		return repository.ErrRoomNotFound
	}
	now := time.Now()
	room.DeletedAt = &now
	f.deleted[id] = true
	return nil
}
func (f *fakeAdminRoomRepo) CountParticipants(ctx context.Context, roomID uuid.UUID) (int64, error) {
	if f.participantByRoom == nil {
		return 0, nil
	}
	return f.participantByRoom[roomID], nil
}

func (f *fakeAdminWebhookRepo) ListRecentDeliveries(ctx context.Context, limit int, onlyFailed bool) ([]*webhooks.WebhookDelivery, error) {
	if f.err != nil {
		return nil, f.err
	}
	if !onlyFailed {
		return f.deliveries, nil
	}
	filtered := make([]*webhooks.WebhookDelivery, 0, len(f.deliveries))
	for _, delivery := range f.deliveries {
		if delivery != nil && !delivery.Success {
			filtered = append(filtered, delivery)
		}
	}
	return filtered, nil
}

func (f *fakeAdminBotRepo) ListCommandEvents(ctx context.Context, limit int, onlyFailed bool) ([]*bots.BotCommandEvent, error) {
	if f.err != nil {
		return nil, f.err
	}
	if !onlyFailed {
		return f.events, nil
	}
	filtered := make([]*bots.BotCommandEvent, 0, len(f.events))
	for _, event := range f.events {
		if event == nil {
			continue
		}
		if event.Status == "failed" || event.Status == "permission_denied" || event.Status == "rate_limited" {
			filtered = append(filtered, event)
		}
	}
	return filtered, nil
}

func addURLParam(req *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func TestAdminHandlerListConferencesUnauthorized(t *testing.T) {
	handler := NewAdminHandler(nil, nil)
	req := httptest.NewRequest("GET", "/api/v1/admin/conferences", nil)
	rr := httptest.NewRecorder()
	handler.ListConferences(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAdminHandlerEndConferenceInvalidID(t *testing.T) {
	handler := NewAdminHandler(nil, nil)
	claims := &auth.SessionClaims{Roles: []string{"admin"}}
	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := addURLParam(httptest.NewRequest("POST", "/api/v1/admin/conferences/invalid/end", nil).WithContext(ctx), "id", "invalid")
	rr := httptest.NewRecorder()
	handler.EndConference(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAdminHandlerListConferencesResponseSchema(t *testing.T) {
	room := models.NewRoom("Meeting A", uuid.New(), models.RoomTypeMeeting)
	handler := NewAdminHandler(nil, &fakeAdminRoomRepo{
		rooms:            map[uuid.UUID]*models.Room{room.ID: room},
		participantByRoom: map[uuid.UUID]int64{room.ID: 2},
	})
	claims := &auth.SessionClaims{Roles: []string{"admin"}}
	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/conferences", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ListConferences(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var payload map[string][]map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &payload)
	assert.NoError(t, err)
	assert.NotEmpty(t, payload["data"])
	assert.Equal(t, "active", payload["data"][0]["status"])
}

func TestAdminHandlerListWebhookDeliveries(t *testing.T) {
	repo := &fakeAdminWebhookRepo{
		deliveries: []*webhooks.WebhookDelivery{
			{
				ID:           uuid.New(),
				WebhookID:    uuid.New(),
				ResponseCode: 200,
				Success:      true,
				RetryCount:   0,
				CreatedAt:    time.Now(),
			},
		},
	}
	handler := NewAdminHandler(nil, nil)
	handler.SetWebhookRepository(repo)

	claims := &auth.SessionClaims{Roles: []string{"admin"}}
	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/webhooks/deliveries?limit=10", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ListWebhookDeliveries(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"success":true`)
}

func TestAdminHandlerListWebhookErrors(t *testing.T) {
	repo := &fakeAdminWebhookRepo{
		deliveries: []*webhooks.WebhookDelivery{
			{
				ID:           uuid.New(),
				WebhookID:    uuid.New(),
				ResponseCode: 500,
				ResponseBody: "dead_letter: status=500",
				Success:      false,
				RetryCount:   2,
				CreatedAt:    time.Now(),
			},
		},
	}
	handler := NewAdminHandler(nil, nil)
	handler.SetWebhookRepository(repo)

	claims := &auth.SessionClaims{Roles: []string{"admin"}}
	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/webhooks/errors", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ListWebhookErrors(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"total":1`)
	assert.Contains(t, rr.Body.String(), `"success":false`)
}

func TestAdminHandlerListWebhookErrorsForbidden(t *testing.T) {
	handler := NewAdminHandler(nil, nil)
	claims := &auth.SessionClaims{Roles: []string{"user"}}
	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/webhooks/errors", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ListWebhookErrors(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAdminHandlerListWebhookDeliveriesInternalError(t *testing.T) {
	repo := &fakeAdminWebhookRepo{err: errors.New("db down")}
	handler := NewAdminHandler(nil, nil)
	handler.SetWebhookRepository(repo)
	claims := &auth.SessionClaims{Roles: []string{"admin"}}
	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/webhooks/deliveries", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ListWebhookDeliveries(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestAdminHandlerListBotErrors(t *testing.T) {
	repo := &fakeAdminBotRepo{
		events: []*bots.BotCommandEvent{
			{
				ID:      uuid.New(),
				Command: "schedule",
				Status:  "failed",
				Error:   "calendar unavailable",
			},
		},
	}
	handler := NewAdminHandler(nil, nil)
	handler.SetBotRepository(repo)
	claims := &auth.SessionClaims{Roles: []string{"admin"}}
	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/bots/errors", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ListBotErrors(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"total":1`)
	assert.Contains(t, rr.Body.String(), `"status":"failed"`)
}

func TestAdminHandlerListBotErrorsForbidden(t *testing.T) {
	handler := NewAdminHandler(nil, nil)
	claims := &auth.SessionClaims{Roles: []string{"user"}}
	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/bots/errors", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ListBotErrors(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}
