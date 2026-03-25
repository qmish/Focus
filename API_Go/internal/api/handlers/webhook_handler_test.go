package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/qmish/focus-api/internal/webhooks"
	"github.com/stretchr/testify/assert"
)

func TestJitsiWebhookSignatureAndIdempotency(t *testing.T) {
	store := newMemoryIncomingStore()
	service := webhooks.NewWebhookHandlerWithConfig("jitsi-secret", store)
	handler := NewInboundWebhookHandler(service)

	payload := `{"event":"conference.created","room":"room-1","timestamp":"2026-03-25T06:00:00Z"}`
	signature := "sha256=" + signHex("jitsi-secret", []byte(payload))

	t.Run("accepts signed webhook", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/jitsi", strings.NewReader(payload))
		req.Header.Set("X-Jitsi-Signature", signature)
		req.Header.Set("X-Idempotency-Key", "evt-1")
		rr := httptest.NewRecorder()

		handler.JitsiWebhook(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), `"accepted"`)
	})

	t.Run("returns duplicate on same idempotency key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/jitsi", strings.NewReader(payload))
		req.Header.Set("X-Jitsi-Signature", signature)
		req.Header.Set("X-Idempotency-Key", "evt-1")
		rr := httptest.NewRecorder()

		handler.JitsiWebhook(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), `"duplicate"`)
	})
}

func TestJitsiWebhookRejectsInvalidSignature(t *testing.T) {
	handler := NewInboundWebhookHandler(webhooks.NewWebhookHandlerWithConfig("jitsi-secret", newMemoryIncomingStore()))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/jitsi", strings.NewReader(`{"event":"conference.created"}`))
	req.Header.Set("X-Jitsi-Signature", "sha256=invalid")
	rr := httptest.NewRecorder()

	handler.JitsiWebhook(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestJitsiWebhookRejectsInvalidPayload(t *testing.T) {
	handler := NewInboundWebhookHandler(webhooks.NewWebhookHandlerWithConfig("", newMemoryIncomingStore()))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/jitsi", strings.NewReader(`{`))
	rr := httptest.NewRecorder()

	handler.JitsiWebhook(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

type memoryIncomingStore struct {
	mu    sync.Mutex
	items map[string]bool
}

func newMemoryIncomingStore() *memoryIncomingStore {
	return &memoryIncomingStore{
		items: map[string]bool{},
	}
}

func (m *memoryIncomingStore) IsIncomingEventProcessed(ctx context.Context, source, idempotencyKey string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.items[source+":"+idempotencyKey], nil
}

func (m *memoryIncomingStore) StoreIncomingEvent(ctx context.Context, event *webhooks.IncomingEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := event.Source + ":" + event.IdempotencyKey
	if m.items[key] {
		return webhooks.ErrWebhookEventAlreadyProcessed
	}
	m.items[key] = true
	return nil
}

func signHex(secret string, payload []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
