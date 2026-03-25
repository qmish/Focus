package repository

import (
	"context"

	"github.com/qmish/focus-api/internal/models"
	"gorm.io/gorm"
)

// CalendarAuditRepository persists calendar operation audit events.
type CalendarAuditRepository struct {
	db *gorm.DB
}

// NewCalendarAuditRepository creates CalendarAuditRepository.
func NewCalendarAuditRepository(db *gorm.DB) *CalendarAuditRepository {
	return &CalendarAuditRepository{db: db}
}

// CreateCalendarAuditEvent stores a calendar audit event.
func (r *CalendarAuditRepository) CreateCalendarAuditEvent(ctx context.Context, event *models.CalendarAuditEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// ListCalendarAuditEvents returns recent calendar audit events.
func (r *CalendarAuditRepository) ListCalendarAuditEvents(ctx context.Context, limit int, onlyFailed bool) ([]*models.CalendarAuditEvent, error) {
	if limit < 1 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	query := r.db.WithContext(ctx).Model(&models.CalendarAuditEvent{})
	if onlyFailed {
		query = query.Where("status <> ?", "success")
	}
	var events []*models.CalendarAuditEvent
	err := query.Order("created_at DESC").Limit(limit).Find(&events).Error
	return events, err
}
