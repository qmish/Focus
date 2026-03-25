package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/qmish/focus-api/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticateRequest(t *testing.T) {
	secret := []byte("test-secret-key-12345")
	validToken := mustSessionToken(t, secret)

	t.Run("accepts bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)

		claims, err := AuthenticateRequest(req, secret)
		require.NoError(t, err)
		require.NotNil(t, claims)
		assert.Equal(t, "user-123", claims.UserID)
	})

	t.Run("accepts token query param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/ws?token="+validToken, nil)

		claims, err := AuthenticateRequest(req, secret)
		require.NoError(t, err)
		require.NotNil(t, claims)
		assert.Equal(t, "user-123", claims.UserID)
	})

	t.Run("accepts access_token query param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/ws?access_token="+validToken, nil)

		claims, err := AuthenticateRequest(req, secret)
		require.NoError(t, err)
		require.NotNil(t, claims)
		assert.Equal(t, "user-123", claims.UserID)
	})

	t.Run("rejects request without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil)

		claims, err := AuthenticateRequest(req, secret)
		require.Error(t, err)
		assert.Nil(t, claims)
		assert.ErrorIs(t, err, ErrMissingWebSocketToken)
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/ws?token=not-a-jwt", nil)

		claims, err := AuthenticateRequest(req, secret)
		require.Error(t, err)
		assert.Nil(t, claims)
		assert.ErrorIs(t, err, ErrInvalidWebSocketToken)
	})

	t.Run("rejects expired token with explicit error", func(t *testing.T) {
		expiredToken := mustSessionTokenWithLifetime(t, secret, -1*time.Minute)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/ws?token="+expiredToken, nil)

		claims, err := AuthenticateRequest(req, secret)
		require.Error(t, err)
		assert.Nil(t, claims)
		assert.ErrorIs(t, err, ErrExpiredWebSocketToken)
	})
}

func mustSessionToken(t *testing.T, secret []byte) string {
	t.Helper()
	return mustSessionTokenWithLifetime(t, secret, time.Hour)
}

func mustSessionTokenWithLifetime(t *testing.T, secret []byte, lifetime time.Duration) string {
	t.Helper()
	token, err := auth.GenerateSessionJWT(&auth.UserInfo{
		Sub:   "user-123",
		Email: "user@example.com",
		Name:  "Test User",
		Roles: []string{"user"},
	}, "session-123", secret, lifetime)
	require.NoError(t, err)
	return token
}
