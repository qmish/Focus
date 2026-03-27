package models

import "time"

// CalendarIdempotencyKey stores deduplication state for calendar create requests.
type CalendarIdempotencyKey struct {
	Key          string     `gorm:"type:varchar(128);primaryKey" json:"key"`
	UserEmail    string     `gorm:"type:varchar(255);primaryKey" json:"user_email"`
	EventID      string     `gorm:"type:varchar(255);index" json:"event_id"`
	RoomID       string     `gorm:"type:varchar(64)" json:"room_id"`
	ResponseBody string     `gorm:"type:text" json:"response_body"`
	CompletedAt  *time.Time `gorm:"index" json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (CalendarIdempotencyKey) TableName() string {
	return "calendar_idempotency_keys"
}
