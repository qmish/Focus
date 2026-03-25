package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/config"
	"github.com/qmish/focus-api/internal/jitsi"
	"github.com/qmish/focus-api/internal/repository"
	"go.uber.org/zap"
)

// AuthHandler обработчики для аутентификации
type AuthHandler struct {
	oidcProvider      *auth.OIDCProvider
	userRepo          *repository.UserRepository
	jitsiGen          *jitsi.TokenGenerator
	config            *config.Config
	logger            *zap.Logger
	sessionSecret     []byte
	groupPolicyMapper *auth.GroupPolicyMapper
}

// NewAuthHandler создаёт новый AuthHandler
func NewAuthHandler(
	oidcProvider *auth.OIDCProvider,
	userRepo *repository.UserRepository,
	jitsiGen *jitsi.TokenGenerator,
	cfg *config.Config,
	logger *zap.Logger,
) *AuthHandler {
	var groupPolicyMapper *auth.GroupPolicyMapper
	if cfg != nil {
		mapper, err := auth.NewGroupPolicyMapperFromJSON(cfg.Keycloak.GroupPolicyMapping)
		if err != nil {
			logger.Warn("invalid group policy mapping config", zap.Error(err))
		} else {
			groupPolicyMapper = mapper
		}
	}

	return &AuthHandler{
		oidcProvider:      oidcProvider,
		userRepo:          userRepo,
		jitsiGen:          jitsiGen,
		config:            cfg,
		logger:            logger,
		sessionSecret:     []byte(cfg.Auth.SessionSecret),
		groupPolicyMapper: groupPolicyMapper,
	}
}

// Login GET /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Генерируем state для защиты от CSRF
	state, err := auth.GenerateState()
	if err != nil {
		h.logger.Error("failed to generate state", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Сохраняем state в cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   600, // 10 минут
	})

	// Редирект на Keycloak
	authURL := h.oidcProvider.AuthURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback GET /api/v1/auth/callback
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Получаем code и state из query параметров
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		http.Error(w, "missing code parameter", http.StatusBadRequest)
		return
	}

	// Проверяем state
	cookie, err := r.Cookie("oauth_state")
	if err != nil || auth.ValidateState(cookie.Value, state) != nil {
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		return
	}

	// Очищаем cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})

	// Обмениваем code на токены
	token, err := h.oidcProvider.Exchange(ctx, code)
	if err != nil {
		h.logger.Error("token exchange failed", zap.Error(err))
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	// Получаем информацию о пользователе
	userInfo, err := h.oidcProvider.GetUserInfo(ctx, token)
	if err != nil {
		h.logger.Error("failed to get user info", zap.Error(err))
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}
	h.groupPolicyMapper.Apply(userInfo)

	h.logger.Info("user authenticated",
		zap.String("user_id", userInfo.Sub),
		zap.String("email", userInfo.Email),
	)

	// Получаем или создаём пользователя в БД
	keycloakID, err := parseUUID(userInfo.Sub)
	if err != nil {
		h.logger.Error("invalid keycloak id", zap.Error(err))
		http.Error(w, "invalid user id", http.StatusInternalServerError)
		return
	}

	user, err := h.userRepo.GetOrCreate(ctx, keycloakID, userInfo.Email, userInfo.Name)
	if err != nil {
		h.logger.Error("failed to get or create user", zap.Error(err))
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	// Обновляем время последнего входа
	if err := h.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		h.logger.Warn("failed to update last login", zap.Error(err))
	}

	// Генерируем session JWT
	sessionID, _ := generateSessionID()
	sessionJWT, err := auth.GenerateSessionJWT(userInfo, sessionID, h.sessionSecret, 24*time.Hour)
	if err != nil {
		h.logger.Error("failed to generate session jwt", zap.Error(err))
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	// Генерируем Jitsi JWT для пользователя
	jitsiJWT, err := h.jitsiGen.GenerateTokenForUser(
		"*", // room wildcard
		user.ID.String(),
		user.Name,
		user.Email,
		true, // все пользователи могут быть модераторами
	)
	if err != nil {
		h.logger.Error("failed to generate jitsi jwt", zap.Error(err))
	}

	// Возвращаем токены
	response := map[string]interface{}{
		"access_token": sessionJWT,
		"token_type":   "Bearer",
		"expires_in":   86400, // 24 часа
		"user": map[string]interface{}{
			"id":        user.ID.String(),
			"email":     user.Email,
			"name":      user.Name,
			"roles":     user.Roles,
			"jitsi_jwt": jitsiJWT,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Refresh POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := extractBearerToken(r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "invalid authorization format", http.StatusBadRequest)
		return
	}
	if refreshToken == "" {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		refreshToken = req.RefreshToken
	}

	if refreshToken == "" {
		http.Error(w, "refresh_token is required", http.StatusBadRequest)
		return
	}
	if h.oidcProvider == nil {
		http.Error(w, "auth provider unavailable", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()

	// Обновляем токен через OIDC провайдера
	newToken, err := h.oidcProvider.RefreshToken(ctx, refreshToken)
	if err != nil {
		h.logger.Error("failed to refresh token", zap.Error(err))
		http.Error(w, "failed to refresh token", http.StatusUnauthorized)
		return
	}

	// Получаем новую информацию о пользователе
	userInfo, err := h.oidcProvider.GetUserInfo(ctx, newToken)
	if err != nil {
		h.logger.Error("failed to get user info", zap.Error(err))
		http.Error(w, "failed to get user info", http.StatusInternalServerError)
		return
	}
	h.groupPolicyMapper.Apply(userInfo)

	// Генерируем новый session JWT
	sessionID, _ := generateSessionID()
	sessionJWT, err := auth.GenerateSessionJWT(userInfo, sessionID, h.sessionSecret, 24*time.Hour)
	if err != nil {
		h.logger.Error("failed to generate session jwt", zap.Error(err))
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"access_token": sessionJWT,
		"token_type":   "Bearer",
		"expires_in":   86400,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Logout POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token, err := extractBearerToken(r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "invalid authorization format", http.StatusBadRequest)
		return
	}
	if token == "" {
		http.Error(w, "missing authorization header", http.StatusBadRequest)
		return
	}

	claims, err := auth.ValidateSessionJWT(token, h.sessionSecret)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}
	auth.RevokeSession(claims.SessionID, expiresAt)

	w.WriteHeader(http.StatusNoContent)
}

// Me GET /api/v1/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"id":          claims.UserID,
		"email":       claims.Email,
		"name":        claims.Name,
		"roles":       claims.Roles,
		"keycloak_id": claims.KeycloakID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Вспомогательные функции

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func parseUUID(s string) (uuid.UUID, error) {
	// Пытаемся распарсить как UUID
	id, err := uuid.Parse(s)
	if err == nil {
		return id, nil
	}

	// Если не получилось, генерируем новый UUID на основе строки
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte(s)), nil
}

func extractBearerToken(authHeader string) (string, error) {
	if strings.TrimSpace(authHeader) == "" {
		return "", nil
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization format")
	}
	return strings.TrimSpace(parts[1]), nil
}
