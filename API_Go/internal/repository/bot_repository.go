package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/bots"
	"gorm.io/gorm"
)

// BotRepository репозиторий для работы с ботами
type BotRepository struct {
	db *gorm.DB
}

// NewBotRepository создаёт новый BotRepository
func NewBotRepository(db *gorm.DB) *BotRepository {
	return &BotRepository{db: db}
}

// Create создаёт нового бота
func (r *BotRepository) Create(ctx context.Context, bot *bots.Bot) error {
	return r.db.WithContext(ctx).Create(bot).Error
}

// GetByID получает бота по ID
func (r *BotRepository) GetByID(ctx context.Context, id uuid.UUID) (*bots.Bot, error) {
	var bot bots.Bot
	if err := r.db.WithContext(ctx).First(&bot, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrBotNotFound
		}
		return nil, err
	}
	return &bot, nil
}

// GetByOwnerID получает ботов владельца
func (r *BotRepository) GetByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*bots.Bot, error) {
	var bots []*bots.Bot
	err := r.db.WithContext(ctx).
		Where("owner_id = ?", ownerID).
		Order("created_at DESC").
		Find(&bots).Error
	return bots, err
}

// GetActiveByCommand получает активных ботов с командой
func (r *BotRepository) GetActiveByCommand(ctx context.Context, command string) ([]*bots.Bot, error) {
	// Упрощённая реализация - в production использовать JSONB запросы
	var botsList []*bots.Bot
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Find(&botsList).Error
	return botsList, err
}

// Update обновляет бота
func (r *BotRepository) Update(ctx context.Context, bot *bots.Bot) error {
	return r.db.WithContext(ctx).Save(bot).Error
}

// Delete удаляет бота
func (r *BotRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&bots.Bot{}, "id = ?", id).Error
}
