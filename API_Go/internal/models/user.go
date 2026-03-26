package models

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// StringArray handles PostgreSQL text[] scanning.
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return "{" + strings.Join(a, ",") + "}", nil
}

func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}
	var raw string
	switch v := value.(type) {
	case []byte:
		raw = string(v)
	case string:
		raw = v
	default:
		return fmt.Errorf("StringArray.Scan: unsupported type %T", value)
	}
	raw = strings.TrimSpace(raw)
	if raw == "{}" || raw == "" {
		*a = StringArray{}
		return nil
	}
	raw = strings.TrimPrefix(raw, "{")
	raw = strings.TrimSuffix(raw, "}")
	*a = strings.Split(raw, ",")
	return nil
}

// User представляет пользователя системы
type User struct {
	ID           uuid.UUID   `gorm:"type:uuid;primary_key" json:"id"`
	KeycloakID   *uuid.UUID  `gorm:"type:uuid;uniqueIndex" json:"keycloak_id"`
	Email        string      `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Name         string      `gorm:"type:varchar(255);not null" json:"name"`
	PasswordHash string      `gorm:"type:varchar(255)" json:"-"`
	AvatarURL    string      `gorm:"type:varchar(512)" json:"avatar_url"`
	Roles        StringArray `gorm:"type:text[]" json:"roles"`
	IsActive     bool        `gorm:"not null;default:true" json:"is_active"`
	BannedUntil  *time.Time  `json:"banned_until,omitempty"`
	LastLoginAt  *time.Time  `json:"last_login_at"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`

	Rooms        []Room            `gorm:"foreignKey:CreatorID" json:"rooms"`
	Messages     []Message         `gorm:"foreignKey:UserID" json:"-"`
	Participants []RoomParticipant `gorm:"foreignKey:UserID" json:"-"`
}

// TableName возвращает имя таблицы
func (User) TableName() string {
	return "users"
}

// NewUser создаёт нового пользователя
func NewUser(keycloakID uuid.UUID, email, name string) *User {
	now := time.Now()
	kid := keycloakID
	return &User{
		ID:         uuid.New(),
		KeycloakID: &kid,
		Email:      email,
		Name:       name,
		Roles:      StringArray{"user"},
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// HasRole проверяет наличие роли
func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// AddRole добавляет роль
func (u *User) AddRole(role string) {
	if !u.HasRole(role) {
		u.Roles = append(u.Roles, role)
	}
}
