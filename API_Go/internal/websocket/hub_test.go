package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewHub(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(logger)

	assert.NotNil(t, hub)
	assert.NotNil(t, hub.Clients)
	assert.NotNil(t, hub.Rooms)
	assert.NotNil(t, hub.Register)
	assert.NotNil(t, hub.Unregister)
	assert.NotNil(t, hub.Broadcast)
}

func TestWSMessageSerialization(t *testing.T) {
	msg := WSMessage{
		Type:      MessageTypeSubscribe,
		RequestID: "test-123",
		Payload:   json.RawMessage(`{"room_id":"room-123"}`),
	}

	data, err := json.Marshal(msg)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded WSMessage
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, msg.Type, decoded.Type)
	assert.Equal(t, msg.RequestID, decoded.RequestID)
}

func TestSubscribePayload(t *testing.T) {
	payload := SubscribePayload{
		RoomID: "room-123",
	}

	data, err := json.Marshal(payload)
	assert.NoError(t, err)

	var decoded SubscribePayload
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, payload.RoomID, decoded.RoomID)
}

func TestMessagePayload(t *testing.T) {
	payload := MessagePayload{
		RoomID:  "room-123",
		UserID:  "user-456",
		Content: "Hello, World!",
		Type:    "text",
	}

	data, err := json.Marshal(payload)
	assert.NoError(t, err)

	var decoded MessagePayload
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, payload.RoomID, decoded.RoomID)
	assert.Equal(t, payload.UserID, decoded.UserID)
	assert.Equal(t, payload.Content, decoded.Content)
	assert.Equal(t, payload.Type, decoded.Type)
}

func TestTypingPayload(t *testing.T) {
	payload := TypingPayload{
		RoomID:   "room-123",
		UserID:   "user-456",
		UserName: "John Doe",
		IsTyping: true,
	}

	data, err := json.Marshal(payload)
	assert.NoError(t, err)

	var decoded TypingPayload
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, payload.RoomID, decoded.RoomID)
	assert.Equal(t, payload.UserID, decoded.UserID)
	assert.Equal(t, payload.UserName, decoded.UserName)
	assert.Equal(t, payload.IsTyping, decoded.IsTyping)
}

func TestUserEventPayload(t *testing.T) {
	payload := UserEventPayload{
		RoomID:   "room-123",
		UserID:   "user-456",
		UserName: "John Doe",
	}

	data, err := json.Marshal(payload)
	assert.NoError(t, err)

	var decoded UserEventPayload
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, payload.RoomID, decoded.RoomID)
	assert.Equal(t, payload.UserID, decoded.UserID)
	assert.Equal(t, payload.UserName, decoded.UserName)
}

func TestErrorPayload(t *testing.T) {
	payload := ErrorPayload{
		Code:    "invalid_message",
		Message: "Failed to parse message",
	}

	data, err := json.Marshal(payload)
	assert.NoError(t, err)

	var decoded ErrorPayload
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, payload.Code, decoded.Code)
	assert.Equal(t, payload.Message, decoded.Message)
}

func TestMessageTypeConstants(t *testing.T) {
	assert.Equal(t, MessageType("subscribe"), MessageTypeSubscribe)
	assert.Equal(t, MessageType("unsubscribe"), MessageTypeUnsubscribe)
	assert.Equal(t, MessageType("message"), MessageTypeMessage)
	assert.Equal(t, MessageType("typing"), MessageTypeTyping)
	assert.Equal(t, MessageType("user_joined"), MessageTypeUserJoined)
	assert.Equal(t, MessageType("user_left"), MessageTypeUserLeft)
	assert.Equal(t, MessageType("error"), MessageTypeError)
}

func TestSubscribeAuthorization(t *testing.T) {
	logger := zap.NewNop()

	t.Run("denies room subscription without access", func(t *testing.T) {
		hub := NewHub(logger)
		hub.SetRoomAccessChecker(func(_ context.Context, userID, roomID string) (bool, error) {
			return false, nil
		})
		client := &Client{
			ID:     "client-1",
			UserID: "user-1",
			Hub:    hub,
			Rooms:  map[string]bool{},
			Send:   make(chan []byte, 1),
		}

		client.handleSubscribe(json.RawMessage(`{"room_id":"room-1"}`))

		assert.Empty(t, client.Rooms)
		msg := mustReadWSMessage(t, client.Send)
		assert.Equal(t, MessageTypeError, msg.Type)

		var payload ErrorPayload
		err := json.Unmarshal(msg.Payload, &payload)
		assert.NoError(t, err)
		assert.Equal(t, "forbidden_room", payload.Code)
	})

	t.Run("allows room subscription for participant", func(t *testing.T) {
		hub := NewHub(logger)
		hub.SetRoomAccessChecker(func(_ context.Context, userID, roomID string) (bool, error) {
			return userID == "user-1" && roomID == "room-1", nil
		})
		client := &Client{
			ID:     "client-1",
			UserID: "user-1",
			Hub:    hub,
			Rooms:  map[string]bool{},
			Send:   make(chan []byte, 1),
		}

		client.handleSubscribe(json.RawMessage(`{"room_id":"room-1"}`))

		assert.True(t, client.Rooms["room-1"])
		assert.NotNil(t, hub.Rooms["room-1"]["client-1"])

		msg := mustReadWSMessage(t, client.Send)
		assert.Equal(t, MessageTypeSubscribe, msg.Type)
	})
}

func TestStrictRoomAccessWithoutChecker(t *testing.T) {
	hub := NewHub(zap.NewNop())
	hub.SetStrictRoomAccess(true)
	client := &Client{
		ID:     "client-1",
		UserID: "user-1",
		Hub:    hub,
		Rooms:  map[string]bool{},
		Send:   make(chan []byte, 1),
	}
	client.handleSubscribe(json.RawMessage(`{"room_id":"room-1"}`))

	assert.Empty(t, client.Rooms)
	msg := mustReadWSMessage(t, client.Send)
	assert.Equal(t, MessageTypeError, msg.Type)

	var payload ErrorPayload
	err := json.Unmarshal(msg.Payload, &payload)
	assert.NoError(t, err)
	assert.Equal(t, "forbidden_room", payload.Code)
}

func TestOriginAllowList(t *testing.T) {
	hub := NewHub(zap.NewNop())
	hub.SetAllowedOrigins([]string{"https://chat.company.com", "https://admin.company.com"})

	allowedRequest := &http.Request{Header: http.Header{}}
	allowedRequest.Header.Set("Origin", "https://chat.company.com")
	assert.True(t, hub.isOriginAllowed(allowedRequest))

	deniedRequest := &http.Request{Header: http.Header{}}
	deniedRequest.Header.Set("Origin", "https://evil.example.com")
	assert.False(t, hub.isOriginAllowed(deniedRequest))
}

func TestMessageAndTypingRequireSubscription(t *testing.T) {
	hub := NewHub(zap.NewNop())
	client := &Client{
		ID:     "client-1",
		UserID: "user-1",
		Hub:    hub,
		Rooms:  map[string]bool{},
		Send:   make(chan []byte, 2),
	}

	client.handleMessagePayload(json.RawMessage(`{"room_id":"room-1","user_id":"spoofed","content":"hello","type":"text"}`))
	msgErr := mustReadWSMessage(t, client.Send)
	assert.Equal(t, MessageTypeError, msgErr.Type)

	client.handleTyping(json.RawMessage(`{"room_id":"room-1","user_id":"spoofed","user_name":"x","is_typing":true}`))
	typingErr := mustReadWSMessage(t, client.Send)
	assert.Equal(t, MessageTypeError, typingErr.Type)
}

func TestMessagePayloadOverridesUserID(t *testing.T) {
	hub := NewHub(zap.NewNop())
	client := &Client{
		ID:     "client-1",
		UserID: "user-1",
		Hub:    hub,
		Rooms:  map[string]bool{"room-1": true},
		Send:   make(chan []byte, 1),
	}

	client.handleMessagePayload(json.RawMessage(`{"room_id":"room-1","user_id":"spoofed","content":"hello","type":"text"}`))

	select {
	case out := <-hub.Broadcast:
		var ws WSMessage
		err := json.Unmarshal(out.Message, &ws)
		assert.NoError(t, err)
		assert.Equal(t, MessageTypeMessage, ws.Type)

		var payload MessagePayload
		err = json.Unmarshal(ws.Payload, &payload)
		assert.NoError(t, err)
		assert.Equal(t, "user-1", payload.UserID)
		assert.Equal(t, "room-1", payload.RoomID)
	default:
		t.Fatal("expected broadcast message")
	}
}

func mustReadWSMessage(t *testing.T, ch <-chan []byte) WSMessage {
	t.Helper()
	select {
	case raw := <-ch:
		var msg WSMessage
		err := json.Unmarshal(raw, &msg)
		if err != nil {
			t.Fatalf("failed to decode ws message: %v", err)
		}
		return msg
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected websocket message")
	}
	return WSMessage{}
}

func TestClientTokenExpired(t *testing.T) {
	client := &Client{ExpiresAt: time.Now().Add(-1 * time.Minute)}
	assert.True(t, client.tokenExpired())

	client = &Client{ExpiresAt: time.Now().Add(10 * time.Minute)}
	assert.False(t, client.tokenExpired())

	client = &Client{}
	assert.False(t, client.tokenExpired())
}
