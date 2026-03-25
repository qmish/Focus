package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
	"github.com/qmish/focus-api/internal/webhooks"
)

// AdminHandler обработчики для админ-панели
type AdminHandler struct {
	userRepo    adminUserRepository
	roomRepo    adminRoomRepository
	webhookRepo adminWebhookRepository
}

type adminUserRepository interface {
	List(ctx context.Context, limit, offset int) ([]*models.User, error)
	Count(ctx context.Context) (int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
}

type adminRoomRepository interface {
	List(ctx context.Context, limit, offset int) ([]*models.Room, error)
	Count(ctx context.Context) (int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Room, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CountParticipants(ctx context.Context, roomID uuid.UUID) (int64, error)
}

type adminWebhookRepository interface {
	ListRecentDeliveries(ctx context.Context, limit int, onlyFailed bool) ([]*webhooks.WebhookDelivery, error)
}

// NewAdminHandler создаёт новый AdminHandler
func NewAdminHandler(userRepo adminUserRepository, roomRepo adminRoomRepository) *AdminHandler {
	return &AdminHandler{
		userRepo: userRepo,
		roomRepo: roomRepo,
	}
}

// SetWebhookRepository sets optional webhook repository for admin visibility endpoints.
func (h *AdminHandler) SetWebhookRepository(webhookRepo adminWebhookRepository) {
	h.webhookRepo = webhookRepo
}

// requireAdmin middleware для проверки роли администратора
func requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := auth.GetUserClaimsFromContext(r.Context())
		if claims == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		isAdmin := false
		for _, role := range claims.Roles {
			if role == "admin" {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

// ListUsers GET /api/v1/admin/users
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Проверяем роль администратора
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Получаем пагинацию
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	// Получаем пользователей
	users, err := h.userRepo.List(r.Context(), perPage, offset)
	if err != nil {
		http.Error(w, "failed to get users", http.StatusInternalServerError)
		return
	}

	// Получаем общее количество
	total, err := h.userRepo.Count(r.Context())
	if err != nil {
		http.Error(w, "failed to count users", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"data": users,
		"pagination": map[string]interface{}{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": (int(total) + perPage - 1) / perPage,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUser GET /api/v1/admin/users/:id
func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Проверяем роль администратора
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id := chi.URLParam(r, "id")
	userID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// UpdateUserRoles PUT /api/v1/admin/users/:id/roles
func (h *AdminHandler) UpdateUserRoles(w http.ResponseWriter, r *http.Request) {
	// Проверяем роль администратора
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id := chi.URLParam(r, "id")
	userID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var req struct {
		Roles []string `json:"roles"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	user.Roles = req.Roles
	if err := h.userRepo.Update(r.Context(), user); err != nil {
		http.Error(w, "failed to update user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// BanUser POST /api/v1/admin/users/:id/ban
func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	// Проверяем роль администратора
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id := chi.URLParam(r, "id")
	userID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var req struct {
		Reason        string `json:"reason"`
		DurationHours int    `json:"duration_hours"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	user.IsActive = false
	if err := h.userRepo.Update(r.Context(), user); err != nil {
		http.Error(w, "failed to ban user", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":           user.ID.String(),
		"banned":       true,
		"reason":       req.Reason,
		"banned_until": nil, // TODO: реализовать временные баны
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UnbanUser POST /api/v1/admin/users/:id/unban
func (h *AdminHandler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	// Проверяем роль администратора
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id := chi.URLParam(r, "id")
	userID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	user.IsActive = true
	if err := h.userRepo.Update(r.Context(), user); err != nil {
		http.Error(w, "failed to unban user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     user.ID.String(),
		"banned": false,
	})
}

// ListConferences GET /api/v1/admin/conferences
func (h *AdminHandler) ListConferences(w http.ResponseWriter, r *http.Request) {
	// Проверяем роль администратора
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Получаем пагинацию
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	if h.roomRepo == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		return
	}

	rooms, err := h.roomRepo.List(r.Context(), perPage, offset)
	if err != nil {
		http.Error(w, "failed to list conferences", http.StatusInternalServerError)
		return
	}

	type conferenceInfo struct {
		ID                string    `json:"id"`
		RoomID            string    `json:"room_id"`
		RoomName          string    `json:"room_name"`
		JitsiRoom         string    `json:"jitsi_room"`
		ParticipantsCount int64     `json:"participants_count"`
		StartedAt         time.Time `json:"started_at"`
		LastActivityAt    time.Time `json:"last_activity_at"`
		Status            string    `json:"status"`
	}

	conferences := make([]conferenceInfo, 0, len(rooms))
	for _, room := range rooms {
		if room == nil || room.Type != models.RoomTypeMeeting {
			continue
		}
		participantsCount, err := h.roomRepo.CountParticipants(r.Context(), room.ID)
		if err != nil {
			participantsCount = 0
		}

		conferences = append(conferences, conferenceInfo{
			ID:                room.ID.String(),
			RoomID:            room.ID.String(),
			RoomName:          room.Name,
			JitsiRoom:         room.JitsiRoomName,
			ParticipantsCount: participantsCount,
			StartedAt:         room.CreatedAt,
			LastActivityAt:    room.UpdatedAt,
			Status:            "active",
		})
	}

	response := map[string]interface{}{
		"data": conferences,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// EndConference POST /api/v1/admin/conferences/:id/end
func (h *AdminHandler) EndConference(w http.ResponseWriter, r *http.Request) {
	// Проверяем роль администратора
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "conference id is required", http.StatusBadRequest)
		return
	}
	roomID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid conference id", http.StatusBadRequest)
		return
	}

	if h.roomRepo == nil {
		http.Error(w, "conference service unavailable", http.StatusServiceUnavailable)
		return
	}

	room, err := h.roomRepo.GetByID(r.Context(), roomID)
	if err != nil {
		if err == repository.ErrRoomNotFound {
			http.Error(w, "conference not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get conference", http.StatusInternalServerError)
		return
	}
	if room.Type != models.RoomTypeMeeting {
		http.Error(w, "room is not a conference", http.StatusBadRequest)
		return
	}

	if err := h.roomRepo.Delete(r.Context(), roomID); err != nil {
		http.Error(w, "failed to end conference", http.StatusInternalServerError)
		return
	}

	endedAt := time.Now().UTC()
	response := map[string]interface{}{
		"id":       roomID.String(),
		"room_id":  roomID.String(),
		"ended":    true,
		"ended_at": endedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetStats GET /api/v1/admin/stats
func (h *AdminHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	// Проверяем роль администратора
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	ctx := r.Context()

	// Получаем статистику
	var userCount, roomCount int64
	if h.userRepo != nil {
		userCount, _ = h.userRepo.Count(ctx)
	}
	if h.roomRepo != nil {
		roomCount, _ = h.roomRepo.Count(ctx)
	}

	response := map[string]interface{}{
		"users": map[string]interface{}{
			"total": userCount,
		},
		"rooms": map[string]interface{}{
			"total": roomCount,
		},
		"conferences": map[string]interface{}{
			"active": 0,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListWebhookDeliveries GET /api/v1/admin/webhooks/deliveries
func (h *AdminHandler) ListWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 200 {
		limit = 50
	}

	if h.webhookRepo == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		return
	}

	deliveries, err := h.webhookRepo.ListRecentDeliveries(r.Context(), limit, false)
	if err != nil {
		http.Error(w, "failed to list webhook deliveries", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": deliveries})
}

// ListWebhookErrors GET /api/v1/admin/webhooks/errors
func (h *AdminHandler) ListWebhookErrors(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil || !hasRole(claims, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 200 {
		limit = 50
	}

	if h.webhookRepo == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data":  []interface{}{},
			"total": 0,
		})
		return
	}

	deliveries, err := h.webhookRepo.ListRecentDeliveries(r.Context(), limit, true)
	if err != nil {
		http.Error(w, "failed to list webhook errors", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  deliveries,
		"total": len(deliveries),
	})
}

// hasRole проверяет наличие роли у пользователя
func hasRole(claims *auth.SessionClaims, role string) bool {
	for _, r := range claims.Roles {
		if r == role {
			return true
		}
	}
	return false
}
