package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSessionRevocationLifecycle(t *testing.T) {
	ResetRevokedSessions()

	sessionID := "session-1"
	RevokeSession(sessionID, time.Now().Add(10*time.Minute))
	assert.True(t, IsSessionRevoked(sessionID))

	ResetRevokedSessions()
	assert.False(t, IsSessionRevoked(sessionID))
}

func TestSessionRevocationExpires(t *testing.T) {
	ResetRevokedSessions()

	sessionID := "session-expired"
	RevokeSession(sessionID, time.Now().Add(-1*time.Minute))
	assert.False(t, IsSessionRevoked(sessionID))
}

func TestSessionRevocationHandlesEmptySessionID(t *testing.T) {
	ResetRevokedSessions()
	RevokeSession("", time.Now().Add(time.Hour))
	assert.False(t, IsSessionRevoked(""))
}
