package push

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/qmish/focus-api/internal/models"
)

// WebPushOptions — параметры VAPID и поведения отправки.
type WebPushOptions struct {
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	Subject         string        // mailto:admin@focus.local или https://...
	TTL             time.Duration // время жизни сообщения у push-сервиса
	Timeout         time.Duration // таймаут HTTP-запроса
}

// WebPushSender отправляет уведомления через стандартный Web Push (VAPID).
type WebPushSender struct {
	opts WebPushOptions
}

// NewWebPushSender создаёт sender. Возвращает ошибку, если VAPID-ключи не заданы.
func NewWebPushSender(opts WebPushOptions) (*WebPushSender, error) {
	if opts.VAPIDPublicKey == "" || opts.VAPIDPrivateKey == "" {
		return nil, fmt.Errorf("webpush: VAPID keys must be set")
	}
	if opts.Subject == "" {
		opts.Subject = "mailto:admin@focus.local"
	}
	if opts.TTL == 0 {
		opts.TTL = 24 * time.Hour
	}
	if opts.Timeout == 0 {
		opts.Timeout = 10 * time.Second
	}
	return &WebPushSender{opts: opts}, nil
}

// Platform — возвращает web.
func (s *WebPushSender) Platform() models.PushPlatform { return models.PushPlatformWeb }

// Send отправляет уведомление одному endpoint'у. При 404/410 возвращает
// *SendError{IsGone: true}, чтобы сервис мог удалить устаревшую подписку.
func (s *WebPushSender) Send(ctx context.Context, t *models.PushToken, n *Notification) error {
	if !t.IsWebPush() {
		return ErrUnsupportedPlatform
	}
	payload, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("webpush: marshal payload: %w", err)
	}

	sub := &webpush.Subscription{
		Endpoint: t.Endpoint,
		Keys: webpush.Keys{
			P256dh: t.P256DHKey,
			Auth:   t.AuthKey,
		},
	}

	options := &webpush.Options{
		Subscriber:      s.opts.Subject,
		VAPIDPublicKey:  s.opts.VAPIDPublicKey,
		VAPIDPrivateKey: s.opts.VAPIDPrivateKey,
		TTL:             int(s.opts.TTL.Seconds()),
		HTTPClient:      &http.Client{Timeout: s.opts.Timeout},
	}

	resp, err := webpush.SendNotificationWithContext(ctx, payload, sub, options)
	if err != nil {
		return &SendError{Endpoint: t.Endpoint, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		_, _ = io.Copy(io.Discard, resp.Body)
		return &SendError{
			Endpoint: t.Endpoint,
			IsGone:   true,
			Err:      fmt.Errorf("webpush: subscription expired (HTTP %d)", resp.StatusCode),
		}
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &SendError{
			Endpoint: t.Endpoint,
			Err:      fmt.Errorf("webpush: HTTP %d: %s", resp.StatusCode, string(body)),
		}
	}
	return nil
}
