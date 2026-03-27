package repository

import (
	"context"
	"time"

	"github.com/qmish/focus-api/internal/models"
	"gorm.io/gorm"
)

type AuditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, entry *models.AuditLog) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *AuditLogRepository) List(ctx context.Context, limit, offset int, actorEmail, action, resourceType string, since time.Time) ([]*models.AuditLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.AuditLog{})
	if actorEmail != "" {
		q = q.Where("actor_email ILIKE ?", "%"+actorEmail+"%")
	}
	if action != "" {
		q = q.Where("action = ?", action)
	}
	if resourceType != "" {
		q = q.Where("resource_type = ?", resourceType)
	}
	if !since.IsZero() {
		q = q.Where("created_at >= ?", since)
	}
	var total int64
	q.Count(&total)

	var entries []*models.AuditLog
	err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&entries).Error
	return entries, total, err
}
