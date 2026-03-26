package handlers

import (
	"context"
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
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
	"go.uber.org/zap"
)

// AuthHandler обработчики для аутентификации
type AuthHandler struct {
	oidcProvider             *auth.OIDCProvider
	userRepo                 *repository.UserRepository
	jitsiGen                 *jitsi.TokenGenerator
	config                   *config.Config
	logger                   *zap.Logger
	sessionSecret            []byte
	sessionTokenLifetime     time.Duration
	sessionValidationSecrets [][]byte
	groupPolicyMapper        *auth.GroupPolicyMapper
	authAuditRepo            authAuditRepository
	sessionRevocationRepo    sessionRevocationRepository
}

type authAuditRepository interface {
	CreateAuthAuditEvent(ctx context.Context, event *models.AuthAuditEvent) error
}

type sessionRevocationRepository interface {
	UpsertRevokedSession(ctx context.Context, sessionID string, expiresAt time.Time) error
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
		oidcProvider:             oidcProvider,
		userRepo:                 userRepo,
		jitsiGen:                 jitsiGen,
		config:                   cfg,
		logger:                   logger,
		sessionSecret:            resolveSessionSecret(cfg),
		sessionTokenLifetime:     resolveSessionLifetime(cfg),
		sessionValidationSecrets: resolveValidationSecrets(cfg),
		groupPolicyMapper:        groupPolicyMapper,
	}
}

// SetAuthAuditRepository sets optional audit repository for auth events.
func (h *AuthHandler) SetAuthAuditRepository(repo authAuditRepository) {
	h.authAuditRepo = repo
}

// SetSessionRevocationRepository sets optional persistent session revocation repository.
func (h *AuthHandler) SetSessionRevocationRepository(repo sessionRevocationRepository) {
	h.sessionRevocationRepo = repo
}

// Login GET /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Генерируем state для защиты от CSRF
	state, err := auth.GenerateState()
	if err != nil {
		h.logger.Error("failed to generate state", zap.Error(err))
		h.recordAudit(r, "login", "failed", "", "", "state_generation_failed")
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
	h.recordAudit(r, "login", "success", "", "", "")
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback GET /api/v1/auth/callback
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Получаем code и state из query параметров
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		h.recordAudit(r, "callback", "failed", "", "", "missing_code")
		http.Error(w, "missing code parameter", http.StatusBadRequest)
		return
	}

	// Проверяем state
	cookie, err := r.Cookie("oauth_state")
	if err != nil || auth.ValidateState(cookie.Value, state) != nil {
		h.recordAudit(r, "callback", "failed", "", "", "invalid_state")
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
		h.recordAudit(r, "callback", "failed", "", "", "token_exchange_failed")
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	// Получаем информацию о пользователе
	userInfo, err := h.oidcProvider.GetUserInfo(ctx, token)
	if err != nil {
		h.logger.Error("failed to get user info", zap.Error(err))
		h.recordAudit(r, "callback", "failed", "", "", "userinfo_failed")
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
		h.recordAudit(r, "callback", "failed", userInfo.Sub, userInfo.Email, "invalid_keycloak_id")
		http.Error(w, "invalid user id", http.StatusInternalServerError)
		return
	}

	user, err := h.userRepo.GetOrCreate(ctx, keycloakID, userInfo.Email, userInfo.Name)
	if err != nil {
		h.logger.Error("failed to get or create user", zap.Error(err))
		h.recordAudit(r, "callback", "failed", userInfo.Sub, userInfo.Email, "user_create_failed")
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	// Обновляем время последнего входа
	if err := h.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		h.logger.Warn("failed to update last login", zap.Error(err))
	}

	// Генерируем session JWT
	sessionID, _ := generateSessionID()
	sessionJWT, err := auth.GenerateSessionJWT(userInfo, sessionID, h.sessionSecret, h.sessionTokenLifetime)
	if err != nil {
		h.logger.Error("failed to generate session jwt", zap.Error(err))
		h.recordAudit(r, "callback", "failed", user.ID.String(), user.Email, "session_jwt_failed")
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
	h.recordAudit(r, "callback", "success", user.ID.String(), user.Email, "")

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

// TokenExchange POST /api/v1/auth/token-exchange
// Accepts a Keycloak ID token from SPA clients and returns an API session JWT.
func (h *AuthHandler) TokenExchange(w http.ResponseWriter, r *http.Request) {
	if h.oidcProvider == nil {
		http.Error(w, "auth provider unavailable", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		http.Error(w, "missing token field", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	userInfo, err := h.oidcProvider.VerifyAccessToken(ctx, req.Token)
	if err != nil {
		h.logger.Error("token exchange: verification failed", zap.Error(err))
		h.recordAudit(r, "token_exchange", "failed", "", "", "verification_failed")
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	h.groupPolicyMapper.Apply(userInfo)

	keycloakID, err := parseUUID(userInfo.Sub)
	if err != nil {
		h.logger.Error("token exchange: invalid keycloak id", zap.Error(err))
		http.Error(w, "invalid user id", http.StatusInternalServerError)
		return
	}

	user, err := h.userRepo.GetOrCreate(ctx, keycloakID, userInfo.Email, userInfo.Name)
	if err != nil {
		h.logger.Error("token exchange: failed to get or create user", zap.Error(err))
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	if err := h.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		h.logger.Warn("token exchange: failed to update last login", zap.Error(err))
	}

	sessionID, _ := generateSessionID()
	userInfo.Sub = user.ID.String()
	sessionJWT, err := auth.GenerateSessionJWT(userInfo, sessionID, h.sessionSecret, h.sessionTokenLifetime)
	if err != nil {
		h.logger.Error("token exchange: failed to generate session jwt", zap.Error(err))
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	jitsiJWT, _ := h.jitsiGen.GenerateTokenForUser("*", user.ID.String(), user.Name, user.Email, true)
	h.recordAudit(r, "token_exchange", "success", user.ID.String(), user.Email, "")

	response := map[string]interface{}{
		"access_token": sessionJWT,
		"token_type":   "Bearer",
		"expires_in":   int(h.sessionTokenLifetime.Seconds()),
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
		h.recordAudit(r, "refresh", "failed", "", "", "invalid_authorization_format")
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
		h.recordAudit(r, "refresh", "failed", "", "", "missing_refresh_token")
		http.Error(w, "refresh_token is required", http.StatusBadRequest)
		return
	}
	if h.oidcProvider == nil {
		h.recordAudit(r, "refresh", "failed", "", "", "provider_unavailable")
		http.Error(w, "auth provider unavailable", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()

	// Обновляем токен через OIDC провайдера
	newToken, err := h.oidcProvider.RefreshToken(ctx, refreshToken)
	if err != nil {
		h.logger.Error("failed to refresh token", zap.Error(err))
		h.recordAudit(r, "refresh", "failed", "", "", "refresh_failed")
		http.Error(w, "failed to refresh token", http.StatusUnauthorized)
		return
	}

	// Получаем новую информацию о пользователе
	userInfo, err := h.oidcProvider.GetUserInfo(ctx, newToken)
	if err != nil {
		h.logger.Error("failed to get user info", zap.Error(err))
		h.recordAudit(r, "refresh", "failed", "", "", "userinfo_failed")
		http.Error(w, "failed to get user info", http.StatusInternalServerError)
		return
	}
	h.groupPolicyMapper.Apply(userInfo)

	// Генерируем новый session JWT
	sessionID, _ := generateSessionID()
	sessionJWT, err := auth.GenerateSessionJWT(userInfo, sessionID, h.sessionSecret, h.sessionTokenLifetime)
	if err != nil {
		h.logger.Error("failed to generate session jwt", zap.Error(err))
		h.recordAudit(r, "refresh", "failed", userInfo.Sub, userInfo.Email, "session_jwt_failed")
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}
	h.recordAudit(r, "refresh", "success", userInfo.Sub, userInfo.Email, "")

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
		h.recordAudit(r, "logout", "failed", "", "", "invalid_authorization_format")
		http.Error(w, "invalid authorization format", http.StatusBadRequest)
		return
	}
	if token == "" {
		h.recordAudit(r, "logout", "failed", "", "", "missing_authorization_header")
		http.Error(w, "missing authorization header", http.StatusBadRequest)
		return
	}

	claims, err := auth.ValidateSessionJWTWithSecrets(token, h.logoutValidationSecrets())
	if err != nil {
		h.recordAudit(r, "logout", "failed", "", "", "invalid_token")
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}
	auth.RevokeSession(claims.SessionID, expiresAt)
	if h.sessionRevocationRepo != nil {
		_ = h.sessionRevocationRepo.UpsertRevokedSession(r.Context(), claims.SessionID, expiresAt)
	}
	h.recordAudit(r, "logout", "success", claims.UserID, claims.Email, "")

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

func (h *AuthHandler) recordAudit(r *http.Request, action, status, userID, userEmail, reason string) {
	if h.authAuditRepo == nil || r == nil {
		return
	}
	_ = h.authAuditRepo.CreateAuthAuditEvent(r.Context(), &models.AuthAuditEvent{
		ID:        uuid.New(),
		Action:    action,
		Status:    status,
		UserID:    userID,
		UserEmail: userEmail,
		ClientIP:  firstNonEmpty(strings.TrimSpace(r.Header.Get("X-Forwarded-For")), strings.TrimSpace(r.RemoteAddr)),
		UserAgent: strings.TrimSpace(r.UserAgent()),
		Error:     reason,
		CreatedAt: time.Now().UTC(),
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func resolveSessionLifetime(cfg *config.Config) time.Duration {
	if cfg == nil || cfg.Auth.SessionTokenLifetime <= 0 {
		return 24 * time.Hour
	}
	return cfg.Auth.SessionTokenLifetime
}

func resolveSessionSecret(cfg *config.Config) []byte {
	if cfg == nil || strings.TrimSpace(cfg.Auth.SessionSecret) == "" {
		return []byte("dev-session-secret-change-me")
	}
	return []byte(cfg.Auth.SessionSecret)
}

func resolveValidationSecrets(cfg *config.Config) [][]byte {
	if cfg == nil || len(cfg.Auth.SessionValidationSecrets) == 0 {
		return nil
	}
	result := make([][]byte, 0, len(cfg.Auth.SessionValidationSecrets))
	for _, secret := range cfg.Auth.SessionValidationSecrets {
		trimmed := strings.TrimSpace(secret)
		if trimmed == "" {
			continue
		}
		result = append(result, []byte(trimmed))
	}
	return result
}

func (h *AuthHandler) logoutValidationSecrets() [][]byte {
	secrets := make([][]byte, 0, 1+len(h.sessionValidationSecrets))
	if len(h.sessionSecret) > 0 {
		secrets = append(secrets, h.sessionSecret)
	}
	secrets = append(secrets, h.sessionValidationSecrets...)
	return secrets
}
