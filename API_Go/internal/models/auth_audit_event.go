package models

import (
	"time"

	"github.com/google/uuid"
)

// AuthAuditEvent stores authentication and authorization audit records.
type AuthAuditEvent struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Action    string    `gorm:"type:varchar(64);index;not null" json:"action"`
	Status    string    `gorm:"type:varchar(32);index;not null" json:"status"`
	UserID    string    `gorm:"type:varchar(64);index" json:"user_id,omitempty"`
	UserEmail string    `gorm:"type:varchar(255);index" json:"user_email,omitempty"`
	ClientIP  string    `gorm:"type:varchar(64)" json:"client_ip,omitempty"`
	UserAgent string    `gorm:"type:text" json:"user_agent,omitempty"`
	Error     string    `gorm:"type:text" json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName returns table name for auth audit events.
func (AuthAuditEvent) TableName() string {
	return "auth_audit_events"
}
