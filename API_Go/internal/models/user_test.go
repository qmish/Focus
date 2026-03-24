package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewUser(t *testing.T) {
	keycloakID := uuid.New()
	email := "test@example.com"
	name := "Test User"

	user := NewUser(keycloakID, email, name)

	assert.Equal(t, email, user.Email)
	assert.Equal(t, name, user.Name)
	assert.Equal(t, keycloakID, user.KeycloakID)
	assert.True(t, user.IsActive)
	assert.Contains(t, user.Roles, "user")
	assert.NotEmpty(t, user.ID)
	assert.NotEmpty(t, user.CreatedAt)
	assert.NotEmpty(t, user.UpdatedAt)
}

func TestUserHasRole(t *testing.T) {
	user := &User{
		Roles: []string{"user", "moderator"},
	}

	assert.True(t, user.HasRole("user"))
	assert.True(t, user.HasRole("moderator"))
	assert.False(t, user.HasRole("admin"))
	assert.False(t, user.HasRole(""))
}

func TestUserAddRole(t *testing.T) {
	user := &User{
		Roles: []string{"user"},
	}

	assert.False(t, user.HasRole("admin"))

	user.AddRole("admin")
	assert.True(t, user.HasRole("admin"))
	assert.Len(t, user.Roles, 2)

	// Добавление существующей роли не дублирует
	user.AddRole("user")
	assert.Len(t, user.Roles, 2)
}

func TestUserTableName(t *testing.T) {
	user := User{}
	assert.Equal(t, "users", user.TableName())
}

func TestUserSerialization(t *testing.T) {
	keycloakID := uuid.New()
	now := time.Now()
	user := &User{
		ID:          uuid.New(),
		KeycloakID:  keycloakID,
		Email:       "test@example.com",
		Name:        "Test User",
		AvatarURL:   "https://example.com/avatar.jpg",
		Roles:       []string{"user", "moderator"},
		IsActive:    true,
		LastLoginAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, user.ID, user.ID)
	assert.Equal(t, user.Email, user.Email)
	assert.Equal(t, user.Name, user.Name)
}
