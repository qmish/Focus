package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLogoutRevokesSession(t *testing.T) {
	auth.ResetRevokedSessions()
	handler := newAuthHandlerForTest("test-secret")

	token := mustSessionTokenForLogout(t, []byte("test-secret"), "session-logout-1")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.Logout(rr, req)
	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.True(t, auth.IsSessionRevoked("session-logout-1"))
}

func TestLogoutRequiresAuthorizationHeader(t *testing.T) {
	handler := newAuthHandlerForTest("test-secret")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rr := httptest.NewRecorder()
	handler.Logout(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestLogoutRejectsInvalidToken(t *testing.T) {
	handler := newAuthHandlerForTest("test-secret")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	rr := httptest.NewRecorder()
	handler.Logout(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func newAuthHandlerForTest(secret string) *AuthHandler {
	return NewAuthHandler(
		nil,
		nil,
		nil,
		&config.Config{Jitsi: config.JitsiConfig{AppSecret: secret}},
		zap.NewNop(),
	)
}

func mustSessionTokenForLogout(t *testing.T, secret []byte, sessionID string) string {
	t.Helper()
	token, err := auth.GenerateSessionJWT(&auth.UserInfo{
		Sub:   "user-123",
		Email: "user@example.com",
		Name:  "User",
		Roles: []string{"user"},
	}, sessionID, secret, time.Hour)
	require.NoError(t, err)
	return token
}
