package repository

import (
	"context"
	"errors"

	"github.com/qmish/focus-api/internal/models"
	"gorm.io/gorm"
)

var ErrConferencePolicyNotFound = errors.New("conference policy not found")

type ConferencePolicyRepository struct {
	db *gorm.DB
}

func NewConferencePolicyRepository(db *gorm.DB) *ConferencePolicyRepository {
	return &ConferencePolicyRepository{db: db}
}

func (r *ConferencePolicyRepository) Get(ctx context.Context) (*models.ConferencePolicy, error) {
	var p models.ConferencePolicy
	if err := r.db.WithContext(ctx).First(&p, "id = ?", "default").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConferencePolicyNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *ConferencePolicyRepository) Upsert(ctx context.Context, p *models.ConferencePolicy) error {
	p.ID = "default"
	return r.db.WithContext(ctx).Save(p).Error
}
