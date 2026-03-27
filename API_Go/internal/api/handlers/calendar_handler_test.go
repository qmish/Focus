package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/exchange"
	"github.com/qmish/focus-api/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestDeleteEventCancellationNotification(t *testing.T) {
	t.Run("sends cancellation notification by default", func(t *testing.T) {
		client := &fakeCalendarService{}
		handler := &CalendarHandler{
			calendarService: client,
			cancellationNotifier: func(ctx context.Context, userEmail, eventID string) error {
				client.notified = append(client.notified, eventID)
				return nil
			},
		}

		req := withClaims(withURLParam(
			httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/events/event-1", nil),
			"id",
			"event-1",
		))
		rr := httptest.NewRecorder()
		handler.DeleteEvent(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
		assert.Equal(t, []string{"event-1"}, client.deleted)
		assert.Equal(t, []string{"event-1"}, client.notified)
	})

	t.Run("skips cancellation notification when disabled", func(t *testing.T) {
		client := &fakeCalendarService{}
		handler := &CalendarHandler{
			calendarService: client,
			cancellationNotifier: func(ctx context.Context, userEmail, eventID string) error {
				client.notified = append(client.notified, eventID)
				return nil
			},
		}

		req := withClaims(withURLParam(
			httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/events/event-2?send_cancellation=false", nil),
			"id",
			"event-2",
		))
		rr := httptest.NewRecorder()
		handler.DeleteEvent(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
		assert.Equal(t, []string{"event-2"}, client.deleted)
		assert.Empty(t, client.notified)
	})
}

func TestDeleteEventUnavailableCalendarService(t *testing.T) {
	handler := &CalendarHandler{}
	req := withClaims(withURLParam(
		httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/events/event-1", nil),
		"id",
		"event-1",
	))
	rr := httptest.NewRecorder()
	handler.DeleteEvent(rr, req)
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
}

type fakeCalendarService struct {
	deleted    []string
	notified   []string
	failCreate bool
	failUpdate bool
	failDelete bool
}

func (f *fakeCalendarService) GetEvents(ctx context.Context, userID string, start, end time.Time) ([]exchange.CalendarEvent, error) {
	return []exchange.CalendarEvent{}, nil
}

func (f *fakeCalendarService) CreateEvent(ctx context.Context, userID string, event exchange.CalendarEvent) (*exchange.CalendarEvent, error) {
	if f.failCreate {
		return nil, assert.AnError
	}
	event.ID = "event-created-1"
	return &event, nil
}

func (f *fakeCalendarService) GetEvent(ctx context.Context, userID, eventID string) (*exchange.CalendarEvent, error) {
	return &exchange.CalendarEvent{ID: eventID, Subject: "event"}, nil
}

func (f *fakeCalendarService) UpdateEvent(ctx context.Context, userID, eventID string, event exchange.CalendarEvent) error {
	if f.failUpdate {
		return assert.AnError
	}
	return nil
}

func (f *fakeCalendarService) DeleteEvent(ctx context.Context, userID, eventID string) error {
	if f.failDelete {
		return assert.AnError
	}
	f.deleted = append(f.deleted, eventID)
	return nil
}

type fakeCalendarAuditRepo struct {
	events []*models.CalendarAuditEvent
}

type fakeMeetingLinkRepo struct {
	links map[string]*models.MeetingLink
}

type fakeCalendarIdempotencyRepo struct {
	records map[string]*models.CalendarIdempotencyKey
}

func (f *fakeMeetingLinkRepo) Create(ctx context.Context, link *models.MeetingLink) error {
	if f.links == nil {
		f.links = make(map[string]*models.MeetingLink)
	}
	f.links[link.ExchangeEventID] = link
	return nil
}

func (f *fakeMeetingLinkRepo) GetByExchangeEventID(ctx context.Context, eventID string) (*models.MeetingLink, error) {
	if f.links == nil {
		return nil, errors.New("not found")
	}
	link, ok := f.links[eventID]
	if !ok {
		return nil, errors.New("not found")
	}
	return link, nil
}

func (f *fakeMeetingLinkRepo) Update(ctx context.Context, link *models.MeetingLink) error {
	if f.links == nil {
		f.links = make(map[string]*models.MeetingLink)
	}
	f.links[link.ExchangeEventID] = link
	return nil
}

func (f *fakeCalendarIdempotencyRepo) CreatePending(ctx context.Context, key, userEmail string) error {
	if f.records == nil {
		f.records = make(map[string]*models.CalendarIdempotencyKey)
	}
	f.records[key+"|"+userEmail] = &models.CalendarIdempotencyKey{
		Key:       key,
		UserEmail: userEmail,
	}
	return nil
}

func (f *fakeCalendarIdempotencyRepo) Get(ctx context.Context, key, userEmail string) (*models.CalendarIdempotencyKey, error) {
	if f.records == nil {
		return nil, errors.New("not found")
	}
	record, ok := f.records[key+"|"+userEmail]
	if !ok {
		return nil, errors.New("not found")
	}
	return record, nil
}

func (f *fakeCalendarIdempotencyRepo) MarkCompleted(ctx context.Context, key, userEmail, eventID, roomID, responseBody string) error {
	if f.records == nil {
		f.records = make(map[string]*models.CalendarIdempotencyKey)
	}
	now := time.Now().UTC()
	f.records[key+"|"+userEmail] = &models.CalendarIdempotencyKey{
		Key:          key,
		UserEmail:    userEmail,
		EventID:      eventID,
		RoomID:       roomID,
		ResponseBody: responseBody,
		CompletedAt:  &now,
	}
	return nil
}

func (f *fakeCalendarAuditRepo) CreateCalendarAuditEvent(ctx context.Context, event *models.CalendarAuditEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	f.events = append(f.events, event)
	return nil
}

func withURLParam(req *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func withClaims(req *http.Request) *http.Request {
	claims := &auth.SessionClaims{
		UserID: "user-1",
		Email:  "user@example.com",
		Name:   "User",
		Roles:  []string{"user"},
	}
	return req.WithContext(context.WithValue(req.Context(), auth.ContextKeyUserClaims, claims))
}

func TestCalendarHandlerCreateEventAudit(t *testing.T) {
	client := &fakeCalendarService{}
	auditRepo := &fakeCalendarAuditRepo{}
	linkRepo := &fakeMeetingLinkRepo{}
	handler := &CalendarHandler{calendarService: client}
	handler.SetCalendarAuditRepository(auditRepo)
	handler.SetMeetingLinkRepository(linkRepo)

	body := `{"subject":"Demo","description":"test","start_time":"2030-01-01T10:00:00Z","end_time":"2030-01-01T11:00:00Z","attendee_emails":["a@example.com"],"create_jitsi_room":false}`
	req := withClaims(httptest.NewRequest(http.MethodPost, "/api/v1/calendar/events", strings.NewReader(body)))
	rr := httptest.NewRecorder()
	handler.CreateEvent(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	if assert.NotEmpty(t, auditRepo.events) {
		assert.Equal(t, "create", auditRepo.events[0].Operation)
		assert.Equal(t, "success", auditRepo.events[0].Status)
		assert.Equal(t, "event-created-1", auditRepo.events[0].EventID)
	}
	if assert.NotNil(t, linkRepo.links["event-created-1"]) {
		assert.Equal(t, "scheduled", linkRepo.links["event-created-1"].Status)
	}
}

func TestCalendarHandlerUpdateEventAudit(t *testing.T) {
	client := &fakeCalendarService{}
	auditRepo := &fakeCalendarAuditRepo{}
	handler := &CalendarHandler{calendarService: client}
	handler.SetCalendarAuditRepository(auditRepo)

	body := `{"subject":"Updated","description":"text"}`
	req := withClaims(withURLParam(
		httptest.NewRequest(http.MethodPut, "/api/v1/calendar/events/event-77", strings.NewReader(body)),
		"id",
		"event-77",
	))
	rr := httptest.NewRecorder()
	handler.UpdateEvent(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	if assert.NotEmpty(t, auditRepo.events) {
		assert.Equal(t, "update", auditRepo.events[0].Operation)
		assert.Equal(t, "success", auditRepo.events[0].Status)
		assert.Equal(t, "event-77", auditRepo.events[0].EventID)
	}
}

func TestCalendarHandlerDeleteEventAuditFailure(t *testing.T) {
	client := &fakeCalendarService{failDelete: true}
	auditRepo := &fakeCalendarAuditRepo{}
	handler := &CalendarHandler{calendarService: client}
	handler.SetCalendarAuditRepository(auditRepo)

	req := withClaims(withURLParam(
		httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/events/event-5", nil),
		"id",
		"event-5",
	))
	rr := httptest.NewRecorder()
	handler.DeleteEvent(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	if assert.NotEmpty(t, auditRepo.events) {
		assert.Equal(t, "delete", auditRepo.events[0].Operation)
		assert.Equal(t, "failed", auditRepo.events[0].Status)
		assert.Equal(t, "event-5", auditRepo.events[0].EventID)
	}
}

func TestCalendarHandlerCreateEventIdempotentReplay(t *testing.T) {
	client := &fakeCalendarService{}
	idempotencyRepo := &fakeCalendarIdempotencyRepo{}
	handler := &CalendarHandler{calendarService: client}
	handler.SetCalendarIdempotencyRepository(idempotencyRepo)

	body := `{"subject":"Demo","description":"test","start_time":"2030-01-01T10:00:00Z","end_time":"2030-01-01T11:00:00Z","attendee_emails":["a@example.com"],"create_jitsi_room":false}`
	req1 := withClaims(httptest.NewRequest(http.MethodPost, "/api/v1/calendar/events", strings.NewReader(body)))
	req1.Header.Set("Idempotency-Key", "cal-req-1")
	rr1 := httptest.NewRecorder()
	handler.CreateEvent(rr1, req1)
	assert.Equal(t, http.StatusCreated, rr1.Code)

	req2 := withClaims(httptest.NewRequest(http.MethodPost, "/api/v1/calendar/events", strings.NewReader(body)))
	req2.Header.Set("Idempotency-Key", "cal-req-1")
	rr2 := httptest.NewRecorder()
	handler.CreateEvent(rr2, req2)

	assert.Equal(t, http.StatusOK, rr2.Code)
	assert.Equal(t, "true", rr2.Header().Get("X-Idempotent-Replay"))
	assert.Equal(t, rr1.Body.String(), rr2.Body.String())
}
