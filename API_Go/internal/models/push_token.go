package models

import (
	"time"

	"github.com/google/uuid"
)

// PushPlatform — тип push-провайдера (Web Push, FCM, APNs).
type PushPlatform string

const (
	PushPlatformWeb  PushPlatform = "web"
	PushPlatformFCM  PushPlatform = "fcm"
	PushPlatformAPNS PushPlatform = "apns"
)

// PushToken хранит подписку на push-уведомления для конкретного устройства/браузера
// одного пользователя. Для Web Push — endpoint + ключи p256dh/auth.
// Для FCM/APNs — registration token устройства.
type PushToken struct {
	ID       uuid.UUID    `gorm:"type:uuid;primary_key" json:"id"`
	UserID   uuid.UUID    `gorm:"type:uuid;not null;index" json:"user_id"`
	Platform PushPlatform `gorm:"type:varchar(16);not null;index" json:"platform"`

	// Endpoint:
	//   - Web Push: URL подписки (https://...).
	//   - FCM/APNs: registration token устройства.
	Endpoint string `gorm:"type:text;not null;uniqueIndex:idx_push_endpoint" json:"endpoint"`

	// Web Push public key + auth secret (Base64URL без padding).
	P256DHKey string `gorm:"type:text" json:"p256dh,omitempty"`
	AuthKey   string `gorm:"type:text" json:"auth,omitempty"`

	// Опциональные поля для отладки/UX.
	UserAgent string `gorm:"type:varchar(512)" json:"user_agent,omitempty"`
	Locale    string `gorm:"type:varchar(16)" json:"locale,omitempty"`

	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	User *User `gorm:"foreignKey:UserID" json:"-"`
}

// TableName возвращает имя таблицы.
func (PushToken) TableName() string {
	return "push_tokens"
}

// IsWebPush — удобный helper.
func (t *PushToken) IsWebPush() bool {
	return t.Platform == PushPlatformWeb
}

// NewWebPushToken создаёт новый Web Push токен для пользователя.
func NewWebPushToken(userID uuid.UUID, endpoint, p256dh, auth string) *PushToken {
	now := time.Now()
	return &PushToken{
		ID:        uuid.New(),
		UserID:    userID,
		Platform:  PushPlatformWeb,
		Endpoint:  endpoint,
		P256DHKey: p256dh,
		AuthKey:   auth,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewMobilePushToken создаёт новый FCM/APNs токен для пользователя.
func NewMobilePushToken(userID uuid.UUID, platform PushPlatform, registrationToken string) *PushToken {
	now := time.Now()
	return &PushToken{
		ID:        uuid.New(),
		UserID:    userID,
		Platform:  platform,
		Endpoint:  registrationToken,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
