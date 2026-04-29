package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/search"
)

// SearchHandler — REST для глобального и локального поиска.
type SearchHandler struct {
	svc *search.Service
}

// NewSearchHandler создаёт обработчик поиска.
func NewSearchHandler(svc *search.Service) *SearchHandler {
	return &SearchHandler{svc: svc}
}

// GlobalResponse — ответ /api/v1/search.
type GlobalResponse struct {
	*search.GlobalResult
	TookMs int64  `json:"took_ms"`
	Query  string `json:"query"`
}

// LocalMessagesResponse — ответ /api/v1/rooms/{id}/messages/search.
type LocalMessagesResponse struct {
	Messages   []*search.MessageHit `json:"messages"`
	NextBefore *uuid.UUID           `json:"next_before,omitempty"`
	TookMs     int64                `json:"took_ms"`
	Query      string               `json:"query"`
}

// Global GET /api/v1/search?q=...&types=users,rooms,messages,files,meetings&limit=20
func (h *SearchHandler) Global(w http.ResponseWriter, r *http.Request) {
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

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if search.CountRunes(query) < search.MinQueryLen {
		http.Error(w, "Запрос должен быть не короче 2 символов", http.StatusBadRequest)
		return
	}

	limit := parseLimit(r.URL.Query().Get("limit"), 20, 50)
	scope := parseScope(r.URL.Query().Get("types"))

	start := time.Now()
	res, err := h.svc.Global(r.Context(), userID, query, scope, limit)
	if err != nil {
		if errors.Is(err, search.ErrEmptyQuery) {
			http.Error(w, "Запрос должен быть не короче 2 символов", http.StatusBadRequest)
			return
		}
		http.Error(w, "Ошибка поиска", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, GlobalResponse{
		GlobalResult: res,
		TookMs:       time.Since(start).Milliseconds(),
		Query:        query,
	})
}

// LocalMessages GET /api/v1/rooms/{id}/messages/search?q=...&before=...&limit=50
func (h *SearchHandler) LocalMessages(w http.ResponseWriter, r *http.Request) {
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

	roomIDStr := chi.URLParam(r, "id")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		http.Error(w, "Некорректный идентификатор комнаты", http.StatusBadRequest)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if search.CountRunes(query) < search.MinQueryLen {
		http.Error(w, "Запрос должен быть не короче 2 символов", http.StatusBadRequest)
		return
	}
	limit := parseLimit(r.URL.Query().Get("limit"), 50, 100)

	opts := search.MessageSearchOpts{Limit: limit}
	if before := strings.TrimSpace(r.URL.Query().Get("before")); before != "" {
		bid, perr := uuid.Parse(before)
		if perr != nil {
			http.Error(w, "Некорректный курсор before", http.StatusBadRequest)
			return
		}
		opts.Before = &bid
	}

	start := time.Now()
	hits, err := h.svc.LocalMessages(r.Context(), userID, roomID, query, opts)
	if err != nil {
		if errors.Is(err, search.ErrEmptyQuery) {
			http.Error(w, "Запрос должен быть не короче 2 символов", http.StatusBadRequest)
			return
		}
		http.Error(w, "Ошибка поиска", http.StatusInternalServerError)
		return
	}

	resp := LocalMessagesResponse{
		Messages: hits,
		TookMs:   time.Since(start).Milliseconds(),
		Query:    query,
	}
	if len(hits) == limit {
		last := hits[len(hits)-1]
		if last.Message != nil {
			id := last.Message.ID
			resp.NextBefore = &id
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func parseLimit(raw string, def, max int) int {
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// parseScope превращает CSV-строку "users,rooms,messages,files,meetings"
// в search.Scope. Пустая строка → DefaultScope().
func parseScope(raw string) search.Scope {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return search.DefaultScope()
	}
	var s search.Scope
	for _, part := range strings.Split(raw, ",") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "users", "people":
			s.Users = true
		case "rooms", "chats":
			s.Rooms = true
		case "messages":
			s.Messages = true
		case "files", "attachments":
			s.Files = true
		case "meetings":
			s.Meetings = true
		}
	}
	if s.IsEmpty() {
		return search.DefaultScope()
	}
	return s
}
