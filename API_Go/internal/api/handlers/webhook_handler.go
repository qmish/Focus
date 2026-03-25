package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/qmish/focus-api/internal/webhooks"
)

// InboundWebhookHandler handles public inbound webhooks.
type InboundWebhookHandler struct {
	webhookHandler *webhooks.WebhookHandler
}

// NewInboundWebhookHandler creates a handler for inbound webhooks.
func NewInboundWebhookHandler(webhookHandler *webhooks.WebhookHandler) *InboundWebhookHandler {
	return &InboundWebhookHandler{
		webhookHandler: webhookHandler,
	}
}

// JitsiWebhook POST /api/v1/webhooks/jitsi
func (h *InboundWebhookHandler) JitsiWebhook(w http.ResponseWriter, r *http.Request) {
	if h.webhookHandler == nil {
		http.Error(w, "webhook handler unavailable", http.StatusServiceUnavailable)
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read payload", http.StatusBadRequest)
		return
	}

	signature := strings.TrimSpace(r.Header.Get("X-Jitsi-Signature"))
	idempotencyKey := strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))

	err = h.webhookHandler.HandleJitsiWebhookWithIdempotency(r.Context(), payload, signature, idempotencyKey)
	if err != nil {
		switch {
		case errors.Is(err, webhooks.ErrMissingWebhookSignature), errors.Is(err, webhooks.ErrInvalidWebhookSignature):
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		case errors.Is(err, webhooks.ErrWebhookEventAlreadyProcessed):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"duplicate"}`))
			return
		case strings.Contains(err.Error(), "failed to parse jitsi webhook"):
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		default:
			http.Error(w, "failed to process webhook", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"accepted"}`))
}
