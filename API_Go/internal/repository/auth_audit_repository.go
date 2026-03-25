package repository

import (
	"context"

	"github.com/qmish/focus-api/internal/models"
	"gorm.io/gorm"
)

// AuthAuditRepository persists auth audit events.
type AuthAuditRepository struct {
	db *gorm.DB
}

// NewAuthAuditRepository creates auth audit repository.
func NewAuthAuditRepository(db *gorm.DB) *AuthAuditRepository {
	return &AuthAuditRepository{db: db}
}

// CreateAuthAuditEvent stores auth audit event.
func (r *AuthAuditRepository) CreateAuthAuditEvent(ctx context.Context, event *models.AuthAuditEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// ListAuthAuditEvents returns recent auth audit events.
func (r *AuthAuditRepository) ListAuthAuditEvents(ctx context.Context, limit int, onlyFailed bool) ([]*models.AuthAuditEvent, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	query := r.db.WithContext(ctx).Model(&models.AuthAuditEvent{})
	if onlyFailed {
		query = query.Where("status <> ?", "success")
	}
	var events []*models.AuthAuditEvent
	err := query.Order("created_at DESC").Limit(limit).Find(&events).Error
	return events, err
}
