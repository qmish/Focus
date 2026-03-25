package models

import "time"

// RevokedSession stores revoked API session IDs for persistent invalidation.
type RevokedSession struct {
	SessionID string    `gorm:"type:varchar(128);primary_key" json:"session_id"`
	ExpiresAt time.Time `gorm:"index;not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName returns table name for revoked sessions.
func (RevokedSession) TableName() string {
	return "revoked_sessions"
}
