package models

import "time"

// AppSetting stores global application appearance and branding settings (singleton row, id="default").
type AppSetting struct {
	ID                  string `gorm:"type:varchar(32);primary_key" json:"id"`
	ThemeMode           string `gorm:"type:varchar(16);not null;default:'system'" json:"theme_mode"`
	ChatAccentColor     string `gorm:"type:varchar(32);not null;default:'#89b4fa'" json:"chat_accent_color"`
	ChatBgPrimary       string `gorm:"type:varchar(32);not null;default:'#1e1e2e'" json:"chat_bg_primary"`
	ChatBgSecondary     string `gorm:"type:varchar(32);not null;default:'#181825'" json:"chat_bg_secondary"`
	ChatTextPrimary     string `gorm:"type:varchar(32);not null;default:'#cdd6f4'" json:"chat_text_primary"`
	ConferenceThemeJSON string `gorm:"type:text;not null;default:'{}'" json:"conference_theme_json"`
	BrandingProductName string `gorm:"type:varchar(128);not null;default:'Focus'" json:"branding_product_name"`
	BrandingLogoURL     string `gorm:"type:varchar(512);not null;default:'/logo.png'" json:"branding_logo_url"`
	UpdatedBy           string `gorm:"type:varchar(255)" json:"updated_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (AppSetting) TableName() string {
	return "app_settings"
}
