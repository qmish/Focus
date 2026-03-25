package repository

import (
	"context"
	"time"

	"github.com/qmish/focus-api/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SessionRevocationRepository persists revoked sessions.
type SessionRevocationRepository struct {
	db *gorm.DB
}

// NewSessionRevocationRepository creates revoked session repository.
func NewSessionRevocationRepository(db *gorm.DB) *SessionRevocationRepository {
	return &SessionRevocationRepository{db: db}
}

// UpsertRevokedSession stores or updates revoked session expiration.
func (r *SessionRevocationRepository) UpsertRevokedSession(ctx context.Context, sessionID string, expiresAt time.Time) error {
	if sessionID == "" {
		return nil
	}
	entry := &models.RevokedSession{
		SessionID: sessionID,
		ExpiresAt: expiresAt.UTC(),
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "session_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"expires_at", "updated_at"}),
	}).Create(entry).Error
}

// ListActiveRevokedSessions returns non-expired revoked sessions.
func (r *SessionRevocationRepository) ListActiveRevokedSessions(ctx context.Context, now time.Time, limit int) ([]*models.RevokedSession, error) {
	if limit < 1 {
		limit = 10000
	}
	var entries []*models.RevokedSession
	err := r.db.WithContext(ctx).
		Where("expires_at > ?", now.UTC()).
		Order("expires_at ASC").
		Limit(limit).
		Find(&entries).Error
	return entries, err
}
