package auth

import (
	"sync"
	"time"
)

type revokedSessionStore struct {
	mu       sync.RWMutex
	sessions map[string]time.Time
}

var globalRevokedSessions = &revokedSessionStore{
	sessions: make(map[string]time.Time),
}

// RevokeSession marks session ID as revoked until its expiration.
func RevokeSession(sessionID string, expiresAt time.Time) {
	if sessionID == "" {
		return
	}
	globalRevokedSessions.mu.Lock()
	defer globalRevokedSessions.mu.Unlock()
	globalRevokedSessions.sessions[sessionID] = expiresAt
}

// IsSessionRevoked checks whether session is currently revoked.
func IsSessionRevoked(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	now := time.Now()

	globalRevokedSessions.mu.RLock()
	expiresAt, ok := globalRevokedSessions.sessions[sessionID]
	globalRevokedSessions.mu.RUnlock()
	if !ok {
		return false
	}

	if !expiresAt.IsZero() && now.After(expiresAt) {
		globalRevokedSessions.mu.Lock()
		delete(globalRevokedSessions.sessions, sessionID)
		globalRevokedSessions.mu.Unlock()
		return false
	}
	return true
}

// ResetRevokedSessions clears state for tests.
func ResetRevokedSessions() {
	globalRevokedSessions.mu.Lock()
	defer globalRevokedSessions.mu.Unlock()
	globalRevokedSessions.sessions = make(map[string]time.Time)
}
