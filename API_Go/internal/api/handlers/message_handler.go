package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/models"
)

// MessageHandler обработчики для messages
type MessageHandler struct {
	// TODO: Добавить message repository
}

// NewMessageHandler создаёт новый MessageHandler
func NewMessageHandler() *MessageHandler {
	return &MessageHandler{}
}

// ListMessages GET /api/v1/messages
func (h *MessageHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	// Получаем room_id из query параметров
	roomIDStr := r.URL.Query().Get("room_id")
	if roomIDStr == "" {
		http.Error(w, "room_id is required", http.StatusBadRequest)
		return
	}

	_, err := uuid.Parse(roomIDStr)
	if err != nil {
		http.Error(w, "invalid room_id", http.StatusBadRequest)
		return
	}

	// Получаем пагинацию
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 200 {
		limit = 50
	}

	// TODO: Реализовать получение сообщений из БД
	messages := []models.Message{}

	response := map[string]interface{}{
		"data":        messages,
		"has_more":    false,
		"next_cursor": "",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateMessage POST /api/v1/messages
func (h *MessageHandler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RoomID    string `json:"room_id"`
		Content   string `json:"content"`
		Type      string `json:"type"`
		ReplyToID string `json:"reply_to_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.RoomID == "" {
		http.Error(w, "room_id is required", http.StatusBadRequest)
		return
	}

	if req.Content == "" || len(req.Content) > 10000 {
		http.Error(w, "invalid content (1-10000 characters)", http.StatusBadRequest)
		return
	}

	roomID, err := uuid.Parse(req.RoomID)
	if err != nil {
		http.Error(w, "invalid room_id", http.StatusBadRequest)
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

	// Определяем тип сообщения
	msgType := models.MessageTypeText
	switch req.Type {
	case "image", "file", "system":
		msgType = models.MessageType(req.Type)
	}

	// Создаём сообщение
	message := models.NewMessage(roomID, userID, req.Content, msgType)

	// Обрабатываем reply_to
	if req.ReplyToID != "" {
		replyToID, err := uuid.Parse(req.ReplyToID)
		if err == nil {
			message.ReplyToID = &replyToID
		}
	}

	// TODO: Сохранить сообщение в БД

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(message)
}

// GetMessage GET /api/v1/messages/{id}
func (h *MessageHandler) GetMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	messageID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}

	// TODO: Получить сообщение из БД

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":    messageID.String(),
		"error": "not implemented",
	})
}

// UpdateMessage PUT /api/v1/messages/{id}
func (h *MessageHandler) UpdateMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	messageID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}

	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Обновить сообщение в БД

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":      messageID.String(),
		"content": req.Content,
		"error":   "not implemented",
	})
}

// DeleteMessage DELETE /api/v1/messages/{id}
func (h *MessageHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}

	// TODO: Удалить сообщение (мягкое удаление)

	w.WriteHeader(http.StatusNoContent)
}
