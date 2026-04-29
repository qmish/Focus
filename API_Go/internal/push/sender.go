// Package push содержит реализации отправки push-уведомлений.
//
// Архитектура:
//
//   - Sender — интерфейс одного провайдера (Web Push / FCM / APNs).
//   - Service — высокоуровневый слой: загружает токены пользователя из
//     PushTokenRepository и делегирует Sender'у. Отправка идёт асинхронно
//     и устойчива к ошибкам отдельных endpoint'ов: 404/410 (Web Push)
//     приводят к удалению устаревшей подписки.
//
// На stage поднимается WebPushSender + NoopSender для FCM/APNs (заглушки).
package push

import (
	"context"
	"errors"

	"github.com/qmish/focus-api/internal/models"
)

// Notification — payload, передаваемый Sender'у.
type Notification struct {
	Title string                 `json:"title"`
	Body  string                 `json:"body"`
	Icon  string                 `json:"icon,omitempty"`
	Badge string                 `json:"badge,omitempty"`
	Tag   string                 `json:"tag,omitempty"`
	URL   string                 `json:"url,omitempty"`
	Data  map[string]interface{} `json:"data,omitempty"`
}

// SendError — ошибка отправки одному endpoint'у.
// Если IsGone == true, токен следует удалить (HTTP 404/410 от Web Push).
type SendError struct {
	Endpoint string
	IsGone   bool
	Err      error
}

func (e *SendError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return "push send error: " + e.Endpoint
	}
	return "push send error " + e.Endpoint + ": " + e.Err.Error()
}

// Sender — провайдер отправки push-сообщений на конкретный токен.
type Sender interface {
	// Platform возвращает платформу, которую обрабатывает sender.
	Platform() models.PushPlatform
	// Send отправляет уведомление. Возвращает *SendError, если хочется
	// сообщить дополнительные детали (Gone и т. п.).
	Send(ctx context.Context, token *models.PushToken, n *Notification) error
}

// ErrUnsupportedPlatform возвращается, если для платформы не зарегистрирован Sender.
var ErrUnsupportedPlatform = errors.New("push: unsupported platform")
