package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MessageType тип сообщения
type MessageType string

const (
	MessageTypeText   MessageType = "text"
	MessageTypeImage  MessageType = "image"
	MessageTypeFile   MessageType = "file"
	MessageTypeSystem MessageType = "system"
)

// Message представляет сообщение в чате
type Message struct {
	ID        uuid.UUID   `gorm:"type:uuid;primary_key" json:"id"`
	RoomID    uuid.UUID   `gorm:"type:uuid;not null;index:idx_room_created" json:"room_id"`
	UserID    uuid.UUID   `gorm:"type:uuid;not null" json:"user_id"`
	Content   string      `gorm:"type:text;not null" json:"content"`
	Type      MessageType `gorm:"type:varchar(20);not null" json:"type"`
	ReplyToID *uuid.UUID  `gorm:"type:uuid" json:"reply_to_id,omitempty"`
	Metadata  Metadata    `gorm:"type:jsonb" json:"metadata"`
	IsDeleted bool        `gorm:"not null;default:false" json:"is_deleted"`
	CreatedAt time.Time   `gorm:"index:idx_room_created" json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`

	Room      *Room             `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	User      *User             `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ReplyTo   *Message          `gorm:"foreignKey:ReplyToID" json:"reply_to,omitempty"`
	Reactions []MessageReaction `gorm:"foreignKey:MessageID" json:"reactions,omitempty"`
}

// Metadata метаданные сообщения
type Metadata struct {
	Edited    *bool      `json:"edited,omitempty"`
	EditedAt  *time.Time `json:"edited_at,omitempty"`
	EditedBy  *uuid.UUID `json:"edited_by,omitempty"`
	ReplyTo   *uuid.UUID `json:"reply_to,omitempty"`
	Reactions []Reaction `json:"reactions,omitempty"`
	FileName  string     `json:"file_name,omitempty"`
	FileSize  int64      `json:"file_size,omitempty"`
	FileMIME  string     `json:"file_mime,omitempty"`
	FileID    string     `json:"file_id,omitempty"`
}

func (m Metadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *Metadata) Scan(value interface{}) error {
	if value == nil {
		*m = Metadata{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("Metadata.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(bytes, m)
}

// Reaction реакция на сообщение
type Reaction struct {
	Emoji   string   `json:"emoji"`
	Count   int      `json:"count"`
	UserIDs []string `json:"user_ids,omitempty"`
}

// TableName возвращает имя таблицы
func (Message) TableName() string {
	return "messages"
}

// NewMessage создаёт новое сообщение
func NewMessage(roomID, userID uuid.UUID, content string, msgType MessageType) *Message {
	now := time.Now()
	return &Message{
		ID:        uuid.New(),
		RoomID:    roomID,
		UserID:    userID,
		Content:   content,
		Type:      msgType,
		Metadata:  Metadata{},
		IsDeleted: false,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// MessageReaction реакция пользователя на сообщение
type MessageReaction struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	MessageID uuid.UUID `gorm:"type:uuid;not null;index" json:"message_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Emoji     string    `gorm:"type:varchar(50);not null" json:"emoji"`
	CreatedAt time.Time `json:"created_at"`

	Message *Message `gorm:"foreignKey:MessageID" json:"-"`
	User    *User    `gorm:"foreignKey:UserID" json:"-"`
}

// TableName возвращает имя таблицы
func (MessageReaction) TableName() string {
	return "message_reactions"
}

// NewMessageReaction создаёт новую реакцию
func NewMessageReaction(messageID, userID uuid.UUID, emoji string) *MessageReaction {
	return &MessageReaction{
		ID:        uuid.New(),
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
	}
}
