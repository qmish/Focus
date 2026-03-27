package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"gorm.io/gorm"
)

var ErrAdminInviteNotFound = errors.New("admin invite not found")

type AdminInviteRepository struct {
	db *gorm.DB
}

func NewAdminInviteRepository(db *gorm.DB) *AdminInviteRepository {
	return &AdminInviteRepository{db: db}
}

func (r *AdminInviteRepository) Create(ctx context.Context, invite *models.AdminInvite) error {
	return r.db.WithContext(ctx).Create(invite).Error
}

func (r *AdminInviteRepository) List(ctx context.Context, limit, offset int) ([]*models.AdminInvite, error) {
	var invites []*models.AdminInvite
	err := r.db.WithContext(ctx).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&invites).Error
	return invites, err
}

func (r *AdminInviteRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.AdminInvite, error) {
	var invite models.AdminInvite
	if err := r.db.WithContext(ctx).First(&invite, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminInviteNotFound
		}
		return nil, err
	}
	return &invite, nil
}

func (r *AdminInviteRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.AdminInvite, error) {
	var invite models.AdminInvite
	if err := r.db.WithContext(ctx).First(&invite, "token_hash = ?", tokenHash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminInviteNotFound
		}
		return nil, err
	}
	return &invite, nil
}

func (r *AdminInviteRepository) Update(ctx context.Context, invite *models.AdminInvite) error {
	return r.db.WithContext(ctx).Save(invite).Error
}

func (r *AdminInviteRepository) MarkSent(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&models.AdminInvite{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":  models.AdminInviteStatusSent,
			"sent_at": &now,
		}).Error
}
