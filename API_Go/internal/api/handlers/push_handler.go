package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
)

// PushHandler — REST для управления подписками на push-уведомления.
type PushHandler struct {
	repo           *repository.PushTokenRepository
	vapidPublicKey string
}

// NewPushHandler создаёт обработчик. Если vapidPublicKey пустой,
// /api/v1/push/vapid-public-key вернёт 503.
func NewPushHandler(repo *repository.PushTokenRepository, vapidPublicKey string) *PushHandler {
	return &PushHandler{repo: repo, vapidPublicKey: vapidPublicKey}
}

// PublicKeyResponse — ответ на запрос VAPID-ключа.
type PublicKeyResponse struct {
	PublicKey string `json:"public_key"`
}

// GetVAPIDPublicKey GET /api/v1/push/vapid-public-key — возвращает
// VAPID-публичный ключ для PushManager.subscribe в браузере.
func (h *PushHandler) GetVAPIDPublicKey(w http.ResponseWriter, _ *http.Request) {
	if strings.TrimSpace(h.vapidPublicKey) == "" {
		http.Error(w, "Push-уведомления не настроены", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, PublicKeyResponse{PublicKey: h.vapidPublicKey})
}

// RegisterRequest — payload для регистрации подписки.
type RegisterRequest struct {
	Platform  string `json:"platform"`
	Endpoint  string `json:"endpoint"`
	P256DH    string `json:"p256dh,omitempty"`
	Auth      string `json:"auth,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	Locale    string `json:"locale,omitempty"`
}

// Register POST /api/v1/push/register — создаёт или обновляет подписку для текущего пользователя.
func (h *PushHandler) Register(w http.ResponseWriter, r *http.Request) {
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

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректное тело запроса", http.StatusBadRequest)
		return
	}
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	if platform == "" {
		platform = string(models.PushPlatformWeb)
	}
	if strings.TrimSpace(req.Endpoint) == "" {
		http.Error(w, "Поле endpoint обязательно", http.StatusBadRequest)
		return
	}
	switch models.PushPlatform(platform) {
	case models.PushPlatformWeb:
		if req.P256DH == "" || req.Auth == "" {
			http.Error(w, "Для web push требуются p256dh и auth", http.StatusBadRequest)
			return
		}
	case models.PushPlatformFCM, models.PushPlatformAPNS:
		// для мобильных ключи не нужны — endpoint уже registration token
	default:
		http.Error(w, "Неподдерживаемая платформа", http.StatusBadRequest)
		return
	}

	tok := &models.PushToken{
		UserID:    userID,
		Platform:  models.PushPlatform(platform),
		Endpoint:  req.Endpoint,
		P256DHKey: req.P256DH,
		AuthKey:   req.Auth,
		UserAgent: req.UserAgent,
		Locale:    req.Locale,
	}
	if err := h.repo.Upsert(r.Context(), tok); err != nil {
		http.Error(w, "Не удалось сохранить подписку", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"id":       tok.ID.String(),
		"endpoint": tok.Endpoint,
		"platform": string(tok.Platform),
	})
}

// UnregisterRequest — payload для отписки.
type UnregisterRequest struct {
	Endpoint string `json:"endpoint"`
}

// Unregister POST /api/v1/push/unregister — удаляет подписку.
func (h *PushHandler) Unregister(w http.ResponseWriter, r *http.Request) {
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

	var req UnregisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректное тело запроса", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Endpoint) == "" {
		http.Error(w, "Поле endpoint обязательно", http.StatusBadRequest)
		return
	}

	if err := h.repo.DeleteByUserAndEndpoint(r.Context(), userID, req.Endpoint); err != nil {
		http.Error(w, "Не удалось удалить подписку", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil && !errors.Is(err, http.ErrAbortHandler) {
		// тихо проглатываем — логирование делаем на уровне middleware
		_ = err
	}
}
