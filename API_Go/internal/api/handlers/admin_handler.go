package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/repository"
)

// AdminHandler обработчики для админ-панели
type AdminHandler struct {
	userRepo *repository.UserRepository
	roomRepo *repository.RoomRepository
}

// NewAdminHandler создаёт новый AdminHandler
func NewAdminHandler(userRepo *repository.UserRepository, roomRepo *repository.RoomRepository) *AdminHandler {
	return &AdminHandler{
		userRepo: userRepo,
		roomRepo: roomRepo,
	}
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

	// TODO: Получить активные конференции из Jitsi
	// Пока возвращаем пустой список
	response := map[string]interface{}{
		"data": []interface{}{},
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
	// conference_id может быть строкой, не обязательно UUID

	// TODO: Завершить конференцию через Jitsi API
	response := map[string]interface{}{
		"id":       id,
		"ended":    true,
		"ended_at": nil,
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

// hasRole проверяет наличие роли у пользователя
func hasRole(claims *auth.SessionClaims, role string) bool {
	for _, r := range claims.Roles {
		if r == role {
			return true
		}
	}
	return false
}
