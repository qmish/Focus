package repository

import (
	"context"
	"errors"

	"github.com/qmish/focus-api/internal/models"
	"gorm.io/gorm"
)

var ErrAppSettingNotFound = errors.New("app setting not found")

type AppSettingsRepository struct {
	db *gorm.DB
}

func NewAppSettingsRepository(db *gorm.DB) *AppSettingsRepository {
	return &AppSettingsRepository{db: db}
}

func (r *AppSettingsRepository) Get(ctx context.Context) (*models.AppSetting, error) {
	var s models.AppSetting
	if err := r.db.WithContext(ctx).First(&s, "id = ?", "default").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAppSettingNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *AppSettingsRepository) Upsert(ctx context.Context, s *models.AppSetting) error {
	s.ID = "default"
	return r.db.WithContext(ctx).Save(s).Error
}
