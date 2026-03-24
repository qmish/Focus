package websocket

import (
	"encoding/json"
	"testing"

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
