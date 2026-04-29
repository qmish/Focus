package push

import (
	"context"
	"sync"

	"github.com/qmish/focus-api/internal/models"
)

// NoopSender — заглушка. Используется в тестах и при отключённом push-стеке.
// Считает все вызовы Send в Calls — это полезно для unit-тестов вызовов из
// CreateMessage (NotifyOfflineRoomMembers).
type NoopSender struct {
	platform models.PushPlatform
	mu       sync.Mutex
	Calls    []NoopCall
}

// NoopCall — запись об одном Send.
type NoopCall struct {
	Endpoint string
	Title    string
	Body     string
	URL      string
}

// NewNoopSender — конструктор заглушки.
func NewNoopSender(platform models.PushPlatform) *NoopSender {
	return &NoopSender{platform: platform}
}

// Platform возвращает платформу.
func (s *NoopSender) Platform() models.PushPlatform { return s.platform }

// Send просто записывает вызов и возвращает nil.
func (s *NoopSender) Send(_ context.Context, t *models.PushToken, n *Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Calls = append(s.Calls, NoopCall{
		Endpoint: t.Endpoint,
		Title:    n.Title,
		Body:     n.Body,
		URL:      n.URL,
	})
	return nil
}

// Reset — обнуляет историю вызовов.
func (s *NoopSender) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Calls = nil
}
