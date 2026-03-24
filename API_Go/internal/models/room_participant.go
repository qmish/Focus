package models

import (
	"time"

	"github.com/google/uuid"
)

// ParticipantRole роль участника в комнате
type ParticipantRole string

const (
	ParticipantRoleMember    ParticipantRole = "member"
	ParticipantRoleModerator ParticipantRole = "moderator"
	ParticipantRoleAdmin     ParticipantRole = "admin"
)

// RoomParticipant связывает пользователя с комнатой
type RoomParticipant struct {
	RoomID     uuid.UUID       `gorm:"type:uuid;primaryKey" json:"room_id"`
	UserID     uuid.UUID       `gorm:"type:uuid;primaryKey" json:"user_id"`
	Role       ParticipantRole `gorm:"type:varchar(20);not null" json:"role"`
	JoinedAt   time.Time       `json:"joined_at"`
	LastReadAt *time.Time      `json:"last_read_at"`
	LeftAt     *time.Time      `json:"-"`

	Room *Room `gorm:"foreignKey:RoomID" json:"-"`
	User *User `gorm:"foreignKey:UserID" json:"-"`
}

// TableName возвращает имя таблицы
func (RoomParticipant) TableName() string {
	return "room_participants"
}

// NewRoomParticipant создаёт новую запись участника
func NewRoomParticipant(roomID, userID uuid.UUID, role ParticipantRole) *RoomParticipant {
	now := time.Now()
	return &RoomParticipant{
		RoomID:     roomID,
		UserID:     userID,
		Role:       role,
		JoinedAt:   now,
		LastReadAt: &now,
	}
}
