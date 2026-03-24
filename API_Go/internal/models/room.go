package models

import (
	"time"

	"github.com/google/uuid"
)

// RoomType тип комнаты
type RoomType string

const (
	RoomTypePublic  RoomType = "public"
	RoomTypePrivate RoomType = "private"
	RoomTypeMeeting RoomType = "meeting"
)

// RoomSettings настройки комнаты
type RoomSettings struct {
	AllowGuests             bool `json:"allow_guests"`
	RequireModeratorForMsgs bool `json:"require_moderator_for_messages"`
	MaxParticipants         int  `json:"max_participants"`
}

// Room представляет комнату чата/встречи
type Room struct {
	ID            uuid.UUID    `gorm:"type:uuid;primary_key" json:"id"`
	Name          string       `gorm:"type:varchar(100);not null" json:"name"`
	Description   string       `gorm:"type:text" json:"description"`
	CreatorID     uuid.UUID    `gorm:"type:uuid;not null" json:"creator_id"`
	Type          RoomType     `gorm:"type:varchar(20);not null" json:"type"`
	JitsiRoomName string       `gorm:"type:varchar(100);uniqueIndex;not null" json:"jitsi_room_name"`
	IsPrivate     bool         `gorm:"not null;default:false" json:"is_private"`
	Settings      RoomSettings `gorm:"type:jsonb" json:"settings"`
	DeletedAt     *time.Time   `gorm:"index" json:"-"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`

	Creator      *User             `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
	Participants []RoomParticipant `gorm:"foreignKey:RoomID" json:"participants_count,omitempty"`
	Messages     []Message         `gorm:"foreignKey:RoomID" json:"-"`
}

// TableName возвращает имя таблицы
func (Room) TableName() string {
	return "rooms"
}

// NewRoom создаёт новую комнату
func NewRoom(name string, creatorID uuid.UUID, roomType RoomType) *Room {
	return &Room{
		ID:            uuid.New(),
		Name:          name,
		CreatorID:     creatorID,
		Type:          roomType,
		JitsiRoomName: uuid.New().String(),
		IsPrivate:     roomType == RoomTypePrivate,
		Settings: RoomSettings{
			AllowGuests:             false,
			RequireModeratorForMsgs: false,
			MaxParticipants:         100,
		},
	}
}

// GetJitsiURL возвращает URL для входа в Jitsi
func (r *Room) GetJitsiURL(baseURL string) string {
	return baseURL + "/" + r.JitsiRoomName
}
