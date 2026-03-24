package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/exchange"
	"github.com/qmish/focus-api/internal/jitsi"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
)

// CalendarHandler обработчики для календаря
type CalendarHandler struct {
	graphClient *exchange.GraphClient
	roomRepo    *repository.RoomRepository
	jitsiGen    *jitsi.TokenGenerator
}

// NewCalendarHandler создаёт новый CalendarHandler
func NewCalendarHandler(graphClient *exchange.GraphClient, roomRepo *repository.RoomRepository, jitsiGen *jitsi.TokenGenerator) *CalendarHandler {
	return &CalendarHandler{
		graphClient: graphClient,
		roomRepo:    roomRepo,
		jitsiGen:    jitsiGen,
	}
}

// GetEvents GET /api/v1/calendar/events
func (h *CalendarHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем параметры времени
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	var start, end time.Time
	var err error

	if startStr != "" {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			http.Error(w, "invalid start time", http.StatusBadRequest)
			return
		}
	} else {
		start = time.Now()
	}

	if endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			http.Error(w, "invalid end time", http.StatusBadRequest)
			return
		}
	} else {
		end = start.Add(7 * 24 * time.Hour) // Неделя по умолчанию
	}

	// Получаем события из Exchange
	// В production использовать реальный email пользователя из claims
	userEmail := claims.Email
	events, err := h.graphClient.GetEvents(r.Context(), userEmail, start, end)
	if err != nil {
		http.Error(w, "failed to get events", http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	response := map[string]interface{}{
		"data": events,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateEvent POST /api/v1/calendar/events
func (h *CalendarHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Subject         string   `json:"subject"`
		Description     string   `json:"description"`
		StartTime       string   `json:"start_time"`
		EndTime         string   `json:"end_time"`
		AttendeeEmails  []string `json:"attendee_emails"`
		CreateJitsiRoom bool     `json:"create_jitsi_room"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.Subject == "" {
		http.Error(w, "subject is required", http.StatusBadRequest)
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		http.Error(w, "invalid start_time", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		http.Error(w, "invalid end_time", http.StatusBadRequest)
		return
	}

	if endTime.Before(startTime) {
		http.Error(w, "end_time must be after start_time", http.StatusBadRequest)
		return
	}

	// Создаём Jitsi комнату если нужно
	var jitsiURL string
	var roomID string
	if req.CreateJitsiRoom {
		userID, _ := uuid.Parse(claims.UserID)
		room := models.NewRoom(req.Subject, userID, "meeting")

		if err := h.roomRepo.Create(r.Context(), room); err != nil {
			http.Error(w, "failed to create room", http.StatusInternalServerError)
			return
		}

		jitsiURL = h.jitsiGen.BaseURL() + "/" + room.JitsiRoomName
		roomID = room.ID.String()
	}

	// Формируем attendees
	attendees := make([]exchange.EventAttendee, 0, len(req.AttendeeEmails))
	for _, email := range req.AttendeeEmails {
		attendees = append(attendees, exchange.EventAttendee{
			Email:  email,
			Status: "pending",
		})
	}

	// Создаём событие в Exchange
	event := exchange.CalendarEvent{
		Subject:     req.Subject,
		Description: req.Description,
		StartTime:   startTime,
		EndTime:     endTime,
		Location:    "Jitsi Meeting",
		JitsiURL:    jitsiURL,
		Attendees:   attendees,
		Organizer: exchange.EventAttendee{
			Email: claims.Email,
			Name:  claims.Name,
		},
	}

	// В production использовать реальный email пользователя
	userEmail := claims.Email
	createdEvent, err := h.graphClient.CreateEvent(r.Context(), userEmail, event)
	if err != nil {
		http.Error(w, "failed to create event", http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	response := map[string]interface{}{
		"id":               createdEvent.ID,
		"subject":          createdEvent.Subject,
		"start_time":       createdEvent.StartTime,
		"end_time":         createdEvent.EndTime,
		"jitsi_url":        createdEvent.JitsiURL,
		"room_id":          roomID,
		"invitations_sent": len(createdEvent.Attendees),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// UpdateEvent PUT /api/v1/calendar/events/:id
func (h *CalendarHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		http.Error(w, "event id is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Subject     string `json:"subject"`
		Description string `json:"description"`
		StartTime   string `json:"start_time"`
		EndTime     string `json:"end_time"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	event := exchange.CalendarEvent{
		Subject:     req.Subject,
		Description: req.Description,
	}

	if req.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			http.Error(w, "invalid start_time", http.StatusBadRequest)
			return
		}
		event.StartTime = startTime
	}

	if req.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			http.Error(w, "invalid end_time", http.StatusBadRequest)
			return
		}
		event.EndTime = endTime
	}

	// В production использовать реальный email пользователя
	userEmail := claims.Email
	err := h.graphClient.UpdateEvent(r.Context(), userEmail, eventID, event)
	if err != nil {
		http.Error(w, "failed to update event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":     eventID,
		"status": "updated",
	})
}

// DeleteEvent DELETE /api/v1/calendar/events/:id
func (h *CalendarHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		http.Error(w, "event id is required", http.StatusBadRequest)
		return
	}

	sendCancellation := r.URL.Query().Get("send_cancellation") != "false"
	_ = sendCancellation // TODO: Реализовать отправку уведомлений об отмене

	// В production использовать реальный email пользователя
	userEmail := claims.Email
	err := h.graphClient.DeleteEvent(r.Context(), userEmail, eventID)
	if err != nil {
		http.Error(w, "failed to delete event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
