package webhooks

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWebhookStruct(t *testing.T) {
	now := time.Now()
	webhook := Webhook{
		ID:              uuid.New(),
		OwnerID:         uuid.New(),
		URL:             "https://example.com/webhook",
		Secret:          "secret-123",
		EventTypes:      []string{"conference.created", "conference.ended"},
		IsActive:        true,
		SignatureMethod: "sha256",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	assert.Equal(t, "https://example.com/webhook", webhook.URL)
	assert.Equal(t, "secret-123", webhook.Secret)
	assert.Len(t, webhook.EventTypes, 2)
	assert.True(t, webhook.IsActive)
}

func TestWebhookTypeConstants(t *testing.T) {
	assert.Equal(t, WebhookType("jitsi"), WebhookTypeJitsi)
	assert.Equal(t, WebhookType("exchange"), WebhookTypeExchange)
	assert.Equal(t, WebhookType("custom"), WebhookTypeCustom)
}

func TestJitsiWebhookEvent(t *testing.T) {
	event := JitsiWebhookEvent{
		Event:          "conference.created",
		ConferenceName: "Test Conference",
		Room:           "room-123",
		Timestamp:      time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"creator": "user-456",
		},
	}

	assert.Equal(t, "conference.created", event.Event)
	assert.Equal(t, "Test Conference", event.ConferenceName)
	assert.Equal(t, "room-123", event.Room)
	assert.NotEmpty(t, event.Data)
}

func TestWebhookDelivery(t *testing.T) {
	now := time.Now()
	delivery := WebhookDelivery{
		ID:           uuid.New(),
		WebhookID:    uuid.New(),
		Payload:      []byte(`{"event":"test"}`),
		ResponseCode: 200,
		ResponseBody: "OK",
		Success:      true,
		RetryCount:   0,
		DeliveredAt:  &now,
		CreatedAt:    now,
	}

	assert.Equal(t, 200, delivery.ResponseCode)
	assert.True(t, delivery.Success)
	assert.Equal(t, 0, delivery.RetryCount)
}

func TestVerifySignature(t *testing.T) {
	secret := "test-secret"
	payload := `{"event":"test"}`

	signature := createSignatureForVerify(secret, []byte(payload))

	// Правильная подпись
	err := VerifySignature(secret, payload, signature)
	assert.NoError(t, err)

	// Неправильная подпись
	err = VerifySignature(secret, payload, "wrong-signature")
	assert.Error(t, err)
}

func TestCreateSignatureForVerify(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"event":"test"}`)

	sig1 := createSignatureForVerify(secret, payload)
	sig2 := createSignatureForVerify(secret, payload)

	// Подписи должны совпадать для одинаковых данных
	assert.Equal(t, sig1, sig2)
	assert.NotEmpty(t, sig1)
}

func TestCreateSignatureDifferentPayloads(t *testing.T) {
	secret := "test-secret"

	sig1 := createSignatureForVerify(secret, []byte(`{"event":"test1"}`))
	sig2 := createSignatureForVerify(secret, []byte(`{"event":"test2"}`))

	// Подписи должны отличаться для разных данных
	assert.NotEqual(t, sig1, sig2)
}

func TestWebhookHandler(t *testing.T) {
	handler := NewWebhookHandler()

	assert.NotNil(t, handler)
}

func TestHandleJitsiWebhookInvalidJSON(t *testing.T) {
	handler := NewWebhookHandler()

	ctx := context.Background()
	err := handler.HandleJitsiWebhook(ctx, []byte(`invalid json`), "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse jitsi webhook")
}

func TestHandleJitsiWebhookConferenceCreated(t *testing.T) {
	handler := NewWebhookHandler()

	ctx := context.Background()
	payload := []byte(`{
		"event": "conference.created",
		"conference_name": "Test Conference",
		"room": "room-123",
		"timestamp": "2024-01-01T12:00:00Z",
		"data": {"creator": "user-456"}
	}`)

	err := handler.HandleJitsiWebhook(ctx, payload, "")
	assert.NoError(t, err)
}

func TestHandleJitsiWebhookConferenceEnded(t *testing.T) {
	handler := NewWebhookHandler()

	ctx := context.Background()
	payload := []byte(`{
		"event": "conference.ended",
		"conference_name": "Test Conference",
		"room": "room-123",
		"timestamp": "2024-01-01T13:00:00Z"
	}`)

	err := handler.HandleJitsiWebhook(ctx, payload, "")
	assert.NoError(t, err)
}

func TestHandleJitsiWebhookUnknownEvent(t *testing.T) {
	handler := NewWebhookHandler()

	ctx := context.Background()
	payload := []byte(`{
		"event": "unknown.event",
		"room": "room-123"
	}`)

	err := handler.HandleJitsiWebhook(ctx, payload, "")
	assert.NoError(t, err)
}

func TestWebhookDispatcher(t *testing.T) {
	dispatcher := NewWebhookDispatcher()

	assert.NotNil(t, dispatcher)
}

func TestCreateSignature(t *testing.T) {
	dispatcher := NewWebhookDispatcher()

	secret := "test-secret"
	payload := []byte(`{"event":"test"}`)
	timestamp := time.Now()

	signature := dispatcher.createSignature(secret, payload, timestamp)

	assert.NotEmpty(t, signature)
}

func TestDispatchNoWebhooks(t *testing.T) {
	dispatcher := NewWebhookDispatcher()

	ctx := context.Background()
	payload := map[string]string{"event": "test"}

	// Должно вернуть nil, если нет вебхуков
	err := dispatcher.Dispatch(ctx, "test.event", payload)
	assert.NoError(t, err)
}

func TestOutgoingWebhook(t *testing.T) {
	webhook := OutgoingWebhook{
		WebhookID: uuid.New(),
		URL:       "https://example.com/webhook",
		Secret:    "secret-123",
		Payload:   []byte(`{"event":"test"}`),
		EventType: "test.event",
	}

	assert.Equal(t, "https://example.com/webhook", webhook.URL)
	assert.Equal(t, "test.event", webhook.EventType)
}
