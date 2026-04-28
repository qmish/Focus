package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/repository"
)

// UserHandler обработчики для пользователей
type UserHandler struct {
	userRepo *repository.UserRepository
}

// NewUserHandler создаёт новый UserHandler
func NewUserHandler(userRepo *repository.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

// SearchUsers GET /api/v1/users/search?q=...&room_id=...
func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	roomIDStr := r.URL.Query().Get("room_id")
	const limit = 10

	var result interface{}
	var err error

	if roomIDStr != "" {
		roomID, parseErr := uuid.Parse(roomIDStr)
		if parseErr != nil {
			http.Error(w, "Некорректный room_id", http.StatusBadRequest)
			return
		}
		result, err = h.userRepo.SearchInRoom(r.Context(), q, roomID, limit)
	} else {
		result, err = h.userRepo.Search(r.Context(), q, limit)
	}

	if err != nil {
		http.Error(w, "Ошибка поиска пользователей", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
