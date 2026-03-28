package models

import (
	"time"

	"github.com/google/uuid"
)

// BotSetting stores admin-managed bot runtime settings.
type BotSetting struct {
	ID           uuid.UUID   `gorm:"type:uuid;primary_key" json:"id"`
	Name         string      `gorm:"type:varchar(128);uniqueIndex;not null" json:"name"`
	Description  string      `gorm:"type:text" json:"description"`
	IsEnabled    bool        `gorm:"not null;default:true" json:"is_enabled"`
	RateLimitMs  int         `gorm:"not null;default:2000" json:"rate_limit_ms"`
	AllowedRooms StringArray `gorm:"type:text[]" json:"allowed_rooms"`
	CommandsJSON string      `gorm:"type:text;not null;default:'[]'" json:"commands_json"`
	ScheduleJSON string      `gorm:"type:text;not null;default:'[]'" json:"schedule_json"`
	BotUserID    *uuid.UUID  `gorm:"type:uuid" json:"bot_user_id,omitempty"`
	AvatarURL    string      `gorm:"type:varchar(512)" json:"avatar_url,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

func (BotSetting) TableName() string {
	return "bot_settings"
}

// BotScheduleEntry represents a single scheduled message.
type BotScheduleEntry struct {
	Cron    string `json:"cron"`
	RoomID  string `json:"room_id"`
	Message string `json:"message"`
}

// BotReminder stores user-created reminders.
type BotReminder struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	RoomID    string    `gorm:"type:varchar(64);index" json:"room_id"`
	UserID    string    `gorm:"type:varchar(64)" json:"user_id"`
	Message   string    `gorm:"type:text" json:"message"`
	FireAt    time.Time `gorm:"index" json:"fire_at"`
	Fired     bool      `gorm:"not null;default:false" json:"fired"`
	CreatedAt time.Time `json:"created_at"`
}

func (BotReminder) TableName() string {
	return "bot_reminders"
}

// BotDialogState stores multi-step dialog conversation state.
type BotDialogState struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	UserID    string    `gorm:"type:varchar(64);index" json:"user_id"`
	RoomID    string    `gorm:"type:varchar(64);index" json:"room_id"`
	BotName   string    `gorm:"type:varchar(128)" json:"bot_name"`
	StepIndex int       `gorm:"not null;default:0" json:"step_index"`
	StateJSON string    `gorm:"type:text" json:"state_json"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (BotDialogState) TableName() string {
	return "bot_dialog_states"
}

// BotTemplate represents a pre-built bot configuration.
type BotTemplate struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Category     string `json:"category"`
	CommandsJSON string `json:"commands_json"`
	ScheduleJSON string `json:"schedule_json,omitempty"`
}
