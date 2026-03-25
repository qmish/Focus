package models

import (
	"time"

	"github.com/google/uuid"
)

// CalendarAuditEvent stores Exchange calendar operation audit trail.
type CalendarAuditEvent struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Operation string    `gorm:"type:varchar(32);index;not null" json:"operation"`
	Status    string    `gorm:"type:varchar(32);index;not null" json:"status"`
	EventID   string    `gorm:"type:varchar(128);index" json:"event_id,omitempty"`
	UserID    string    `gorm:"type:varchar(64);index" json:"user_id,omitempty"`
	UserEmail string    `gorm:"type:varchar(255);index" json:"user_email,omitempty"`
	Details   string    `gorm:"type:text" json:"details,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName returns table name for calendar audit events.
func (CalendarAuditEvent) TableName() string {
	return "calendar_audit_events"
}
