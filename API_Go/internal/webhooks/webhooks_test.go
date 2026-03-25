package webhooks

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
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

func TestHandleJitsiWebhookRequiresSignatureWhenSecretConfigured(t *testing.T) {
	handler := NewWebhookHandlerWithConfig("test-secret", nil)
	ctx := context.Background()
	payload := []byte(`{"event":"conference.created","room":"room-123"}`)

	err := handler.HandleJitsiWebhook(ctx, payload, "")
	assert.ErrorIs(t, err, ErrMissingWebhookSignature)
}

func TestHandleJitsiWebhookIdempotencyByPayloadHash(t *testing.T) {
	store := &testIncomingStore{items: map[string]bool{}}
	handler := NewWebhookHandlerWithConfig("", store)
	ctx := context.Background()
	payload := []byte(`{"event":"conference.created","room":"room-123"}`)

	err := handler.HandleJitsiWebhookWithIdempotency(ctx, payload, "", "")
	assert.NoError(t, err)

	err = handler.HandleJitsiWebhookWithIdempotency(ctx, payload, "", "")
	assert.ErrorIs(t, err, ErrWebhookEventAlreadyProcessed)
}

func TestHandleJitsiWebhookConferenceLifecycleTouchesRoom(t *testing.T) {
	room := models.NewRoom("Meeting", uuid.New(), models.RoomTypeMeeting)
	room.JitsiRoomName = "room-123"
	lifecycle := &fakeRoomLifecycleRepo{
		roomsByJitsi: map[string]*models.Room{
			"room-123": room,
		},
		participants: map[string]*models.RoomParticipant{},
	}
	handler := NewWebhookHandler()
	handler.SetRoomLifecycleRepository(lifecycle)

	ctx := context.Background()
	createdPayload := []byte(`{
		"event": "conference.created",
		"room": "room-123",
		"timestamp": "2026-03-25T10:00:00Z"
	}`)
	endedPayload := []byte(`{
		"event": "conference.ended",
		"room": "room-123",
		"timestamp": "2026-03-25T11:00:00Z"
	}`)
	err := handler.HandleJitsiWebhook(ctx, createdPayload, "")
	assert.NoError(t, err)
	err = handler.HandleJitsiWebhook(ctx, endedPayload, "")
	assert.NoError(t, err)
	assert.Equal(t, 2, lifecycle.updateCalls)
	assert.Equal(t, "2026-03-25T11:00:00Z", room.UpdatedAt.UTC().Format(time.RFC3339))
}

func TestHandleJitsiWebhookParticipantLifecycleSyncsParticipants(t *testing.T) {
	room := models.NewRoom("Meeting", uuid.New(), models.RoomTypeMeeting)
	room.JitsiRoomName = "room-123"
	userID := uuid.New()
	lifecycle := &fakeRoomLifecycleRepo{
		roomsByJitsi: map[string]*models.Room{
			"room-123": room,
		},
		participants: map[string]*models.RoomParticipant{},
	}
	handler := NewWebhookHandler()
	handler.SetRoomLifecycleRepository(lifecycle)

	ctx := context.Background()
	joinPayload := []byte(`{
		"event":"participant.joined",
		"room":"room-123",
		"timestamp":"2026-03-25T12:00:00Z",
		"data":{"user_id":"` + userID.String() + `"}
	}`)
	leavePayload := []byte(`{
		"event":"participant.left",
		"room":"room-123",
		"timestamp":"2026-03-25T13:00:00Z",
		"data":{"user_id":"` + userID.String() + `"}
	}`)
	err := handler.HandleJitsiWebhook(ctx, joinPayload, "")
	assert.NoError(t, err)
	assert.Equal(t, 1, lifecycle.addCalls)

	err = handler.HandleJitsiWebhook(ctx, leavePayload, "")
	assert.NoError(t, err)
	assert.Equal(t, 1, lifecycle.removeCalls)
	assert.Equal(t, 2, lifecycle.updateCalls)
}

func TestHandleJitsiWebhookParticipantLifecycleIgnoresMissingRoom(t *testing.T) {
	handler := NewWebhookHandler()
	handler.SetRoomLifecycleRepository(&fakeRoomLifecycleRepo{
		roomsByJitsi: map[string]*models.Room{},
		participants: map[string]*models.RoomParticipant{},
	})
	err := handler.HandleJitsiWebhook(context.Background(), []byte(`{
		"event":"participant.joined",
		"room":"unknown-room",
		"data":{"user_id":"`+uuid.New().String()+`"}
	}`), "")
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

func TestDispatchSuccessWithDeliveryLog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "conference.created", r.Header.Get("X-Webhook-Event"))
		assert.NotEmpty(t, r.Header.Get("X-Webhook-Signature"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	webhook := &Webhook{
		ID:     uuid.New(),
		URL:    server.URL,
		Secret: "test-secret",
	}
	provider := &fakeActiveWebhookProvider{hooks: []*Webhook{webhook}}
	store := &fakeDeliveryStore{}
	dispatcher := NewWebhookDispatcherWithConfig(provider, store, nil, 2, time.Millisecond)
	dispatcher.sleep = func(time.Duration) {}

	err := dispatcher.Dispatch(context.Background(), "conference.created", map[string]string{"room": "room-1"})
	assert.NoError(t, err)
	assert.Len(t, store.deliveries, 1)
	assert.True(t, store.deliveries[0].Success)
	assert.Equal(t, 0, store.deliveries[0].RetryCount)
}

func TestDispatchRetriesAndDeadLetter(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("failure"))
	}))
	defer server.Close()

	webhook := &Webhook{
		ID:     uuid.New(),
		URL:    server.URL,
		Secret: "test-secret",
	}
	provider := &fakeActiveWebhookProvider{hooks: []*Webhook{webhook}}
	store := &fakeDeliveryStore{}
	dispatcher := NewWebhookDispatcherWithConfig(provider, store, nil, 1, time.Millisecond)
	dispatcher.sleep = func(time.Duration) {}

	err := dispatcher.Dispatch(context.Background(), "conference.ended", map[string]string{"room": "room-1"})
	assert.Error(t, err)
	assert.Equal(t, 2, attempts) // initial + one retry
	assert.Len(t, store.deliveries, 1)
	assert.False(t, store.deliveries[0].Success)
	assert.Equal(t, 1, store.deliveries[0].RetryCount)
	assert.Contains(t, store.deliveries[0].ResponseBody, "dead_letter")
}

func TestDispatchWithRepositoryError(t *testing.T) {
	provider := &fakeActiveWebhookProvider{err: assert.AnError}
	dispatcher := NewWebhookDispatcherWithConfig(provider, nil, nil, 0, time.Millisecond)
	dispatcher.sleep = func(time.Duration) {}

	err := dispatcher.Dispatch(context.Background(), "conference.created", map[string]string{"room": "room-1"})
	assert.Error(t, err)
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

type testIncomingStore struct {
	items map[string]bool
}

func (s *testIncomingStore) IsIncomingEventProcessed(ctx context.Context, source, idempotencyKey string) (bool, error) {
	return s.items[source+":"+idempotencyKey], nil
}

func (s *testIncomingStore) StoreIncomingEvent(ctx context.Context, event *IncomingEvent) error {
	key := event.Source + ":" + event.IdempotencyKey
	if s.items[key] {
		return ErrWebhookEventAlreadyProcessed
	}
	s.items[key] = true
	return nil
}

type fakeActiveWebhookProvider struct {
	hooks []*Webhook
	err   error
}

func (f *fakeActiveWebhookProvider) GetActiveByEventType(ctx context.Context, eventType string) ([]*Webhook, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.hooks, nil
}

type fakeDeliveryStore struct {
	mu         sync.Mutex
	deliveries []*WebhookDelivery
}

type fakeRoomLifecycleRepo struct {
	roomsByJitsi map[string]*models.Room
	participants map[string]*models.RoomParticipant
	updateCalls  int
	addCalls     int
	removeCalls  int
}

func (f *fakeRoomLifecycleRepo) GetByJitsiRoomName(ctx context.Context, jitsiRoomName string) (*models.Room, error) {
	room, ok := f.roomsByJitsi[jitsiRoomName]
	if !ok {
		return nil, errors.New("room not found")
	}
	return room, nil
}

func (f *fakeRoomLifecycleRepo) Update(ctx context.Context, room *models.Room) error {
	f.updateCalls++
	return nil
}

func (f *fakeRoomLifecycleRepo) GetParticipant(ctx context.Context, roomID, userID uuid.UUID) (*models.RoomParticipant, error) {
	return f.participants[f.participantKey(roomID, userID)], nil
}

func (f *fakeRoomLifecycleRepo) AddParticipant(ctx context.Context, roomID, userID uuid.UUID, role models.ParticipantRole) error {
	f.addCalls++
	f.participants[f.participantKey(roomID, userID)] = &models.RoomParticipant{
		RoomID: roomID,
		UserID: userID,
		Role:   role,
	}
	return nil
}

func (f *fakeRoomLifecycleRepo) RemoveParticipant(ctx context.Context, roomID, userID uuid.UUID) error {
	f.removeCalls++
	delete(f.participants, f.participantKey(roomID, userID))
	return nil
}

func (f *fakeRoomLifecycleRepo) participantKey(roomID, userID uuid.UUID) string {
	return roomID.String() + ":" + userID.String()
}

func (f *fakeDeliveryStore) CreateDelivery(ctx context.Context, delivery *WebhookDelivery) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deliveries = append(f.deliveries, delivery)
	return nil
}
