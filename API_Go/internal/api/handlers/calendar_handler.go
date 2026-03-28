package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
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
	calendarService      calendarService
	roomRepo             *repository.RoomRepository
	jitsiGen             *jitsi.TokenGenerator
	cancellationNotifier CancellationNotifier
	calendarAuditRepo    calendarAuditRepository
	meetingLinkRepo      meetingLinkRepository
	idempotencyRepo      calendarIdempotencyRepository
}

type calendarService interface {
	GetEvents(ctx context.Context, userID string, start, end time.Time) ([]exchange.CalendarEvent, error)
	CreateEvent(ctx context.Context, userID string, event exchange.CalendarEvent) (*exchange.CalendarEvent, error)
	GetEvent(ctx context.Context, userID, eventID string) (*exchange.CalendarEvent, error)
	UpdateEvent(ctx context.Context, userID, eventID string, event exchange.CalendarEvent) error
	DeleteEvent(ctx context.Context, userID, eventID string) error
}

type CancellationNotifier func(ctx context.Context, userEmail, eventID string) error

type calendarAuditRepository interface {
	CreateCalendarAuditEvent(ctx context.Context, event *models.CalendarAuditEvent) error
}

type meetingLinkRepository interface {
	Create(ctx context.Context, link *models.MeetingLink) error
	GetByExchangeEventID(ctx context.Context, eventID string) (*models.MeetingLink, error)
	Update(ctx context.Context, link *models.MeetingLink) error
}

type calendarIdempotencyRepository interface {
	CreatePending(ctx context.Context, key, userEmail string) error
	Get(ctx context.Context, key, userEmail string) (*models.CalendarIdempotencyKey, error)
	MarkCompleted(ctx context.Context, key, userEmail, eventID, roomID, responseBody string) error
}

type calendarEventResponse struct {
	ID              string                   `json:"id"`
	Subject         string                   `json:"subject"`
	Description     string                   `json:"description,omitempty"`
	StartTime       time.Time                `json:"start_time"`
	EndTime         time.Time                `json:"end_time"`
	Location        string                   `json:"location,omitempty"`
	JitsiURL        string                   `json:"jitsi_url,omitempty"`
	RoomID          string                   `json:"room_id,omitempty"`
	SyncStatus      string                   `json:"sync_status,omitempty"`
	ExchangeEventID string                   `json:"exchange_event_id,omitempty"`
	Organizer       exchange.EventAttendee   `json:"organizer,omitempty"`
	Attendees       []exchange.EventAttendee `json:"attendees,omitempty"`
}

// NewCalendarHandler создаёт новый CalendarHandler
func NewCalendarHandler(service exchange.CalendarService, roomRepo *repository.RoomRepository, jitsiGen *jitsi.TokenGenerator) *CalendarHandler {
	return &CalendarHandler{
		calendarService: service,
		roomRepo:        roomRepo,
		jitsiGen:        jitsiGen,
		cancellationNotifier: func(ctx context.Context, userEmail, eventID string) error {
			return nil
		},
	}
}

// SetCancellationNotifier sets custom cancellation notification sender.
func (h *CalendarHandler) SetCancellationNotifier(notifier CancellationNotifier) {
	if notifier == nil {
		return
	}
	h.cancellationNotifier = notifier
}

// SetCalendarAuditRepository sets optional calendar audit repository.
func (h *CalendarHandler) SetCalendarAuditRepository(repo calendarAuditRepository) {
	h.calendarAuditRepo = repo
}

// SetMeetingLinkRepository sets optional room/event mapping repository.
func (h *CalendarHandler) SetMeetingLinkRepository(repo meetingLinkRepository) {
	h.meetingLinkRepo = repo
}

func (h *CalendarHandler) SetCalendarIdempotencyRepository(repo calendarIdempotencyRepository) {
	h.idempotencyRepo = repo
}

// GetEvents GET /api/v1/calendar/events
func (h *CalendarHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	if h.calendarService == nil {
		http.Error(w, "Служба календаря недоступна", http.StatusServiceUnavailable)
		return
	}
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
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
			http.Error(w, "Некорректное время начала", http.StatusBadRequest)
			return
		}
	} else {
		start = time.Now()
	}

	if endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			http.Error(w, "Некорректное время окончания", http.StatusBadRequest)
			return
		}
	} else {
		end = start.Add(7 * 24 * time.Hour) // Неделя по умолчанию
	}

	// Получаем события из Exchange
	// В production использовать реальный email пользователя из claims
	userEmail := claims.Email
	events, err := h.calendarService.GetEvents(r.Context(), userEmail, start, end)
	if err != nil {
		log.Printf("calendar: не удалось получить события для %s: %v", userEmail, err)
		events = nil
	}

	// Формируем ответ
	enriched := make([]calendarEventResponse, 0, len(events))
	for _, event := range events {
		item := toCalendarEventResponse(event)
		if h.meetingLinkRepo != nil && strings.TrimSpace(event.ID) != "" {
			if link, linkErr := h.meetingLinkRepo.GetByExchangeEventID(r.Context(), event.ID); linkErr == nil {
				item.RoomID = link.RoomID.String()
				item.SyncStatus = link.Status
			}
		}
		enriched = append(enriched, item)
	}
	response := map[string]interface{}{
		"data": enriched,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateEvent POST /api/v1/calendar/events
func (h *CalendarHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	if h.calendarService == nil {
		http.Error(w, "Служба календаря недоступна", http.StatusServiceUnavailable)
		return
	}
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}
	userEmail := strings.TrimSpace(strings.ToLower(claims.Email))
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey != "" && h.idempotencyRepo != nil {
		if existing, err := h.idempotencyRepo.Get(r.Context(), idempotencyKey, userEmail); err == nil &&
			existing.CompletedAt != nil && strings.TrimSpace(existing.ResponseBody) != "" {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Idempotent-Replay", "true")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(existing.ResponseBody))
			return
		}
		_ = h.idempotencyRepo.CreatePending(r.Context(), idempotencyKey, userEmail)
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
		http.Error(w, "Некорректные данные запроса", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.Subject == "" {
		http.Error(w, "Отсутствует тема", http.StatusBadRequest)
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		http.Error(w, "Некорректное start_time", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		http.Error(w, "Некорректное end_time", http.StatusBadRequest)
		return
	}

	if endTime.Before(startTime) {
		http.Error(w, "Время окончания должно быть позже времени начала", http.StatusBadRequest)
		return
	}

	// Создаём Jitsi комнату если нужно
	var jitsiURL string
	var roomID string
	if req.CreateJitsiRoom {
		userID, _ := uuid.Parse(claims.UserID)
		room := models.NewRoom(req.Subject, userID, "meeting")

		if err := h.roomRepo.Create(r.Context(), room); err != nil {
			http.Error(w, "Не удалось создать комнату", http.StatusInternalServerError)
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

	createdEvent, err := h.calendarService.CreateEvent(r.Context(), userEmail, event)
	if err != nil {
		h.recordCalendarAudit(r, "create", "failed", "", "create_event_failed")
		http.Error(w, "Не удалось создать событие", http.StatusInternalServerError)
		return
	}
	h.recordCalendarAudit(r, "create", "success", createdEvent.ID, "")

	var link *models.MeetingLink
	if h.meetingLinkRepo != nil && strings.TrimSpace(createdEvent.ID) != "" {
		organizer := strings.TrimSpace(claims.Email)
		if organizer == "" {
			organizer = strings.TrimSpace(createdEvent.Organizer.Email)
		}
		now := time.Now().UTC()
		link = &models.MeetingLink{
			ID:              uuid.New(),
			RoomID:          uuid.Nil,
			ExchangeEventID: createdEvent.ID,
			OrganizerEmail:  organizer,
			Subject:         strings.TrimSpace(createdEvent.Subject),
			StartAt:         createdEvent.StartTime.UTC(),
			EndAt:           createdEvent.EndTime.UTC(),
			Status:          "scheduled",
			SyncSource:      "focus",
			LastSyncAt:      &now,
		}
		if roomID != "" {
			if roomUUID, parseErr := uuid.Parse(roomID); parseErr == nil {
				link.RoomID = roomUUID
			}
		}
		if err := h.meetingLinkRepo.Create(r.Context(), link); err != nil {
			log.Printf("WARNING: failed to create meeting link for event %s: %v", createdEvent.ID, err)
		}
	}

	// Формируем ответ
	resp := toCalendarEventResponse(*createdEvent)
	resp.RoomID = roomID
	if link != nil {
		resp.SyncStatus = link.Status
	}
	response := map[string]interface{}{
		"id":                resp.ID,
		"subject":           resp.Subject,
		"description":       resp.Description,
		"start_time":        resp.StartTime,
		"end_time":          resp.EndTime,
		"jitsi_url":         resp.JitsiURL,
		"location":          resp.Location,
		"room_id":           resp.RoomID,
		"sync_status":       resp.SyncStatus,
		"exchange_event_id": resp.ExchangeEventID,
		"invitations_sent":  len(createdEvent.Attendees),
	}
	responseBytes, _ := json.Marshal(response)
	if idempotencyKey != "" && h.idempotencyRepo != nil {
		_ = h.idempotencyRepo.MarkCompleted(
			r.Context(),
			idempotencyKey,
			userEmail,
			createdEvent.ID,
			roomID,
			string(responseBytes),
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(responseBytes)
}

// UpdateEvent PUT /api/v1/calendar/events/:id
func (h *CalendarHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	if h.calendarService == nil {
		http.Error(w, "Служба календаря недоступна", http.StatusServiceUnavailable)
		return
	}
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		http.Error(w, "Отсутствует идентификатор события", http.StatusBadRequest)
		return
	}

	var req struct {
		Subject     string `json:"subject"`
		Description string `json:"description"`
		StartTime   string `json:"start_time"`
		EndTime     string `json:"end_time"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректные данные запроса", http.StatusBadRequest)
		return
	}

	event := exchange.CalendarEvent{
		Subject:     req.Subject,
		Description: req.Description,
	}

	if req.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			http.Error(w, "Некорректное start_time", http.StatusBadRequest)
			return
		}
		event.StartTime = startTime
	}

	if req.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			http.Error(w, "Некорректное end_time", http.StatusBadRequest)
			return
		}
		event.EndTime = endTime
	}

	// В production использовать реальный email пользователя
	userEmail := claims.Email
	err := h.calendarService.UpdateEvent(r.Context(), userEmail, eventID, event)
	if err != nil {
		h.recordCalendarAudit(r, "update", "failed", eventID, "update_event_failed")
		http.Error(w, "Не удалось обновить событие", http.StatusInternalServerError)
		return
	}
	h.recordCalendarAudit(r, "update", "success", eventID, "")
	if h.meetingLinkRepo != nil {
		if link, lookupErr := h.meetingLinkRepo.GetByExchangeEventID(r.Context(), eventID); lookupErr == nil {
			if strings.TrimSpace(event.Subject) != "" {
				link.Subject = strings.TrimSpace(event.Subject)
			}
			if !event.StartTime.IsZero() {
				link.StartAt = event.StartTime.UTC()
			}
			if !event.EndTime.IsZero() {
				link.EndAt = event.EndTime.UTC()
			}
			now := time.Now().UTC()
			link.LastSyncAt = &now
			link.Status = "updated"
			_ = h.meetingLinkRepo.Update(r.Context(), link)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":     eventID,
		"status": "updated",
	})
}

// DeleteEvent DELETE /api/v1/calendar/events/:id
func (h *CalendarHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	if h.calendarService == nil {
		http.Error(w, "Служба календаря недоступна", http.StatusServiceUnavailable)
		return
	}
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		http.Error(w, "Отсутствует идентификатор события", http.StatusBadRequest)
		return
	}

	sendCancellation := r.URL.Query().Get("send_cancellation") != "false"

	// В production использовать реальный email пользователя
	userEmail := claims.Email
	err := h.calendarService.DeleteEvent(r.Context(), userEmail, eventID)
	if err != nil {
		h.recordCalendarAudit(r, "delete", "failed", eventID, "delete_event_failed")
		http.Error(w, "Не удалось удалить событие", http.StatusInternalServerError)
		return
	}
	h.recordCalendarAudit(r, "delete", "success", eventID, "")
	if h.meetingLinkRepo != nil {
		if link, lookupErr := h.meetingLinkRepo.GetByExchangeEventID(r.Context(), eventID); lookupErr == nil {
			link.Status = "cancelled"
			now := time.Now().UTC()
			link.LastSyncAt = &now
			_ = h.meetingLinkRepo.Update(r.Context(), link)
		}
	}
	if sendCancellation && h.cancellationNotifier != nil {
		_ = h.cancellationNotifier(r.Context(), userEmail, eventID)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CalendarHandler) recordCalendarAudit(r *http.Request, operation, status, eventID, details string) {
	if h.calendarAuditRepo == nil {
		return
	}
	claims := auth.GetUserClaimsFromContext(r.Context())
	userID := ""
	userEmail := ""
	if claims != nil {
		userID = strings.TrimSpace(claims.UserID)
		userEmail = strings.TrimSpace(claims.Email)
	}
	_ = h.calendarAuditRepo.CreateCalendarAuditEvent(r.Context(), &models.CalendarAuditEvent{
		ID:        uuid.New(),
		Operation: strings.TrimSpace(operation),
		Status:    strings.TrimSpace(status),
		EventID:   strings.TrimSpace(eventID),
		UserID:    userID,
		UserEmail: userEmail,
		Details:   strings.TrimSpace(details),
		CreatedAt: time.Now().UTC(),
	})
}

func toCalendarEventResponse(event exchange.CalendarEvent) calendarEventResponse {
	return calendarEventResponse{
		ID:              strings.TrimSpace(event.ID),
		Subject:         strings.TrimSpace(event.Subject),
		Description:     strings.TrimSpace(event.Description),
		StartTime:       event.StartTime,
		EndTime:         event.EndTime,
		Location:        strings.TrimSpace(event.Location),
		JitsiURL:        strings.TrimSpace(event.JitsiURL),
		RoomID:          strings.TrimSpace(event.RoomID),
		SyncStatus:      strings.TrimSpace(event.SyncStatus),
		ExchangeEventID: strings.TrimSpace(event.ExchangeEventID),
		Organizer:       event.Organizer,
		Attendees:       event.Attendees,
	}
}
