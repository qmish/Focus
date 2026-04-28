package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/bots"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
	"github.com/qmish/focus-api/internal/websocket"
)

var mentionRegex = regexp.MustCompile(`@(\w+)`)

// MessageHandler обработчики для messages
type MessageHandler struct {
	msgRepo    *repository.MessageRepository
	userRepo   *repository.UserRepository
	roomRepo   *repository.RoomRepository
	wsHub      *websocket.Hub
	botEngine  *bots.BotEngine
	editWindow time.Duration
}

// NewMessageHandler создаёт новый MessageHandler
func NewMessageHandler(msgRepo *repository.MessageRepository, userRepo *repository.UserRepository, wsHub *websocket.Hub, botEngine *bots.BotEngine) *MessageHandler {
	return &MessageHandler{
		msgRepo:   msgRepo,
		userRepo:  userRepo,
		wsHub:     wsHub,
		botEngine: botEngine,
	}
}

// SetRoomRepository устанавливает репозиторий комнат для проверки ролей участников.
func (h *MessageHandler) SetRoomRepository(roomRepo *repository.RoomRepository) {
	h.roomRepo = roomRepo
}

// SetEditWindow задаёт временное окно для редактирования собственных сообщений.
// duration <= 0 — без ограничения (редактировать можно в любое время).
func (h *MessageHandler) SetEditWindow(window time.Duration) {
	h.editWindow = window
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

	type messageWithExtras struct {
		*models.Message
		ThreadCount      int64             `json:"thread_count"`
		ReactionsSummary []ReactionSummary `json:"reactions_summary"`
	}

	enriched := make([]messageWithExtras, 0, len(messages))
	for _, msg := range messages {
		tc, _ := h.msgRepo.CountThreadReplies(r.Context(), msg.ID)
		var summary []ReactionSummary
		if len(msg.Reactions) > 0 {
			deref := make([]models.MessageReaction, len(msg.Reactions))
			for i, r := range msg.Reactions {
				deref[i] = r
			}
			summary = AggregateReactions(deref)
		}
		enriched = append(enriched, messageWithExtras{Message: msg, ThreadCount: tc, ReactionsSummary: summary})
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

	// Парсинг @mentions
	if h.userRepo != nil && req.Content != "" {
		matches := mentionRegex.FindAllStringSubmatch(req.Content, -1)
		if len(matches) > 0 {
			names := make([]string, 0, len(matches))
			seen := make(map[string]bool)
			for _, m := range matches {
				name := m[1]
				if !seen[name] {
					seen[name] = true
					names = append(names, name)
				}
			}
			mentionedUsers, err := h.userRepo.FindByNames(r.Context(), names)
			if err == nil && len(mentionedUsers) > 0 {
				mentionIDs := make([]string, 0, len(mentionedUsers))
				for _, u := range mentionedUsers {
					mentionIDs = append(mentionIDs, u.ID.String())
				}
				message.Metadata.Mentions = mentionIDs
				_ = h.msgRepo.Update(r.Context(), message)
			}
		}
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

	// Отправка персональных WS-уведомлений об упоминании
	if len(message.Metadata.Mentions) > 0 {
		contentPreview := req.Content
		if len(contentPreview) > 100 {
			contentPreview = contentPreview[:100] + "..."
		}
		for _, mentionedUID := range message.Metadata.Mentions {
			if mentionedUID == userID.String() {
				continue
			}
			mentionPayload, _ := json.Marshal(map[string]string{
				"room_id":         roomID.String(),
				"message_id":      message.ID.String(),
				"mentioned_by":    userID.String(),
				"content_preview": contentPreview,
			})
			h.wsHub.SendToUser(mentionedUID, websocket.WSMessage{
				Type:    websocket.MessageTypeMention,
				Payload: mentionPayload,
			})
		}
	}

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

	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректные данные запроса", http.StatusBadRequest)
		return
	}
	if req.Content == "" {
		http.Error(w, "Отсутствует содержимое", http.StatusBadRequest)
		return
	}
	if len(req.Content) > 10000 {
		http.Error(w, "Слишком длинное содержимое (макс. 10000 символов)", http.StatusBadRequest)
		return
	}

	message, err := h.msgRepo.GetByID(r.Context(), messageID)
	if err != nil {
		if errors.Is(err, repository.ErrMessageNotFound) {
			http.Error(w, "Сообщение не найдено", http.StatusNotFound)
			return
		}
		http.Error(w, "Не удалось получить сообщение", http.StatusInternalServerError)
		return
	}
	if message.IsDeleted {
		http.Error(w, "Сообщение удалено", http.StatusGone)
		return
	}
	if message.UserID.String() != claims.UserID {
		http.Error(w, "Доступ запрещён: вы не автор сообщения", http.StatusForbidden)
		return
	}

	if h.editWindow > 0 && time.Since(message.CreatedAt) > h.editWindow {
		http.Error(w, "Истёк срок редактирования сообщения", http.StatusGone)
		return
	}

	editorID, err := uuid.Parse(claims.UserID)
	if err != nil {
		http.Error(w, "Некорректный идентификатор пользователя", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	edited := true
	message.Content = req.Content
	message.Metadata.Edited = &edited
	message.Metadata.EditedAt = &now
	message.Metadata.EditedBy = &editorID

	if err := h.msgRepo.Update(r.Context(), message); err != nil {
		http.Error(w, "Не удалось обновить сообщение", http.StatusInternalServerError)
		return
	}

	if reloaded, errReload := h.msgRepo.GetByID(r.Context(), messageID); errReload == nil && reloaded != nil {
		message = reloaded
	}

	if h.wsHub != nil {
		wsPayload, _ := json.Marshal(message)
		h.wsHub.BroadcastToRoom(message.RoomID.String(), websocket.WSMessage{
			Type:    websocket.MessageTypeMessageUpdated,
			Payload: wsPayload,
		})
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

	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	message, err := h.msgRepo.GetByID(r.Context(), messageID)
	if err != nil {
		if errors.Is(err, repository.ErrMessageNotFound) {
			http.Error(w, "Сообщение не найдено", http.StatusNotFound)
			return
		}
		http.Error(w, "Не удалось получить сообщение", http.StatusInternalServerError)
		return
	}
	if message.IsDeleted {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	requesterID, err := uuid.Parse(claims.UserID)
	if err != nil {
		http.Error(w, "Некорректный идентификатор пользователя", http.StatusInternalServerError)
		return
	}

	if !h.canDeleteMessage(r.Context(), claims, requesterID, message) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}

	if err := h.msgRepo.Delete(r.Context(), messageID); err != nil {
		http.Error(w, "Не удалось удалить сообщение", http.StatusInternalServerError)
		return
	}

	if h.wsHub != nil {
		payload, _ := json.Marshal(map[string]string{
			"message_id":     messageID.String(),
			"room_id":        message.RoomID.String(),
			"thread_root_id": optionalUUIDString(message.ThreadRootID),
			"deleted_by":     requesterID.String(),
		})
		h.wsHub.BroadcastToRoom(message.RoomID.String(), websocket.WSMessage{
			Type:    websocket.MessageTypeMessageDeleted,
			Payload: payload,
		})
	}

	w.WriteHeader(http.StatusNoContent)
}

// canDeleteMessage реализует гибрид-авторизацию удаления:
// 1) автор всегда может удалить своё сообщение;
// 2) глобальный admin (claims.Roles содержит "admin") — всегда;
// 3) иначе — moderator/admin в RoomParticipant для этой комнаты.
func (h *MessageHandler) canDeleteMessage(ctx context.Context, claims *auth.SessionClaims, requesterID uuid.UUID, message *models.Message) bool {
	if message.UserID == requesterID {
		return true
	}
	if slices.Contains(claims.Roles, "admin") {
		return true
	}
	if h.roomRepo == nil {
		return false
	}
	participant, err := h.roomRepo.GetParticipant(ctx, message.RoomID, requesterID)
	if err != nil || participant == nil {
		return false
	}
	switch participant.Role {
	case models.ParticipantRoleAdmin, models.ParticipantRoleModerator:
		return true
	}
	return false
}

func optionalUUIDString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
