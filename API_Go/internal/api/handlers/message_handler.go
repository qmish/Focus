package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/bots"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
	"github.com/qmish/focus-api/internal/websocket"
)

// MessageHandler обработчики для messages
type MessageHandler struct {
	msgRepo    *repository.MessageRepository
	wsHub      *websocket.Hub
	botEngine  *bots.BotEngine
}

// NewMessageHandler создаёт новый MessageHandler
func NewMessageHandler(msgRepo *repository.MessageRepository, wsHub *websocket.Hub, botEngine *bots.BotEngine) *MessageHandler {
	return &MessageHandler{
		msgRepo:   msgRepo,
		wsHub:     wsHub,
		botEngine: botEngine,
	}
}

// ListMessages GET /api/v1/messages
func (h *MessageHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	// Получаем room_id из query параметров
	roomIDStr := r.URL.Query().Get("room_id")
	if roomIDStr == "" {
		http.Error(w, "Отсутствует room_id", http.StatusBadRequest)
		return
	}

	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		http.Error(w, "Некорректный room_id", http.StatusBadRequest)
		return
	}

	// Получаем пагинацию
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 200 {
		limit = 50
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	// Получаем сообщения из БД
	messages, err := h.msgRepo.GetByRoomID(r.Context(), roomID, limit, offset)
	if err != nil {
		http.Error(w, "Не удалось получить сообщения", http.StatusInternalServerError)
		return
	}

	// Переворачиваем порядок (новые сверху)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	type messageWithThread struct {
		*models.Message
		ThreadCount int64 `json:"thread_count"`
	}

	enriched := make([]messageWithThread, 0, len(messages))
	for _, msg := range messages {
		tc, _ := h.msgRepo.CountThreadReplies(r.Context(), msg.ID)
		enriched = append(enriched, messageWithThread{Message: msg, ThreadCount: tc})
	}

	hasMore := len(messages) == limit

	response := map[string]interface{}{
		"data":        enriched,
		"has_more":    hasMore,
		"next_cursor": "",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateMessage POST /api/v1/messages
func (h *MessageHandler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RoomID       string          `json:"room_id"`
		Content      string          `json:"content"`
		Type         string          `json:"type"`
		ReplyToID    string          `json:"reply_to_id"`
		ThreadRootID string          `json:"thread_root_id"`
		Metadata     json.RawMessage `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректные данные запроса", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.RoomID == "" {
		http.Error(w, "Отсутствует room_id", http.StatusBadRequest)
		return
	}

	if len(req.Content) > 10000 {
		http.Error(w, "Слишком длинное содержимое (макс. 10000 символов)", http.StatusBadRequest)
		return
	}
	if req.Content == "" && req.Type != "file" && req.Type != "image" {
		http.Error(w, "Отсутствует содержимое", http.StatusBadRequest)
		return
	}

	roomID, err := uuid.Parse(req.RoomID)
	if err != nil {
		http.Error(w, "Некорректный room_id", http.StatusBadRequest)
		return
	}

	// Получаем пользователя из контекста
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		http.Error(w, "Некорректный идентификатор пользователя", http.StatusInternalServerError)
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

	// Обрабатываем metadata (file attachments)
	if req.Metadata != nil {
		var meta models.Metadata
		if err := json.Unmarshal(req.Metadata, &meta); err == nil {
			message.Metadata = meta
		}
	}

	// Обрабатываем reply_to
	if req.ReplyToID != "" {
		replyToID, err := uuid.Parse(req.ReplyToID)
		if err == nil {
			message.ReplyToID = &replyToID
		}
	}

	// Обрабатываем thread_root_id
	if req.ThreadRootID != "" {
		threadRootID, err := uuid.Parse(req.ThreadRootID)
		if err == nil {
			message.ThreadRootID = &threadRootID
		}
	}

	// Сохраняем сообщение в БД
	if err := h.msgRepo.Create(r.Context(), message); err != nil {
		http.Error(w, "Не удалось создать сообщение", http.StatusInternalServerError)
		return
	}

	// Подгружаем User для ответа
	created, _ := h.msgRepo.GetByID(r.Context(), message.ID)
	if created != nil {
		message = created
	}

	wsPayload, _ := json.Marshal(message)
	wsType := websocket.MessageTypeMessage
	if message.ThreadRootID != nil {
		wsType = websocket.MessageTypeThreadReply
	}
	h.wsHub.BroadcastToRoom(roomID.String(), websocket.WSMessage{
		Type:    wsType,
		Payload: wsPayload,
	})

	if h.botEngine != nil {
		_ = h.botEngine.HandleMessage(r.Context(), roomID.String(), userID.String(), req.Content)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(message)
}

// GetMessage GET /api/v1/messages/{id}
func (h *MessageHandler) GetMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	messageID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "Некорректный идентификатор сообщения", http.StatusBadRequest)
		return
	}

	message, err := h.msgRepo.GetByID(r.Context(), messageID)
	if err != nil {
		if err == repository.ErrMessageNotFound {
			http.Error(w, "Сообщение не найдено", http.StatusNotFound)
			return
		}
		http.Error(w, "Не удалось получить сообщение", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(message)
}

// UpdateMessage PUT /api/v1/messages/{id}
func (h *MessageHandler) UpdateMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	messageID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "Некорректный идентификатор сообщения", http.StatusBadRequest)
		return
	}

	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректные данные запроса", http.StatusBadRequest)
		return
	}

	message, err := h.msgRepo.GetByID(r.Context(), messageID)
	if err != nil {
		if err == repository.ErrMessageNotFound {
			http.Error(w, "Сообщение не найдено", http.StatusNotFound)
			return
		}
		http.Error(w, "Не удалось получить сообщение", http.StatusInternalServerError)
		return
	}

	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}
	if message.UserID.String() != claims.UserID {
		http.Error(w, "Доступ запрещён: вы не автор сообщения", http.StatusForbidden)
		return
	}

	// Обновляем содержимое
	message.Content = req.Content
	edited := true
	message.Metadata.Edited = &edited

	if err := h.msgRepo.Update(r.Context(), message); err != nil {
		http.Error(w, "Не удалось обновить сообщение", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(message)
}

// GetThread GET /api/v1/messages/{id}/thread
func (h *MessageHandler) GetThread(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rootID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "Некорректный идентификатор сообщения", http.StatusBadRequest)
		return
	}

	rootMsg, err := h.msgRepo.GetByID(r.Context(), rootID)
	if err != nil {
		if err == repository.ErrMessageNotFound {
			http.Error(w, "Сообщение не найдено", http.StatusNotFound)
			return
		}
		http.Error(w, "Не удалось получить сообщение", http.StatusInternalServerError)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	replies, err := h.msgRepo.GetThreadMessages(r.Context(), rootID, limit, offset)
	if err != nil {
		http.Error(w, "Не удалось получить ответы треда", http.StatusInternalServerError)
		return
	}

	total, _ := h.msgRepo.CountThreadReplies(r.Context(), rootID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"root":    rootMsg,
		"replies": replies,
		"total":   total,
	})
}

// DeleteMessage DELETE /api/v1/messages/{id}
func (h *MessageHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	messageID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "Некорректный идентификатор сообщения", http.StatusBadRequest)
		return
	}

	message, err := h.msgRepo.GetByID(r.Context(), messageID)
	if err != nil {
		if err == repository.ErrMessageNotFound {
			http.Error(w, "Сообщение не найдено", http.StatusNotFound)
			return
		}
		http.Error(w, "Не удалось получить сообщение", http.StatusInternalServerError)
		return
	}

	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}
	if message.UserID.String() != claims.UserID {
		http.Error(w, "Доступ запрещён: вы не автор сообщения", http.StatusForbidden)
		return
	}

	if err := h.msgRepo.Delete(r.Context(), messageID); err != nil {
		http.Error(w, "Не удалось удалить сообщение", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
