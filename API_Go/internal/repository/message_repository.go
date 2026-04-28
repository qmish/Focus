package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"gorm.io/gorm"
)

// MessageRepository репозиторий для работы с сообщениями
type MessageRepository struct {
	db *gorm.DB
}

// NewMessageRepository создаёт новый MessageRepository
func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create создаёт новое сообщение
func (r *MessageRepository) Create(ctx context.Context, message *models.Message) error {
	if err := r.db.WithContext(ctx).Create(message).Error; err != nil {
		return err
	}
	return nil
}

// GetByID получает сообщение по ID
func (r *MessageRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Message, error) {
	var message models.Message
	if err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Room").
		First(&message, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}
	return &message, nil
}

// GetByRoomID получает сообщения комнаты (без thread-ответов)
func (r *MessageRepository) GetByRoomID(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]*models.Message, error) {
	var messages []*models.Message
	err := r.db.WithContext(ctx).
		Where("room_id = ? AND is_deleted = ? AND thread_root_id IS NULL", roomID, false).
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	return messages, err
}

// GetByRoomIDWithCursor получает сообщения комнаты с курсором (без thread-ответов)
func (r *MessageRepository) GetByRoomIDWithCursor(ctx context.Context, roomID uuid.UUID, cursor string, limit int) ([]*models.Message, error) {
	query := r.db.WithContext(ctx).
		Where("room_id = ? AND is_deleted = ? AND thread_root_id IS NULL", roomID, false).
		Preload("User").
		Order("created_at DESC").
		Limit(limit)

	if cursor != "" {
		cursorID, err := uuid.Parse(cursor)
		if err == nil {
			query = query.Where("created_at < (SELECT created_at FROM messages WHERE id = ?)", cursorID)
		}
	}

	var messages []*models.Message
	err := query.Find(&messages).Error
	return messages, err
}

// Update обновляет сообщение
func (r *MessageRepository) Update(ctx context.Context, message *models.Message) error {
	message.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(message).Error
}

// Delete удаляет сообщение (мягкое удаление)
func (r *MessageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.Message{}).
		Where("id = ?", id).
		Update("is_deleted", true).Error
}

// AddReaction добавляет реакцию на сообщение
func (r *MessageRepository) AddReaction(ctx context.Context, reaction *models.MessageReaction) error {
	return r.db.WithContext(ctx).Create(reaction).Error
}

// RemoveReaction удаляет реакцию с сообщения
func (r *MessageRepository) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	return r.db.WithContext(ctx).Where("message_id = ? AND user_id = ? AND emoji = ?", messageID, userID, emoji).
		Delete(&models.MessageReaction{}).Error
}

// GetReactions получает реакции сообщения
func (r *MessageRepository) GetReactions(ctx context.Context, messageID uuid.UUID) ([]*models.MessageReaction, error) {
	var reactions []*models.MessageReaction
	err := r.db.WithContext(ctx).Where("message_id = ?", messageID).Find(&reactions).Error
	return reactions, err
}

// Count возвращает количество сообщений в комнате
func (r *MessageRepository) Count(ctx context.Context, roomID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Message{}).
		Where("room_id = ? AND is_deleted = ?", roomID, false).
		Count(&count).Error
	return count, err
}

// CountSince returns non-deleted messages count since specified time.
func (r *MessageRepository) CountSince(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Message{}).
		Where("is_deleted = ? AND created_at >= ?", false, since).
		Count(&count).Error
	return count, err
}

// DayMessageCount holds a date and its message count.
type DayMessageCount struct {
	Date  string `gorm:"column:date" json:"date"`
	Count int64  `gorm:"column:count" json:"count"`
}

// CountByDay returns per-day message counts since the given time using a single GROUP BY query.
func (r *MessageRepository) CountByDay(ctx context.Context, since time.Time) ([]DayMessageCount, error) {
	var results []DayMessageCount
	err := r.db.WithContext(ctx).
		Model(&models.Message{}).
		Select("DATE(created_at) AS date, COUNT(*) AS count").
		Where("is_deleted = ? AND created_at >= ?", false, since).
		Group("DATE(created_at)").
		Order("date ASC").
		Find(&results).Error
	return results, err
}

// GetLastMessage получает последнее сообщение в комнате
func (r *MessageRepository) GetLastMessage(ctx context.Context, roomID uuid.UUID) (*models.Message, error) {
	var message models.Message
	err := r.db.WithContext(ctx).
		Where("room_id = ? AND is_deleted = ?", roomID, false).
		Preload("User").
		Order("created_at DESC").
		First(&message).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &message, nil
}

// MarkAsDeleted помечает сообщение как удалённое
func (r *MessageRepository) MarkAsDeleted(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Message{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"updated_at": now,
		}).Error
}

// Search ищет сообщения по содержимому
func (r *MessageRepository) Search(ctx context.Context, roomID uuid.UUID, query string, limit int) ([]*models.Message, error) {
	var messages []*models.Message
	searchPattern := "%" + query + "%"
	err := r.db.WithContext(ctx).
		Where("room_id = ? AND is_deleted = ? AND content ILIKE ?", roomID, false, searchPattern).
		Preload("User").
		Limit(limit).
		Order("created_at DESC").
		Find(&messages).Error
	return messages, err
}

// GetThreadMessages возвращает ответы в треде
func (r *MessageRepository) GetThreadMessages(ctx context.Context, rootID uuid.UUID, limit, offset int) ([]*models.Message, error) {
	var messages []*models.Message
	err := r.db.WithContext(ctx).
		Where("thread_root_id = ? AND is_deleted = ?", rootID, false).
		Preload("User").
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	return messages, err
}

// CountThreadReplies возвращает количество ответов в треде
func (r *MessageRepository) CountThreadReplies(ctx context.Context, rootID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Message{}).
		Where("thread_root_id = ? AND is_deleted = ?", rootID, false).
		Count(&count).Error
	return count, err
}
