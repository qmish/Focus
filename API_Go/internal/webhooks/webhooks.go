package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// WebhookType тип вебхука
type WebhookType string

const (
	WebhookTypeJitsi    WebhookType = "jitsi"
	WebhookTypeExchange WebhookType = "exchange"
	WebhookTypeCustom   WebhookType = "custom"
)

// Webhook модель вебхука
type Webhook struct {
	ID              uuid.UUID `json:"id"`
	OwnerID         uuid.UUID `json:"owner_id"`
	URL             string    `json:"url"`
	Secret          string    `json:"secret"`
	EventTypes      []string  `json:"event_types"`
	IsActive        bool      `json:"is_active"`
	SignatureMethod string    `json:"signature_method"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// WebhookDelivery лог доставки вебхука
type WebhookDelivery struct {
	ID           uuid.UUID  `json:"id"`
	WebhookID    uuid.UUID  `json:"webhook_id"`
	Payload      []byte     `json:"payload"`
	ResponseCode int        `json:"response_code"`
	ResponseBody string     `json:"response_body"`
	Success      bool       `json:"success"`
	RetryCount   int        `json:"retry_count"`
	DeliveredAt  *time.Time `json:"delivered_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

var (
	ErrMissingWebhookSignature      = errors.New("missing webhook signature")
	ErrInvalidWebhookSignature      = errors.New("invalid webhook signature")
	ErrWebhookEventAlreadyProcessed = errors.New("webhook event already processed")
)

// IncomingEvent stores inbound webhooks for tracing/idempotency.
type IncomingEvent struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	Source         string     `gorm:"type:varchar(32);not null;index" json:"source"`
	EventType      string     `gorm:"type:varchar(128);not null" json:"event_type"`
	IdempotencyKey string     `gorm:"type:varchar(128);not null;uniqueIndex:idx_incoming_source_idem" json:"idempotency_key"`
	Signature      string     `gorm:"type:text" json:"signature"`
	Payload        string     `gorm:"type:text;not null" json:"payload"`
	ReceivedAt     time.Time  `gorm:"not null" json:"received_at"`
	ProcessedAt    *time.Time `json:"processed_at,omitempty"`
}

// TableName returns the table name for incoming events.
func (IncomingEvent) TableName() string {
	return "incoming_webhook_events"
}

// IncomingEventStore persists inbound webhook events.
type IncomingEventStore interface {
	IsIncomingEventProcessed(ctx context.Context, source, idempotencyKey string) (bool, error)
	StoreIncomingEvent(ctx context.Context, event *IncomingEvent) error
}

// WebhookHandler обработчик вебхуков
type WebhookHandler struct {
	secret string
	store  IncomingEventStore
}

// NewWebhookHandler создаёт новый WebhookHandler
func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{}
}

// NewWebhookHandlerWithConfig creates webhook handler with signature/idempotency config.
func NewWebhookHandlerWithConfig(secret string, store IncomingEventStore) *WebhookHandler {
	return &WebhookHandler{
		secret: strings.TrimSpace(secret),
		store:  store,
	}
}

// JitsiWebhookEvent событие от Jitsi
type JitsiWebhookEvent struct {
	Event          string                 `json:"event"`
	ConferenceName string                 `json:"conference_name"`
	Room           string                 `json:"room"`
	Timestamp      string                 `json:"timestamp"`
	Data           map[string]interface{} `json:"data"`
}

// HandleJitsiWebhook обрабатывает входящий вебхук от Jitsi
func (h *WebhookHandler) HandleJitsiWebhook(ctx context.Context, payload []byte, signature string) error {
	return h.HandleJitsiWebhookWithIdempotency(ctx, payload, signature, "")
}

// HandleJitsiWebhookWithIdempotency validates signature, deduplicates, and processes webhook.
func (h *WebhookHandler) HandleJitsiWebhookWithIdempotency(ctx context.Context, payload []byte, signature, idempotencyKey string) error {
	if h.secret != "" {
		if strings.TrimSpace(signature) == "" {
			return ErrMissingWebhookSignature
		}
		if err := verifyWebhookSignature(h.secret, payload, signature); err != nil {
			return ErrInvalidWebhookSignature
		}
	}

	// Парсим событие
	var event JitsiWebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to parse jitsi webhook: %w", err)
	}
	if h.store != nil {
		key := normalizeIdempotencyKey(idempotencyKey, payload)
		processed, err := h.store.IsIncomingEventProcessed(ctx, string(WebhookTypeJitsi), key)
		if err != nil {
			return err
		}
		if processed {
			return ErrWebhookEventAlreadyProcessed
		}
		processedAt := time.Now().UTC()
		err = h.store.StoreIncomingEvent(ctx, &IncomingEvent{
			ID:             uuid.New(),
			Source:         string(WebhookTypeJitsi),
			EventType:      event.Event,
			IdempotencyKey: key,
			Signature:      signature,
			Payload:        string(payload),
			ReceivedAt:     processedAt,
			ProcessedAt:    &processedAt,
		})
		if err != nil {
			return err
		}
	}

	// Обрабатываем по типу события
	switch event.Event {
	case "conference.created":
		return h.handleConferenceCreated(ctx, &event)
	case "conference.ended":
		return h.handleConferenceEnded(ctx, &event)
	case "participant.joined":
		return h.handleParticipantJoined(ctx, &event)
	case "participant.left":
		return h.handleParticipantLeft(ctx, &event)
	}

	return nil
}

func (h *WebhookHandler) handleConferenceCreated(ctx context.Context, event *JitsiWebhookEvent) error {
	// TODO: Сохранить событие в БД, отправить уведомления
	return nil
}

func (h *WebhookHandler) handleConferenceEnded(ctx context.Context, event *JitsiWebhookEvent) error {
	// TODO: Обновить статус комнаты
	return nil
}

func (h *WebhookHandler) handleParticipantJoined(ctx context.Context, event *JitsiWebhookEvent) error {
	// TODO: Отправить уведомление в чат
	return nil
}

func (h *WebhookHandler) handleParticipantLeft(ctx context.Context, event *JitsiWebhookEvent) error {
	// TODO: Отправить уведомление в чат
	return nil
}

// OutgoingWebhook исходящий вебхук
type OutgoingWebhook struct {
	WebhookID uuid.UUID
	URL       string
	Secret    string
	Payload   []byte
	EventType string
}

// ActiveWebhookProvider provides active webhooks by event type.
type ActiveWebhookProvider interface {
	GetActiveByEventType(ctx context.Context, eventType string) ([]*Webhook, error)
}

// DeliveryStore stores webhook delivery results.
type DeliveryStore interface {
	CreateDelivery(ctx context.Context, delivery *WebhookDelivery) error
}

// HTTPDoer abstracts http.Client for testing.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// WebhookDispatcher диспетчер исходящих вебхуков
type WebhookDispatcher struct {
	provider   ActiveWebhookProvider
	store      DeliveryStore
	httpClient HTTPDoer
	maxRetries int
	baseDelay  time.Duration
	sleep      func(time.Duration)
}

// NewWebhookDispatcher создаёт новый WebhookDispatcher
func NewWebhookDispatcher() *WebhookDispatcher {
	return &WebhookDispatcher{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		maxRetries: 2,
		baseDelay:  250 * time.Millisecond,
		sleep:      time.Sleep,
	}
}

// NewWebhookDispatcherWithConfig creates dispatcher with dependencies and retry policy.
func NewWebhookDispatcherWithConfig(
	provider ActiveWebhookProvider,
	store DeliveryStore,
	httpClient HTTPDoer,
	maxRetries int,
	baseDelay time.Duration,
) *WebhookDispatcher {
	dispatcher := NewWebhookDispatcher()
	dispatcher.provider = provider
	dispatcher.store = store
	if httpClient != nil {
		dispatcher.httpClient = httpClient
	}
	if maxRetries >= 0 {
		dispatcher.maxRetries = maxRetries
	}
	if baseDelay > 0 {
		dispatcher.baseDelay = baseDelay
	}
	return dispatcher
}

// Dispatch рассылает событие всем подписанным вебхукам
func (d *WebhookDispatcher) Dispatch(ctx context.Context, eventType string, payload interface{}) error {
	if d.provider == nil {
		return nil
	}

	payloadBytes, err := marshalWebhookPayload(payload)
	if err != nil {
		return err
	}

	webhooks, err := d.provider.GetActiveByEventType(ctx, eventType)
	if err != nil {
		return err
	}
	if len(webhooks) == 0 {
		return nil
	}

	var failures []string
	for _, webhook := range webhooks {
		if webhook == nil {
			continue
		}
		if err := d.sendWebhook(ctx, webhook, payloadBytes, eventType); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("outbound webhook dispatch failures: %s", strings.Join(failures, "; "))
	}
	return nil
}

func (d *WebhookDispatcher) sendWebhook(ctx context.Context, webhook *Webhook, payload []byte, eventType string) error {
	attempts := d.maxRetries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastCode int
	var lastBody string
	var lastErr error

	for attempt := 0; attempt < attempts; attempt++ {
		code, body, err := d.doRequest(ctx, webhook, payload, eventType)
		lastCode = code
		lastBody = body
		lastErr = err

		if err == nil && code >= 200 && code < 300 {
			d.recordDelivery(ctx, webhook.ID, payload, code, body, true, attempt)
			return nil
		}

		if attempt < attempts-1 && d.sleep != nil {
			d.sleep(d.backoff(attempt))
		}
	}

	deadLetterBody := lastBody
	if lastErr != nil {
		deadLetterBody = fmt.Sprintf("dead_letter: %v", lastErr)
	} else {
		deadLetterBody = fmt.Sprintf("dead_letter: status=%d body=%s", lastCode, lastBody)
	}
	d.recordDelivery(ctx, webhook.ID, payload, lastCode, deadLetterBody, false, d.maxRetries)

	if lastErr != nil {
		return fmt.Errorf("webhook %s delivery failed: %w", webhook.ID.String(), lastErr)
	}
	return fmt.Errorf("webhook %s delivery failed with status %d", webhook.ID.String(), lastCode)
}

func (d *WebhookDispatcher) createSignature(secret string, payload []byte, timestamp time.Time) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(timestamp.Format(time.RFC3339)))
	h.Write([]byte("."))
	h.Write(payload)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (d *WebhookDispatcher) backoff(attempt int) time.Duration {
	delay := d.baseDelay
	for i := 0; i < attempt; i++ {
		delay *= 2
	}
	return delay
}

func (d *WebhookDispatcher) doRequest(ctx context.Context, webhook *Webhook, payload []byte, eventType string) (int, string, error) {
	if d.httpClient == nil {
		return 0, "", fmt.Errorf("http client is not configured")
	}
	timestamp := time.Now().UTC()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(payload))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", eventType)
	req.Header.Set("X-Webhook-Timestamp", timestamp.Format(time.RFC3339))
	req.Header.Set("X-Webhook-Signature", d.createSignature(webhook.Secret, payload, timestamp))

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return resp.StatusCode, string(bodyBytes), nil
}

func (d *WebhookDispatcher) recordDelivery(
	ctx context.Context,
	webhookID uuid.UUID,
	payload []byte,
	responseCode int,
	responseBody string,
	success bool,
	retryCount int,
) {
	if d.store == nil {
		return
	}
	now := time.Now().UTC()
	_ = d.store.CreateDelivery(ctx, &WebhookDelivery{
		ID:           uuid.New(),
		WebhookID:    webhookID,
		Payload:      payload,
		ResponseCode: responseCode,
		ResponseBody: responseBody,
		Success:      success,
		RetryCount:   retryCount,
		DeliveredAt:  &now,
		CreatedAt:    now,
	})
}

func marshalWebhookPayload(payload interface{}) ([]byte, error) {
	switch v := payload.(type) {
	case []byte:
		return v, nil
	case json.RawMessage:
		return []byte(v), nil
	default:
		return json.Marshal(payload)
	}
}

// VerifySignature проверяет подпись вебхука
func VerifySignature(secret, payload, signature string) error {
	expectedSig := createSignatureForVerify(secret, []byte(payload))
	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

func createSignatureForVerify(secret string, payload []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func verifyWebhookSignature(secret string, payload []byte, rawSignature string) error {
	signature := normalizeSignature(rawSignature)
	// Backward-compatible base64 signature.
	expectedBase64 := createSignatureForVerify(secret, payload)
	if hmac.Equal([]byte(signature), []byte(expectedBase64)) {
		return nil
	}

	// Jitsi-compatible hex signature.
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	expectedHex := hex.EncodeToString(h.Sum(nil))
	if hmac.Equal([]byte(strings.ToLower(signature)), []byte(expectedHex)) {
		return nil
	}
	return ErrInvalidWebhookSignature
}

func normalizeSignature(rawSignature string) string {
	sig := strings.TrimSpace(rawSignature)
	if strings.HasPrefix(strings.ToLower(sig), "sha256=") {
		return strings.TrimSpace(sig[7:])
	}
	return sig
}

func normalizeIdempotencyKey(idempotencyKey string, payload []byte) string {
	if key := strings.TrimSpace(idempotencyKey); key != "" {
		return key
	}
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}
