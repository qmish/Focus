package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/webhooks"
	"gorm.io/gorm"
)

// WebhookRepository репозиторий для работы с вебхуками
type WebhookRepository struct {
	db *gorm.DB
}

// NewWebhookRepository создаёт новый WebhookRepository
func NewWebhookRepository(db *gorm.DB) *WebhookRepository {
	return &WebhookRepository{db: db}
}

// Create создаёт новый вебхук
func (r *WebhookRepository) Create(ctx context.Context, webhook *webhooks.Webhook) error {
	return r.db.WithContext(ctx).Create(webhook).Error
}

// GetByID получает вебхук по ID
func (r *WebhookRepository) GetByID(ctx context.Context, id uuid.UUID) (*webhooks.Webhook, error) {
	var webhook webhooks.Webhook
	if err := r.db.WithContext(ctx).First(&webhook, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrWebhookNotFound
		}
		return nil, err
	}
	return &webhook, nil
}

// GetByOwnerID получает вебхуки владельца
func (r *WebhookRepository) GetByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*webhooks.Webhook, error) {
	var webhooks []*webhooks.Webhook
	err := r.db.WithContext(ctx).
		Where("owner_id = ?", ownerID).
		Order("created_at DESC").
		Find(&webhooks).Error
	return webhooks, err
}

// GetActiveByEventType получает активные вебхуки для типа события
func (r *WebhookRepository) GetActiveByEventType(ctx context.Context, eventType string) ([]*webhooks.Webhook, error) {
	var webhooks []*webhooks.Webhook
	err := r.db.WithContext(ctx).
		Where("is_active = ? AND event_types @> ?", true, []string{eventType}).
		Find(&webhooks).Error
	return webhooks, err
}

// Update обновляет вебхук
func (r *WebhookRepository) Update(ctx context.Context, webhook *webhooks.Webhook) error {
	webhook.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(webhook).Error
}

// Delete удаляет вебхук
func (r *WebhookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&webhooks.Webhook{}, "id = ?", id).Error
}

// CreateDelivery создаёт запись о доставке вебхука
func (r *WebhookRepository) CreateDelivery(ctx context.Context, delivery *webhooks.WebhookDelivery) error {
	return r.db.WithContext(ctx).Create(delivery).Error
}

// GetDeliveries получает логи доставки для вебхука
func (r *WebhookRepository) GetDeliveries(ctx context.Context, webhookID uuid.UUID, limit int) ([]*webhooks.WebhookDelivery, error) {
	var deliveries []*webhooks.WebhookDelivery
	err := r.db.WithContext(ctx).
		Where("webhook_id = ?", webhookID).
		Order("created_at DESC").
		Limit(limit).
		Find(&deliveries).Error
	return deliveries, err
}
