package models

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	ActorEmail   string    `gorm:"type:varchar(255);index;not null" json:"actor_email"`
	Action       string    `gorm:"type:varchar(64);index;not null" json:"action"`
	ResourceType string    `gorm:"type:varchar(64);index;not null" json:"resource_type"`
	ResourceID   string    `gorm:"type:varchar(128)" json:"resource_id"`
	Details      string    `gorm:"type:text" json:"details"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
