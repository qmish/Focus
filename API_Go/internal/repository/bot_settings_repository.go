package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"gorm.io/gorm"
)

var ErrBotSettingNotFound = errors.New("bot setting not found")

type BotSettingsRepository struct {
	db *gorm.DB
}

func NewBotSettingsRepository(db *gorm.DB) *BotSettingsRepository {
	return &BotSettingsRepository{db: db}
}

func (r *BotSettingsRepository) List(ctx context.Context) ([]*models.BotSetting, error) {
	var settings []*models.BotSetting
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&settings).Error
	return settings, err
}

func (r *BotSettingsRepository) Create(ctx context.Context, s *models.BotSetting) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *BotSettingsRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.BotSetting, error) {
	var s models.BotSetting
	if err := r.db.WithContext(ctx).First(&s, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBotSettingNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *BotSettingsRepository) Update(ctx context.Context, s *models.BotSetting) error {
	return r.db.WithContext(ctx).Save(s).Error
}
