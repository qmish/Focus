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
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

func (BotSetting) TableName() string {
	return "bot_settings"
}
