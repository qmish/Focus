package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

var (
	ErrInvalidToken  = errors.New("invalid token")
	ErrInvalidState  = errors.New("invalid state parameter")
	ErrTokenExchange = errors.New("token exchange failed")
)

// OIDCConfig конфигурация OIDC провайдера
type OIDCConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// OIDCProvider OIDC провайдер
type OIDCProvider struct {
	Provider     *oidc.Provider
	OAuth2Config *oauth2.Config
	Verifier     *oidc.IDTokenVerifier
}

// UserInfo информация о пользователе из OIDC
type UserInfo struct {
	Sub           string   `json:"sub"`
	Email         string   `json:"email"`
	Name          string   `json:"name"`
	EmailVerified bool     `json:"email_verified"`
	Roles         []string `json:"roles"`
	Groups        []string `json:"groups,omitempty"`
	Scopes        []string `json:"scopes,omitempty"`
	Scope         string   `json:"scope,omitempty"`
	Audiences     []string `json:"audiences,omitempty"`
}

// SessionClaims claims для сессионного JWT
type SessionClaims struct {
	UserID     string   `json:"user_id"`
	Email      string   `json:"email"`
	Name       string   `json:"name"`
	Roles      []string `json:"roles"`
	Scopes     []string `json:"scopes,omitempty"`
	Scope      string   `json:"scope,omitempty"`
	KeycloakID string   `json:"keycloak_id"`
	SessionID  string   `json:"session_id"`
	jwt.RegisteredClaims
}

// NewOIDCProvider создаёт новый OIDC провайдер
func NewOIDCProvider(cfg OIDCConfig) (*OIDCProvider, error) {
	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       append(cfg.Scopes, oidc.ScopeOpenID, "profile", "email"),
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	return &OIDCProvider{
		Provider:     provider,
		OAuth2Config: oauth2Config,
		Verifier:     verifier,
	}, nil
}

// AuthURL генерирует URL для редиректа на Keycloak
func (p *OIDCProvider) AuthURL(state string) string {
	return p.OAuth2Config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "login"),
	)
}

// Exchange обменивает код авторизации на токены
func (p *OIDCProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := p.OAuth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenExchange, err)
	}
	return token, nil
}

// VerifyIDToken проверяет ID токен и извлекает информацию о пользователе
func (p *OIDCProvider) VerifyIDToken(ctx context.Context, rawIDToken string) (*UserInfo, error) {
	idToken, err := p.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	var userInfo UserInfo
	if err := idToken.Claims(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}
	var rawClaims map[string]interface{}
	if err := idToken.Claims(&rawClaims); err == nil {
		mergeKeycloakClaims(&userInfo, rawClaims, p.OAuth2Config.ClientID)
	}

	return &userInfo, nil
}

// GetUserInfo получает информацию о пользователе из OIDC
func (p *OIDCProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no id_token in response")
	}

	return p.VerifyIDToken(ctx, rawIDToken)
}

// RefreshToken обновляет access токен используя refresh токен
func (p *OIDCProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	tokenSource := p.OAuth2Config.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	})

	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return newToken, nil
}

// GenerateState генерирует state параметр для OAuth flow
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ValidateState проверяет state параметр
func ValidateState(expected, actual string) error {
	if expected != actual {
		return ErrInvalidState
	}
	return nil
}

// GenerateSessionJWT генерирует JWT для сессии
func GenerateSessionJWT(userInfo *UserInfo, sessionID string, secret []byte, lifetime time.Duration) (string, error) {
	now := time.Now()
	exp := now.Add(lifetime)

	claims := SessionClaims{
		UserID:     userInfo.Sub,
		Email:      userInfo.Email,
		Name:       userInfo.Name,
		Roles:      userInfo.Roles,
		Scopes:     userInfo.AllScopes(),
		KeycloakID: userInfo.Sub,
		SessionID:  sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "focus-api",
			Audience:  resolveAudiences(userInfo),
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func resolveAudiences(userInfo *UserInfo) jwt.ClaimStrings {
	if userInfo == nil || len(userInfo.Audiences) == 0 {
		return jwt.ClaimStrings{"focus-frontend"}
	}
	return jwt.ClaimStrings(dedupeStrings(userInfo.Audiences))
}

// ValidateSessionJWT проверяет сессионный JWT
func ValidateSessionJWT(tokenString string, secret []byte) (*SessionClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &SessionClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*SessionClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// AuthMiddleware middleware для проверки аутентификации
type AuthMiddleware struct {
	secret           []byte
	requiredAudience string
	serviceAudiences []string
	serviceScopes    []string
}

// NewAuthMiddleware создаёт новый auth middleware
func NewAuthMiddleware(secret []byte) *AuthMiddleware {
	return NewAuthMiddlewareWithPolicies(secret, "", nil, nil)
}

// NewAuthMiddlewareWithAudience creates auth middleware with required audience check.
func NewAuthMiddlewareWithAudience(secret []byte, requiredAudience string) *AuthMiddleware {
	return NewAuthMiddlewareWithPolicies(secret, requiredAudience, nil, nil)
}

// NewAuthMiddlewareWithPolicies creates auth middleware with audience/scope policies.
func NewAuthMiddlewareWithPolicies(secret []byte, requiredAudience string, serviceAudiences, serviceScopes []string) *AuthMiddleware {
	return &AuthMiddleware{
		secret:           secret,
		requiredAudience: strings.TrimSpace(requiredAudience),
		serviceAudiences: dedupeStrings(serviceAudiences),
		serviceScopes:    dedupeStrings(serviceScopes),
	}
}

// Middleware возвращает HTTP middleware для проверки JWT
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid authorization format", http.StatusUnauthorized)
			return
		}

		claims, err := ValidateSessionJWT(parts[1], m.secret)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		if !m.isAudienceAllowed(claims) {
			http.Error(w, "invalid audience", http.StatusUnauthorized)
			return
		}
		if !m.isServiceScopeValid(claims) {
			http.Error(w, "insufficient_scope", http.StatusUnauthorized)
			return
		}
		if IsSessionRevoked(claims.SessionID) {
			http.Error(w, "session revoked", http.StatusUnauthorized)
			return
		}

		// Добавляем claims в контекст
		ctx := context.WithValue(r.Context(), ContextKeyUserClaims, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) isAudienceAllowed(claims *SessionClaims) bool {
	if claims == nil {
		return false
	}
	if m.requiredAudience == "" {
		return true
	}
	if slices.Contains(claims.Audience, m.requiredAudience) {
		return true
	}
	if !hasRole(claims.Roles, "service") {
		return false
	}
	for _, audience := range m.serviceAudiences {
		if slices.Contains(claims.Audience, audience) {
			return true
		}
	}
	return false
}

func (m *AuthMiddleware) isServiceScopeValid(claims *SessionClaims) bool {
	if claims == nil || !hasRole(claims.Roles, "service") || len(m.serviceScopes) == 0 {
		return true
	}
	scopes := claims.AllScopes()
	for _, required := range m.serviceScopes {
		if !slices.Contains(scopes, required) {
			return false
		}
	}
	return true
}

// context key для хранения claims
type contextKey string

const ContextKeyUserClaims contextKey = "user_claims"

// GetUserClaimsFromContext извлекает claims из контекста
func GetUserClaimsFromContext(ctx context.Context) *SessionClaims {
	claims, ok := ctx.Value(ContextKeyUserClaims).(*SessionClaims)
	if !ok {
		return nil
	}
	return claims
}

// AllScopes returns combined scopes from both "scope" and "scopes" claims.
func (u *UserInfo) AllScopes() []string {
	scopes := parseScopeString(u.Scope)
	scopes = append(scopes, u.Scopes...)
	return dedupeStrings(scopes)
}

// AllScopes returns combined scopes from both "scope" and "scopes" claims.
func (c *SessionClaims) AllScopes() []string {
	scopes := parseScopeString(c.Scope)
	scopes = append(scopes, c.Scopes...)
	return dedupeStrings(scopes)
}

// RequireRole middleware для проверки роли
func RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserClaimsFromContext(r.Context())
			if claims == nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			hasRole := false
			for _, role := range claims.Roles {
				if role == requiredRole {
					hasRole = true
					break
				}
			}

			if !hasRole {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AccessRule задает централизованную политику проверки ролей/скоупов.
type AccessRule struct {
	AnyRoles  []string
	AllScopes []string
	AnyScopes []string
}

// RequireAccess проверяет доступ по ролям и скопам из claims.
func RequireAccess(rule AccessRule) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserClaimsFromContext(r.Context())
			if claims == nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if !hasAccess(claims, rule) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func hasAccess(claims *SessionClaims, rule AccessRule) bool {
	anyRolesOk := len(rule.AnyRoles) == 0
	if !anyRolesOk {
		for _, role := range claims.Roles {
			if slices.Contains(rule.AnyRoles, role) {
				anyRolesOk = true
				break
			}
		}
	}

	allScopesOk := true
	allScopes := claims.AllScopes()
	for _, required := range rule.AllScopes {
		if !slices.Contains(allScopes, required) {
			allScopesOk = false
			break
		}
	}

	anyScopesOk := len(rule.AnyScopes) == 0
	if !anyScopesOk {
		for _, scope := range allScopes {
			if slices.Contains(rule.AnyScopes, scope) {
				anyScopesOk = true
				break
			}
		}
	}

	// If both AnyRoles and AnyScopes are provided, allow either path.
	if len(rule.AnyRoles) > 0 && len(rule.AnyScopes) > 0 {
		return (anyRolesOk || anyScopesOk) && allScopesOk
	}
	return anyRolesOk && anyScopesOk && allScopesOk
}

func parseScopeString(scope string) []string {
	if strings.TrimSpace(scope) == "" {
		return nil
	}
	return strings.Fields(scope)
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || slices.Contains(result, trimmed) {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func mergeKeycloakClaims(userInfo *UserInfo, rawClaims map[string]interface{}, clientID string) {
	if userInfo == nil || rawClaims == nil {
		return
	}
	roles := append([]string{}, userInfo.Roles...)
	groups := append([]string{}, userInfo.Groups...)

	roles = append(roles, readNestedStringSlice(rawClaims, "realm_access", "roles")...)
	if strings.TrimSpace(clientID) != "" {
		roles = append(roles, readNestedStringSlice(rawClaims, "resource_access", clientID, "roles")...)
	}
	groups = append(groups, readStringSlice(rawClaims["groups"])...)

	if scope, ok := rawClaims["scope"].(string); ok && strings.TrimSpace(scope) != "" {
		userInfo.Scope = scope
	}

	userInfo.Roles = dedupeStrings(roles)
	userInfo.Groups = dedupeStrings(groups)
	userInfo.Scopes = userInfo.AllScopes()
	userInfo.Scope = ""
}

func readNestedStringSlice(root map[string]interface{}, path ...string) []string {
	current := interface{}(root)
	for _, key := range path {
		asMap, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		next, ok := asMap[key]
		if !ok {
			return nil
		}
		current = next
	}
	return readStringSlice(current)
}

func readStringSlice(value interface{}) []string {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		str, ok := item.(string)
		if !ok || strings.TrimSpace(str) == "" {
			continue
		}
		result = append(result, strings.TrimSpace(str))
	}
	return result
}

// GroupPolicyRule maps Keycloak/AD group to API roles/scopes.
type GroupPolicyRule struct {
	Group  string   `json:"group"`
	Roles  []string `json:"roles"`
	Scopes []string `json:"scopes"`
}

// GroupPolicyMapper applies group-based role/scope mapping.
type GroupPolicyMapper struct {
	rulesByGroup map[string]GroupPolicyRule
}

// NewGroupPolicyMapperFromJSON builds mapper from JSON array of rules.
func NewGroupPolicyMapperFromJSON(raw string) (*GroupPolicyMapper, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var rules []GroupPolicyRule
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return nil, fmt.Errorf("invalid KEYCLOAK_GROUP_POLICY_MAPPING: %w", err)
	}
	rulesByGroup := make(map[string]GroupPolicyRule, len(rules))
	for _, rule := range rules {
		group := strings.TrimSpace(rule.Group)
		if group == "" {
			continue
		}
		rule.Roles = dedupeStrings(rule.Roles)
		rule.Scopes = dedupeStrings(rule.Scopes)
		rulesByGroup[group] = rule
	}
	if len(rulesByGroup) == 0 {
		return nil, nil
	}
	return &GroupPolicyMapper{rulesByGroup: rulesByGroup}, nil
}

// Apply extends user roles/scopes according to group policies.
func (m *GroupPolicyMapper) Apply(userInfo *UserInfo) {
	if m == nil || userInfo == nil || len(userInfo.Groups) == 0 {
		return
	}
	roles := append([]string{}, userInfo.Roles...)
	scopes := userInfo.AllScopes()
	for _, group := range userInfo.Groups {
		rule, ok := m.rulesByGroup[strings.TrimSpace(group)]
		if !ok {
			continue
		}
		roles = append(roles, rule.Roles...)
		scopes = append(scopes, rule.Scopes...)
	}
	userInfo.Roles = dedupeStrings(roles)
	userInfo.Scopes = dedupeStrings(scopes)
	userInfo.Scope = ""
}
