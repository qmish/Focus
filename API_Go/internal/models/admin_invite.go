package models

import (
	"time"

	"github.com/google/uuid"
)

type AdminInviteStatus string

const (
	AdminInviteStatusPending  AdminInviteStatus = "pending"
	AdminInviteStatusSent     AdminInviteStatus = "sent"
	AdminInviteStatusAccepted AdminInviteStatus = "accepted"
	AdminInviteStatusExpired  AdminInviteStatus = "expired"
)

// AdminInvite stores email invitation lifecycle for admin onboarding.
type AdminInvite struct {
	ID         uuid.UUID         `gorm:"type:uuid;primary_key" json:"id"`
	Email      string            `gorm:"type:varchar(255);index;not null" json:"email"`
	TokenHash  string            `gorm:"type:varchar(128);uniqueIndex;not null" json:"-"`
	Roles      StringArray       `gorm:"type:text[]" json:"roles"`
	Status     AdminInviteStatus `gorm:"type:varchar(32);index;not null;default:'pending'" json:"status"`
	InvitedBy  string            `gorm:"type:varchar(255);not null" json:"invited_by"`
	ExpiresAt  time.Time         `gorm:"index;not null" json:"expires_at"`
	SentAt     *time.Time        `json:"sent_at,omitempty"`
	AcceptedAt *time.Time        `json:"accepted_at,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

func (AdminInvite) TableName() string {
	return "admin_invites"
}
