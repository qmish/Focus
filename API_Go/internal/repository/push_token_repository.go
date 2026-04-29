package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrPushTokenNotFound возвращается, если push-токен не найден.
var ErrPushTokenNotFound = errors.New("push token not found")

// PushTokenRepository отвечает за CRUD push-токенов.
type PushTokenRepository struct {
	db *gorm.DB
}

// NewPushTokenRepository создаёт новый репозиторий.
func NewPushTokenRepository(db *gorm.DB) *PushTokenRepository {
	return &PushTokenRepository{db: db}
}

// Upsert создаёт или обновляет токен по уникальному endpoint. При совпадении
// endpoint обновляются user_id, ключи, user_agent, locale и updated_at.
// Это позволяет корректно обрабатывать смену пользователя на одном устройстве.
func (r *PushTokenRepository) Upsert(ctx context.Context, t *models.PushToken) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	now := time.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "endpoint"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"user_id", "platform", "p256dh_key", "auth_key",
				"user_agent", "locale", "updated_at",
			}),
		}).
		Create(t).Error
}

// GetByEndpoint возвращает токен по endpoint, либо ErrPushTokenNotFound.
func (r *PushTokenRepository) GetByEndpoint(ctx context.Context, endpoint string) (*models.PushToken, error) {
	var t models.PushToken
	err := r.db.WithContext(ctx).Where("endpoint = ?", endpoint).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPushTokenNotFound
		}
		return nil, err
	}
	return &t, nil
}

// ListByUser возвращает все токены пользователя.
func (r *PushTokenRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.PushToken, error) {
	var tokens []*models.PushToken
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&tokens).Error
	return tokens, err
}

// ListByUsers — массовая выборка по списку user_id для рассылки.
func (r *PushTokenRepository) ListByUsers(ctx context.Context, userIDs []uuid.UUID) ([]*models.PushToken, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	var tokens []*models.PushToken
	err := r.db.WithContext(ctx).
		Where("user_id IN ?", userIDs).
		Find(&tokens).Error
	return tokens, err
}

// DeleteByEndpoint удаляет токен по endpoint (например, при unsubscribe).
func (r *PushTokenRepository) DeleteByEndpoint(ctx context.Context, endpoint string) error {
	return r.db.WithContext(ctx).
		Where("endpoint = ?", endpoint).
		Delete(&models.PushToken{}).Error
}

// DeleteByID удаляет токен по ID.
func (r *PushTokenRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.PushToken{}, "id = ?", id).Error
}

// DeleteByUserAndEndpoint — удаление, ограниченное владельцем (защита от удаления чужой подписки).
func (r *PushTokenRepository) DeleteByUserAndEndpoint(ctx context.Context, userID uuid.UUID, endpoint string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND endpoint = ?", userID, endpoint).
		Delete(&models.PushToken{}).Error
}

// TouchLastUsed обновляет last_used_at у токена. Используется после успешной отправки push.
func (r *PushTokenRepository) TouchLastUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.PushToken{}).
		Where("id = ?", id).
		Update("last_used_at", now).Error
}
