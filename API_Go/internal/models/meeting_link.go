package models

import (
	"time"

	"github.com/google/uuid"
)

// MeetingLink stores mapping between Focus rooms and Exchange calendar events.
type MeetingLink struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	RoomID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"room_id"`
	ExchangeEventID string     `gorm:"type:varchar(255);not null;uniqueIndex" json:"exchange_event_id"`
	OrganizerEmail  string     `gorm:"type:varchar(255);not null;index" json:"organizer_email"`
	Subject         string     `gorm:"type:varchar(255);not null" json:"subject"`
	StartAt         time.Time  `gorm:"index;not null" json:"start_at"`
	EndAt           time.Time  `gorm:"index;not null" json:"end_at"`
	Status          string     `gorm:"type:varchar(32);index;not null;default:'scheduled'" json:"status"`
	SyncSource      string     `gorm:"type:varchar(32);not null;default:'focus'" json:"sync_source"`
	LastSyncAt      *time.Time `gorm:"index" json:"last_sync_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	Room *Room `gorm:"foreignKey:RoomID" json:"room,omitempty"`
}

func (MeetingLink) TableName() string {
	return "meeting_links"
}

