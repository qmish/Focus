package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessage(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	content := "Hello, World!"

	msg := NewMessage(roomID, userID, content, MessageTypeText)

	assert.Equal(t, roomID, msg.RoomID)
	assert.Equal(t, userID, msg.UserID)
	assert.Equal(t, content, msg.Content)
	assert.Equal(t, MessageTypeText, msg.Type)
	assert.False(t, msg.IsDeleted)
	assert.NotEmpty(t, msg.ID)
	assert.NotEmpty(t, msg.CreatedAt)
}

func TestNewMessageImage(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()

	msg := NewMessage(roomID, userID, "image.jpg", MessageTypeImage)

	assert.Equal(t, MessageTypeImage, msg.Type)
}

func TestNewMessageFile(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()

	msg := NewMessage(roomID, userID, "document.pdf", MessageTypeFile)

	assert.Equal(t, MessageTypeFile, msg.Type)
}

func TestNewMessageSystem(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()

	msg := NewMessage(roomID, userID, "User joined", MessageTypeSystem)

	assert.Equal(t, MessageTypeSystem, msg.Type)
}

func TestMessageWithReplyTo(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	replyToID := uuid.New()

	msg := NewMessage(roomID, userID, "Reply", MessageTypeText)
	msg.ReplyToID = &replyToID

	assert.NotNil(t, msg.ReplyToID)
	assert.Equal(t, replyToID, *msg.ReplyToID)
}

func TestMessageMetadata(t *testing.T) {
	msg := &Message{
		Metadata: Metadata{
			Edited: boolPtr(true),
		},
	}

	assert.NotNil(t, msg.Metadata.Edited)
	assert.True(t, *msg.Metadata.Edited)
}

func TestMessageTableName(t *testing.T) {
	msg := Message{}
	assert.Equal(t, "messages", msg.TableName())
}

func TestMessageType(t *testing.T) {
	assert.Equal(t, MessageType("text"), MessageTypeText)
	assert.Equal(t, MessageType("image"), MessageTypeImage)
	assert.Equal(t, MessageType("file"), MessageTypeFile)
	assert.Equal(t, MessageType("system"), MessageTypeSystem)
}

func TestNewMessageReaction(t *testing.T) {
	messageID := uuid.New()
	userID := uuid.New()
	emoji := "👍"

	reaction := NewMessageReaction(messageID, userID, emoji)

	assert.Equal(t, messageID, reaction.MessageID)
	assert.Equal(t, userID, reaction.UserID)
	assert.Equal(t, emoji, reaction.Emoji)
	assert.NotEmpty(t, reaction.ID)
}

func TestMessageReactionTableName(t *testing.T) {
	r := MessageReaction{}
	assert.Equal(t, "message_reactions", r.TableName())
}

func TestMessageWithThreadRootID(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	threadRootID := uuid.New()

	msg := NewMessage(roomID, userID, "Thread reply", MessageTypeText)
	msg.ThreadRootID = &threadRootID

	assert.NotNil(t, msg.ThreadRootID)
	assert.Equal(t, threadRootID, *msg.ThreadRootID)
	assert.Nil(t, msg.ThreadRoot)
}

func TestMessageWithoutThreadRootID(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()

	msg := NewMessage(roomID, userID, "Regular message", MessageTypeText)

	assert.Nil(t, msg.ThreadRootID)
	assert.Nil(t, msg.ThreadRoot)
}

func TestMessageThreadAndReplyToCombined(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	threadRootID := uuid.New()
	replyToID := uuid.New()

	msg := NewMessage(roomID, userID, "Combined", MessageTypeText)
	msg.ThreadRootID = &threadRootID
	msg.ReplyToID = &replyToID

	assert.NotNil(t, msg.ThreadRootID)
	assert.NotNil(t, msg.ReplyToID)
	assert.Equal(t, threadRootID, *msg.ThreadRootID)
	assert.Equal(t, replyToID, *msg.ReplyToID)
}

func TestMetadataMentionsSerialization(t *testing.T) {
	mentions := []string{uuid.New().String(), uuid.New().String()}
	meta := Metadata{Mentions: mentions}

	val, err := meta.Value()
	assert.NoError(t, err)
	assert.NotNil(t, val)

	var decoded Metadata
	err = decoded.Scan(val)
	assert.NoError(t, err)
	assert.Equal(t, mentions, decoded.Mentions)
}

func TestMetadataEmptyMentions(t *testing.T) {
	meta := Metadata{}
	val, err := meta.Value()
	assert.NoError(t, err)

	var decoded Metadata
	err = decoded.Scan(val)
	assert.NoError(t, err)
	assert.Nil(t, decoded.Mentions)
}

// --- Этап 4: сериализация полей редактирования ---

func TestMetadataEditedAtSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	edited := true
	meta := Metadata{
		Edited:   &edited,
		EditedAt: &now,
	}

	val, err := meta.Value()
	require.NoError(t, err)
	require.NotNil(t, val)

	var decoded Metadata
	require.NoError(t, decoded.Scan(val))
	require.NotNil(t, decoded.Edited)
	assert.True(t, *decoded.Edited)
	require.NotNil(t, decoded.EditedAt)
	assert.True(t, decoded.EditedAt.Equal(now), "EditedAt must round-trip via JSON serialization")
}

func TestMetadataEditedBySerialization(t *testing.T) {
	editorID := uuid.New()
	edited := true
	meta := Metadata{
		Edited:   &edited,
		EditedBy: &editorID,
	}

	val, err := meta.Value()
	require.NoError(t, err)

	var decoded Metadata
	require.NoError(t, decoded.Scan(val))
	require.NotNil(t, decoded.EditedBy)
	assert.Equal(t, editorID, *decoded.EditedBy)
}

func TestMetadataEditedFieldsAbsentWhenNil(t *testing.T) {
	meta := Metadata{}
	val, err := meta.Value()
	require.NoError(t, err)

	var decoded Metadata
	require.NoError(t, decoded.Scan(val))
	assert.Nil(t, decoded.Edited)
	assert.Nil(t, decoded.EditedAt)
	assert.Nil(t, decoded.EditedBy)
}

func boolPtr(b bool) *bool {
	return &b
}
