package push

import (
	"context"

	"github.com/qmish/focus-api/internal/models"
)

// APNSSender — заглушка для Apple Push Notification service.
//
// Полная интеграция с APNs требует ключа .p8 (Token-based authentication),
// teamID и keyID. Каркас оставлен здесь, чтобы Service мог обрабатывать
// токены платформы apns. Реальная отправка через TLS HTTP/2 будет
// реализована в отдельном PR.
type APNSSender struct {
	enabled bool
}

// NewAPNSSender создаёт заглушку.
func NewAPNSSender(enabled bool) *APNSSender {
	return &APNSSender{enabled: enabled}
}

// Platform возвращает apns.
func (s *APNSSender) Platform() models.PushPlatform { return models.PushPlatformAPNS }

// Send — пока заглушка.
func (s *APNSSender) Send(_ context.Context, _ *models.PushToken, _ *Notification) error {
	if !s.enabled {
		return ErrSenderDisabled
	}
	return nil
}
