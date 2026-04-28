package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
	"github.com/qmish/focus-api/internal/websocket"
)

// ReactionHandler обработчики для реакций на сообщения
type ReactionHandler struct {
	msgRepo *repository.MessageRepository
	wsHub   *websocket.Hub
}

// NewReactionHandler создаёт новый ReactionHandler
func NewReactionHandler(msgRepo *repository.MessageRepository, wsHub *websocket.Hub) *ReactionHandler {
	return &ReactionHandler{msgRepo: msgRepo, wsHub: wsHub}
}

// AddReaction POST /api/v1/messages/{id}/reactions
func (h *ReactionHandler) AddReaction(w http.ResponseWriter, r *http.Request) {
	messageIDStr := chi.URLParam(r, "id")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		http.Error(w, "Некорректный идентификатор сообщения", http.StatusBadRequest)
		return
	}

	var req struct {
		Emoji string `json:"emoji"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Emoji == "" {
		http.Error(w, "Отсутствует emoji", http.StatusBadRequest)
		return
	}

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

	msg, err := h.msgRepo.GetByID(r.Context(), messageID)
	if err != nil {
		if err == repository.ErrMessageNotFound {
			http.Error(w, "Сообщение не найдено", http.StatusNotFound)
			return
		}
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	reaction := models.NewMessageReaction(messageID, userID, req.Emoji)
	if err := h.msgRepo.AddReaction(r.Context(), reaction); err != nil {
		http.Error(w, "Не удалось добавить реакцию", http.StatusInternalServerError)
		return
	}

	wsPayload, _ := json.Marshal(map[string]string{
		"message_id": messageID.String(),
		"room_id":    msg.RoomID.String(),
		"user_id":    userID.String(),
		"emoji":      req.Emoji,
	})
	h.wsHub.BroadcastToRoom(msg.RoomID.String(), websocket.WSMessage{
		Type:    websocket.MessageTypeReactionAdded,
		Payload: wsPayload,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(reaction)
}

// RemoveReaction DELETE /api/v1/messages/{id}/reactions/{emoji}
func (h *ReactionHandler) RemoveReaction(w http.ResponseWriter, r *http.Request) {
	messageIDStr := chi.URLParam(r, "id")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		http.Error(w, "Некорректный идентификатор сообщения", http.StatusBadRequest)
		return
	}

	emoji := chi.URLParam(r, "emoji")
	if emoji == "" {
		http.Error(w, "Отсутствует emoji", http.StatusBadRequest)
		return
	}

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

	msg, err := h.msgRepo.GetByID(r.Context(), messageID)
	if err != nil {
		if err == repository.ErrMessageNotFound {
			http.Error(w, "Сообщение не найдено", http.StatusNotFound)
			return
		}
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	if err := h.msgRepo.RemoveReaction(r.Context(), messageID, userID, emoji); err != nil {
		http.Error(w, "Не удалось удалить реакцию", http.StatusInternalServerError)
		return
	}

	wsPayload, _ := json.Marshal(map[string]string{
		"message_id": messageID.String(),
		"room_id":    msg.RoomID.String(),
		"user_id":    userID.String(),
		"emoji":      emoji,
	})
	h.wsHub.BroadcastToRoom(msg.RoomID.String(), websocket.WSMessage{
		Type:    websocket.MessageTypeReactionRemoved,
		Payload: wsPayload,
	})

	w.WriteHeader(http.StatusNoContent)
}

// ReactionSummary агрегированная реакция
type ReactionSummary struct {
	Emoji   string   `json:"emoji"`
	Count   int      `json:"count"`
	UserIDs []string `json:"user_ids"`
}

// AggregateReactions группирует реакции по emoji
func AggregateReactions(reactions []models.MessageReaction) []ReactionSummary {
	emojiMap := make(map[string]*ReactionSummary)
	order := make([]string, 0)
	for _, r := range reactions {
		if _, ok := emojiMap[r.Emoji]; !ok {
			emojiMap[r.Emoji] = &ReactionSummary{Emoji: r.Emoji, UserIDs: []string{}}
			order = append(order, r.Emoji)
		}
		s := emojiMap[r.Emoji]
		s.Count++
		s.UserIDs = append(s.UserIDs, r.UserID.String())
	}
	result := make([]ReactionSummary, 0, len(order))
	for _, emoji := range order {
		result = append(result, *emojiMap[emoji])
	}
	return result
}

// ListReactions GET /api/v1/messages/{id}/reactions
func (h *ReactionHandler) ListReactions(w http.ResponseWriter, r *http.Request) {
	messageIDStr := chi.URLParam(r, "id")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		http.Error(w, "Некорректный идентификатор сообщения", http.StatusBadRequest)
		return
	}

	reactions, err := h.msgRepo.GetReactions(r.Context(), messageID)
	if err != nil {
		http.Error(w, "Не удалось получить реакции", http.StatusInternalServerError)
		return
	}

	deref := make([]models.MessageReaction, len(reactions))
	for i, r := range reactions {
		deref[i] = *r
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AggregateReactions(deref))
}
