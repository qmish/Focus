package models

import "time"

// ExchangeSetting stores UI-managed on-prem EWS integration configuration.
type ExchangeSetting struct {
	ID             string    `gorm:"type:varchar(32);primaryKey" json:"id"`
	EWSURL         string    `gorm:"type:varchar(512);not null" json:"ews_url"`
	Username       string    `gorm:"type:varchar(255);not null" json:"username"`
	Password       string    `gorm:"type:text;not null" json:"-"`
	Domain         string    `gorm:"type:varchar(255)" json:"domain"`
	AuthMode       string    `gorm:"type:varchar(32);not null;default:'basic'" json:"auth_mode"`
	CACertPath     string    `gorm:"type:varchar(512)" json:"ca_cert_path"`
	InsecureTLS    bool      `gorm:"not null;default:false" json:"insecure_tls"`
	Krb5ConfigPath string    `gorm:"type:varchar(512)" json:"krb5_config_path"`
	Krb5KeytabPath string    `gorm:"type:varchar(512)" json:"krb5_keytab_path"`
	Krb5Realm      string    `gorm:"type:varchar(255)" json:"krb5_realm"`
	Krb5SPN        string    `gorm:"type:varchar(255)" json:"krb5_spn"`
	Impersonation  bool      `gorm:"not null;default:true" json:"impersonation"`
	TimeoutSeconds int       `gorm:"not null;default:15" json:"timeout_seconds"`
	SyncEnabled    bool      `gorm:"not null;default:false" json:"sync_enabled"`
	SyncIntervalS  int       `gorm:"not null;default:120" json:"sync_interval_seconds"`
	SyncLookbackS  int       `gorm:"not null;default:43200" json:"sync_lookback_seconds"`
	SyncLookaheadS int       `gorm:"not null;default:1209600" json:"sync_lookahead_seconds"`
	UpdatedBy      string    `gorm:"type:varchar(255)" json:"updated_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (ExchangeSetting) TableName() string {
	return "exchange_settings"
}
