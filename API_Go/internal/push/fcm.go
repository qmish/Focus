package push

import (
	"context"
	"errors"

	"github.com/qmish/focus-api/internal/models"
)

// FCMSender — заглушка для Firebase Cloud Messaging.
//
// На текущем этапе мы не отправляем настоящие push в Android: реальная
// FCM-интеграция требует сервисный JSON-аккаунт и доменных согласований.
// Каркас существует, чтобы Service мог корректно «маршрутизировать»
// токены платформы fcm — настоящий клиент будет добавлен позже.
type FCMSender struct {
	enabled bool
}

// NewFCMSender создаёт заглушку. Если enabled=false, sender всегда возвращает
// ErrSenderDisabled при попытке отправки.
func NewFCMSender(enabled bool) *FCMSender {
	return &FCMSender{enabled: enabled}
}

// ErrSenderDisabled — sender не сконфигурирован/не включён.
var ErrSenderDisabled = errors.New("push: sender disabled")

// Platform возвращает fcm.
func (s *FCMSender) Platform() models.PushPlatform { return models.PushPlatformFCM }

// Send всегда возвращает ErrSenderDisabled, если не включён.
func (s *FCMSender) Send(_ context.Context, _ *models.PushToken, _ *Notification) error {
	if !s.enabled {
		return ErrSenderDisabled
	}
	// Реальный вызов FCM HTTP v1 API будет добавлен в отдельном PR.
	// На данный момент мы просто возвращаем nil, чтобы интеграционные
	// сценарии не падали в окружении со включённым FCM-флагом.
	return nil
}
