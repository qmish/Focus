package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/qmish/focus-api/internal/models"
	"gorm.io/gorm"
)

var ErrCalendarIdempotencyKeyNotFound = errors.New("calendar idempotency key not found")

type CalendarIdempotencyRepository struct {
	db *gorm.DB
}

func NewCalendarIdempotencyRepository(db *gorm.DB) *CalendarIdempotencyRepository {
	return &CalendarIdempotencyRepository{db: db}
}

func (r *CalendarIdempotencyRepository) CreatePending(ctx context.Context, key, userEmail string) error {
	record := &models.CalendarIdempotencyKey{
		Key:       strings.TrimSpace(key),
		UserEmail: strings.TrimSpace(strings.ToLower(userEmail)),
	}
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *CalendarIdempotencyRepository) Get(ctx context.Context, key, userEmail string) (*models.CalendarIdempotencyKey, error) {
	var record models.CalendarIdempotencyKey
	err := r.db.WithContext(ctx).Where("key = ? AND user_email = ?", strings.TrimSpace(key), strings.TrimSpace(strings.ToLower(userEmail))).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCalendarIdempotencyKeyNotFound
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *CalendarIdempotencyRepository) MarkCompleted(ctx context.Context, key, userEmail, eventID, roomID, responseBody string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&models.CalendarIdempotencyKey{}).
		Where("key = ? AND user_email = ?", strings.TrimSpace(key), strings.TrimSpace(strings.ToLower(userEmail))).
		Updates(map[string]any{
			"event_id":      strings.TrimSpace(eventID),
			"room_id":       strings.TrimSpace(roomID),
			"response_body": responseBody,
			"completed_at":  &now,
			"updated_at":    now,
		}).Error
}
