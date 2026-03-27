package repository

import (
	"context"
	"errors"
	"time"

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

func (r *BotSettingsRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.BotSetting{}, "id = ?", id).Error
}

func (r *BotSettingsRepository) CountEvents(ctx context.Context, botName string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("bot_command_events").
		Where("command IN (SELECT jsonb_array_elements_text(commands_json::jsonb) FROM bot_settings WHERE name = ?)", botName).
		Where("created_at >= ?", since).
		Count(&count).Error
	return count, err
}
