package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"gorm.io/gorm"
)

var (
	ErrMeetingLinkNotFound = errors.New("meeting link not found")
)

// MeetingLinkRepository persists Focus<->Exchange mapping.
type MeetingLinkRepository struct {
	db *gorm.DB
}

func NewMeetingLinkRepository(db *gorm.DB) *MeetingLinkRepository {
	return &MeetingLinkRepository{db: db}
}

func (r *MeetingLinkRepository) Create(ctx context.Context, link *models.MeetingLink) error {
	return r.db.WithContext(ctx).Create(link).Error
}

func (r *MeetingLinkRepository) UpsertByExchangeEventID(ctx context.Context, link *models.MeetingLink) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.MeetingLink
		err := tx.Where("exchange_event_id = ?", link.ExchangeEventID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(link).Error
		}
		if err != nil {
			return err
		}
		existing.RoomID = link.RoomID
		existing.OrganizerEmail = link.OrganizerEmail
		existing.Subject = link.Subject
		existing.StartAt = link.StartAt
		existing.EndAt = link.EndAt
		existing.Status = link.Status
		existing.SyncSource = link.SyncSource
		existing.LastSyncAt = link.LastSyncAt
		return tx.Save(&existing).Error
	})
}

func (r *MeetingLinkRepository) GetByExchangeEventID(ctx context.Context, eventID string) (*models.MeetingLink, error) {
	var link models.MeetingLink
	err := r.db.WithContext(ctx).Where("exchange_event_id = ?", eventID).First(&link).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMeetingLinkNotFound
	}
	if err != nil {
		return nil, err
	}
	return &link, nil
}

func (r *MeetingLinkRepository) GetByRoomID(ctx context.Context, roomID uuid.UUID) (*models.MeetingLink, error) {
	var link models.MeetingLink
	err := r.db.WithContext(ctx).Where("room_id = ?", roomID).First(&link).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMeetingLinkNotFound
	}
	if err != nil {
		return nil, err
	}
	return &link, nil
}

func (r *MeetingLinkRepository) ListByOrganizerAndWindow(ctx context.Context, organizerEmail string, from, to time.Time, limit int) ([]*models.MeetingLink, error) {
	if limit < 1 {
		limit = 500
	}
	var links []*models.MeetingLink
	err := r.db.WithContext(ctx).
		Where("organizer_email = ? AND start_at <= ? AND end_at >= ?", organizerEmail, to, from).
		Order("start_at ASC").
		Limit(limit).
		Find(&links).Error
	return links, err
}

func (r *MeetingLinkRepository) Update(ctx context.Context, link *models.MeetingLink) error {
	link.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(link).Error
}

