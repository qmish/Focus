package models

import (
	"time"

	"github.com/google/uuid"
)

// User представляет пользователя системы
type User struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	KeycloakID  uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"keycloak_id"`
	Email       string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Name        string     `gorm:"type:varchar(255);not null" json:"name"`
	AvatarURL   string     `gorm:"type:varchar(512)" json:"avatar_url"`
	Roles       []string   `gorm:"type:text[]" json:"roles"`
	IsActive    bool       `gorm:"not null;default:true" json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

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
	return &User{
		ID:         uuid.New(),
		KeycloakID: keycloakID,
		Email:      email,
		Name:       name,
		Roles:      []string{"user"},
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
