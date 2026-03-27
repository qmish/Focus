package repository

import (
	"context"
	"errors"

	"github.com/qmish/focus-api/internal/models"
	"gorm.io/gorm"
)

var ErrExchangeSettingNotFound = errors.New("exchange settings not found")

type ExchangeSettingsRepository struct {
	db *gorm.DB
}

func NewExchangeSettingsRepository(db *gorm.DB) *ExchangeSettingsRepository {
	return &ExchangeSettingsRepository{db: db}
}

func (r *ExchangeSettingsRepository) Get(ctx context.Context) (*models.ExchangeSetting, error) {
	var s models.ExchangeSetting
	if err := r.db.WithContext(ctx).First(&s, "id = ?", "default").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExchangeSettingNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *ExchangeSettingsRepository) Upsert(ctx context.Context, s *models.ExchangeSetting) error {
	s.ID = "default"
	return r.db.WithContext(ctx).Save(s).Error
}
