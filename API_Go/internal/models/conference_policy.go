package models

import "time"

// ConferencePolicy stores admin-managed conference default policies (singleton, id="default").
type ConferencePolicy struct {
	ID                 string `gorm:"type:varchar(32);primary_key" json:"id"`
	MaxParticipants    int    `gorm:"not null;default:100" json:"max_participants"`
	MaxDurationMinutes int    `gorm:"not null;default:480" json:"max_duration_minutes"`
	RecordingEnabled   bool   `gorm:"not null;default:false" json:"recording_enabled"`
	LobbyEnabled       bool   `gorm:"not null;default:false" json:"lobby_enabled"`
	AutoMuteOnJoin     bool   `gorm:"not null;default:false" json:"auto_mute_on_join"`
	RequirePassword    bool   `gorm:"not null;default:false" json:"require_password"`
	UpdatedBy          string `gorm:"type:varchar(255)" json:"updated_by"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (ConferencePolicy) TableName() string {
	return "conference_policies"
}
