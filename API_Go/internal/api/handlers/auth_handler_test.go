package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/config"
	"github.com/qmish/focus-api/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLogoutRevokesSession(t *testing.T) {
	auth.ResetRevokedSessions()
	handler := newAuthHandlerForTest("test-secret")
	auditRepo := &fakeAuthAuditRepo{}
	handler.SetAuthAuditRepository(auditRepo)

	token := mustSessionTokenForLogout(t, []byte("test-secret"), "session-logout-1")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.Logout(rr, req)
	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.True(t, auth.IsSessionRevoked("session-logout-1"))
	require.NotEmpty(t, auditRepo.events)
	assert.Equal(t, "logout", auditRepo.events[0].Action)
	assert.Equal(t, "success", auditRepo.events[0].Status)
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
	auditRepo := &fakeAuthAuditRepo{}
	handler.SetAuthAuditRepository(auditRepo)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	rr := httptest.NewRecorder()
	handler.Logout(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.NotEmpty(t, auditRepo.events)
	assert.Equal(t, "failed", auditRepo.events[0].Status)
}

func TestRefreshSupportsAuthorizationHeader(t *testing.T) {
	handler := newAuthHandlerForTest("test-secret")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer refresh-token")
	rr := httptest.NewRecorder()
	handler.Refresh(rr, req)
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
}

func TestRefreshRequiresRefreshToken(t *testing.T) {
	handler := newAuthHandlerForTest("test-secret")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(`{"refresh_token":""}`))
	rr := httptest.NewRecorder()
	handler.Refresh(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestExtractBearerToken(t *testing.T) {
	t.Run("empty header", func(t *testing.T) {
		token, err := extractBearerToken("")
		assert.NoError(t, err)
		assert.Empty(t, token)
	})

	t.Run("invalid format", func(t *testing.T) {
		_, err := extractBearerToken("token")
		assert.Error(t, err)
	})

	t.Run("valid bearer", func(t *testing.T) {
		token, err := extractBearerToken("Bearer abc123")
		assert.NoError(t, err)
		assert.Equal(t, "abc123", token)
	})
}

func newAuthHandlerForTest(secret string) *AuthHandler {
	return NewAuthHandler(
		nil,
		nil,
		nil,
		&config.Config{
			Auth:  config.AuthConfig{SessionSecret: secret},
			Jitsi: config.JitsiConfig{AppSecret: "jitsi-secret-for-tests"},
		},
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

type fakeAuthAuditRepo struct {
	events []*models.AuthAuditEvent
}

func (f *fakeAuthAuditRepo) CreateAuthAuditEvent(ctx context.Context, event *models.AuthAuditEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	f.events = append(f.events, event)
	return nil
}
