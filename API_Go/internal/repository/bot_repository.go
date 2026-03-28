package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/bots"
	"github.com/qmish/focus-api/internal/models"
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

// CreateCommandEvent stores bot command execution event.
func (r *BotRepository) CreateCommandEvent(ctx context.Context, event *bots.BotCommandEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// ListCommandEvents returns recent bot command events.
func (r *BotRepository) ListCommandEvents(ctx context.Context, limit int, onlyFailed bool) ([]*bots.BotCommandEvent, error) {
	if limit < 1 {
		limit = 50
	}
	var events []*bots.BotCommandEvent
	query := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit)
	if onlyFailed {
		query = query.Where("status IN ?", []string{"failed", "permission_denied", "rate_limited"})
	}
	err := query.Find(&events).Error
	return events, err
}

// ListCommandEventsFiltered returns command events with filters for full history.
func (r *BotRepository) ListCommandEventsFiltered(ctx context.Context, limit, offset int, command, userID, roomID, status string, since time.Time) ([]*bots.BotCommandEvent, int64, error) {
	if limit < 1 {
		limit = 50
	}
	query := r.db.WithContext(ctx).Model(&bots.BotCommandEvent{})
	if command != "" {
		query = query.Where("command = ?", command)
	}
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if roomID != "" {
		query = query.Where("room_id = ?", roomID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if !since.IsZero() {
		query = query.Where("created_at >= ?", since)
	}
	var total int64
	query.Count(&total)
	var events []*bots.BotCommandEvent
	err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&events).Error
	return events, total, err
}

// ListCommandStatsGrouped returns per-command stats breakdown.
func (r *BotRepository) ListCommandStatsGrouped(ctx context.Context, since time.Time) ([]CommandStat, error) {
	var stats []CommandStat
	err := r.db.WithContext(ctx).
		Model(&bots.BotCommandEvent{}).
		Select("command, status, count(*) as count").
		Where("created_at >= ?", since).
		Group("command, status").
		Order("command, status").
		Find(&stats).Error
	return stats, err
}

// CommandStat represents a per-command stat row.
type CommandStat struct {
	Command string `json:"command"`
	Status  string `json:"status"`
	Count   int64  `json:"count"`
}

// CreateReminder creates a new bot reminder.
func (r *BotRepository) CreateReminder(ctx context.Context, reminder *models.BotReminder) error {
	return r.db.WithContext(ctx).Create(reminder).Error
}

// ListPendingReminders returns reminders not yet fired before the given time.
func (r *BotRepository) ListPendingReminders(ctx context.Context, before time.Time) ([]*models.BotReminder, error) {
	var reminders []*models.BotReminder
	err := r.db.WithContext(ctx).
		Where("fired = ? AND fire_at <= ?", false, before).
		Order("fire_at ASC").
		Limit(100).
		Find(&reminders).Error
	return reminders, err
}

// MarkFired marks a reminder as fired.
func (r *BotRepository) MarkFired(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.BotReminder{}).Where("id = ?", id).Update("fired", true).Error
}
