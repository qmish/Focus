package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
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
}

// SessionClaims claims для сессионного JWT
type SessionClaims struct {
	UserID     string   `json:"user_id"`
	Email      string   `json:"email"`
	Name       string   `json:"name"`
	Roles      []string `json:"roles"`
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
		KeycloakID: userInfo.Sub,
		SessionID:  sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "focus-api",
			Audience:  jwt.ClaimStrings{"focus-frontend"},
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
	secret []byte
}

// NewAuthMiddleware создаёт новый auth middleware
func NewAuthMiddleware(secret []byte) *AuthMiddleware {
	return &AuthMiddleware{secret: secret}
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

		// Добавляем claims в контекст
		ctx := context.WithValue(r.Context(), ContextKeyUserClaims, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
