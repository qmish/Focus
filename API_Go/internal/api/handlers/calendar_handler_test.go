package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/exchange"
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
	deleted  []string
	notified []string
}

func (f *fakeCalendarService) GetEvents(ctx context.Context, userID string, start, end time.Time) ([]exchange.CalendarEvent, error) {
	return []exchange.CalendarEvent{}, nil
}

func (f *fakeCalendarService) CreateEvent(ctx context.Context, userID string, event exchange.CalendarEvent) (*exchange.CalendarEvent, error) {
	return &event, nil
}

func (f *fakeCalendarService) UpdateEvent(ctx context.Context, userID, eventID string, event exchange.CalendarEvent) error {
	return nil
}

func (f *fakeCalendarService) DeleteEvent(ctx context.Context, userID, eventID string) error {
	f.deleted = append(f.deleted, eventID)
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
