package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
)

// RoomHandler обработчики для rooms
type RoomHandler struct {
	roomRepo *repository.RoomRepository
	userRepo *repository.UserRepository
}

// NewRoomHandler создаёт новый RoomHandler
func NewRoomHandler(roomRepo *repository.RoomRepository, userRepo *repository.UserRepository) *RoomHandler {
	return &RoomHandler{
		roomRepo: roomRepo,
		userRepo: userRepo,
	}
}

// ListRooms GET /api/v1/rooms
func (h *RoomHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
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

	// Получаем пользователя из контекста
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusInternalServerError)
		return
	}

	// Получаем комнаты пользователя
	rooms, err := h.roomRepo.ListByParticipant(r.Context(), userID, perPage, offset)
	if err != nil {
		http.Error(w, "failed to get rooms", http.StatusInternalServerError)
		return
	}

	// Получаем общее количество
	total, err := h.roomRepo.Count(r.Context())
	if err != nil {
		http.Error(w, "failed to count rooms", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"data": rooms,
		"pagination": map[string]interface{}{
			"page":      page,
			"per_page":  perPage,
			"total":     total,
			"total_pages": (int(total) + perPage - 1) / perPage,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateRoom POST /api/v1/rooms
func (h *RoomHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Type        string   `json:"type"`
		ParticipantIDs []string `json:"participant_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.Name == "" || len(req.Name) < 3 || len(req.Name) > 100 {
		http.Error(w, "invalid room name (3-100 characters)", http.StatusBadRequest)
		return
	}

	// Получаем пользователя из контекста
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusInternalServerError)
		return
	}

	// Определяем тип комнаты
	roomType := models.RoomTypePublic
	switch req.Type {
	case "private":
		roomType = models.RoomTypePrivate
	case "meeting":
		roomType = models.RoomTypeMeeting
	}

	// Создаём комнату
	room := models.NewRoom(req.Name, userID, roomType)
	room.Description = req.Description

	if err := h.roomRepo.Create(r.Context(), room); err != nil {
		http.Error(w, "failed to create room", http.StatusInternalServerError)
		return
	}

	// Добавляем создателя как участника
	if err := h.roomRepo.AddParticipant(r.Context(), room.ID, userID, models.ParticipantRoleAdmin); err != nil {
		http.Error(w, "failed to add participant", http.StatusInternalServerError)
		return
	}

	// Добавляем остальных участников
	for _, pid := range req.ParticipantIDs {
		participantID, err := uuid.Parse(pid)
		if err != nil {
			continue
		}
		h.roomRepo.AddParticipant(r.Context(), room.ID, participantID, models.ParticipantRoleMember)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(room)
}

// GetRoom GET /api/v1/rooms/{id}
func (h *RoomHandler) GetRoom(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	roomID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return
	}

	room, err := h.roomRepo.GetByID(r.Context(), roomID)
	if err != nil {
		if err == repository.ErrRoomNotFound {
			http.Error(w, "room not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get room", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room)
}

// UpdateRoom PUT /api/v1/rooms/{id}
func (h *RoomHandler) UpdateRoom(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	roomID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return
	}

	var req struct {
		Name        string              `json:"name"`
		Description string              `json:"description"`
		Settings    *models.RoomSettings `json:"settings"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	room, err := h.roomRepo.GetByID(r.Context(), roomID)
	if err != nil {
		if err == repository.ErrRoomNotFound {
			http.Error(w, "room not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get room", http.StatusInternalServerError)
		return
	}

	// Обновляем поля
	if req.Name != "" {
		room.Name = req.Name
	}
	if req.Description != "" {
		room.Description = req.Description
	}
	if req.Settings != nil {
		room.Settings = *req.Settings
	}

	if err := h.roomRepo.Update(r.Context(), room); err != nil {
		http.Error(w, "failed to update room", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room)
}

// DeleteRoom DELETE /api/v1/rooms/{id}
func (h *RoomHandler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	roomID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return
	}

	if err := h.roomRepo.Delete(r.Context(), roomID); err != nil {
		http.Error(w, "failed to delete room", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// JoinRoom POST /api/v1/rooms/{id}/join
func (h *RoomHandler) JoinRoom(w http.ResponseWriter, r *http.Request) {
	// Реализация будет добавлена позже
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}
