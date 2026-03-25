package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthModelSmokeAPIAndWebSocket(t *testing.T) {
	auth.ResetRevokedSessions()
	secret := []byte("smoke-secret")
	token, err := auth.GenerateSessionJWT(&auth.UserInfo{
		Sub:   "user-smoke",
		Email: "smoke@example.com",
		Name:  "Smoke User",
		Roles: []string{"user"},
	}, "smoke-session", secret, time.Hour)
	require.NoError(t, err)

	t.Run("same token accepted by api middleware and websocket auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		mw := auth.NewAuthMiddleware(secret)
		handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		wsReq := httptest.NewRequest(http.MethodGet, "/api/v1/ws?token="+token, nil)
		claims, wsErr := websocket.AuthenticateRequest(wsReq, secret)
		require.NoError(t, wsErr)
		require.NotNil(t, claims)
		assert.Equal(t, "smoke-session", claims.SessionID)
	})

	t.Run("revoked token denied for api and websocket", func(t *testing.T) {
		auth.RevokeSession("smoke-session", time.Now().Add(time.Hour))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		mw := auth.NewAuthMiddleware(secret)
		handler := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)

		wsReq := httptest.NewRequest(http.MethodGet, "/api/v1/ws?token="+token, nil)
		claims, wsErr := websocket.AuthenticateRequest(wsReq, secret)
		require.Error(t, wsErr)
		assert.Nil(t, claims)
		assert.ErrorIs(t, wsErr, websocket.ErrRevokedWebSocketToken)
	})
}
