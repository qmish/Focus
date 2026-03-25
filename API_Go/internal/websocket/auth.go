package websocket

import (
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/qmish/focus-api/internal/auth"
)

var (
	ErrMissingWebSocketToken = errors.New("missing websocket auth token")
	ErrInvalidWebSocketToken = errors.New("invalid websocket auth token")
	ErrExpiredWebSocketToken = errors.New("expired websocket auth token")
)

// AuthenticateRequest validates auth for websocket upgrade requests.
// It supports Authorization: Bearer <token> and fallback query params:
// - token
// - access_token
func AuthenticateRequest(r *http.Request, secret []byte) (*auth.SessionClaims, error) {
	token := extractToken(r)
	if token == "" {
		return nil, ErrMissingWebSocketToken
	}

	claims, err := auth.ValidateSessionJWT(token, secret)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) || strings.Contains(strings.ToLower(err.Error()), "expired") {
			return nil, ErrExpiredWebSocketToken
		}
		return nil, ErrInvalidWebSocketToken
	}

	return claims, nil
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
		return token
	}

	if token := strings.TrimSpace(r.URL.Query().Get("access_token")); token != "" {
		return token
	}

	return ""
}
