package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "host=localhost port=5432 user=focus password=focus dbname=focus_test sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		t.Skipf("Skipping DB-dependent test: %v", err)
	}
	require.NoError(t, db.AutoMigrate(&models.User{}))
	t.Cleanup(func() {
		db.Exec("DELETE FROM users WHERE email LIKE '%@test-getorcreate.local'")
	})
	return db
}

// TestUserRepository_GetOrCreate_LinksExistingByEmail проверяет, что при
// SSO-логине, когда в БД уже существует пользователь с тем же email, но без
// keycloak_id (или с другим), GetOrCreate привязывает существующую запись к
// новому keycloak_id, а не падает на unique constraint email.
func TestUserRepository_GetOrCreate_LinksExistingByEmail(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	email := "linked-" + uuid.NewString()[:8] + "@test-getorcreate.local"

	now := time.Now()
	existing := &models.User{
		ID:        uuid.New(),
		Email:     email,
		Name:      "Local Registered",
		Roles:     models.StringArray{"user"},
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, db.Create(existing).Error)

	newKeycloakID := uuid.New()

	user, err := repo.GetOrCreate(ctx, newKeycloakID, email, "Local Registered")
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.Equal(t, existing.ID, user.ID, "запись должна остаться той же по ID")
	require.NotNil(t, user.KeycloakID)
	assert.Equal(t, newKeycloakID, *user.KeycloakID, "должен быть прописан новый keycloak_id")
	assert.Equal(t, email, user.Email)
}

// TestUserRepository_GetOrCreate_CreatesNewWhenNoExisting проверяет создание
// нового пользователя через SSO, когда в БД нет записи ни по keycloak_id, ни
// по email.
func TestUserRepository_GetOrCreate_CreatesNewWhenNoExisting(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	email := "new-" + uuid.NewString()[:8] + "@test-getorcreate.local"
	keycloakID := uuid.New()

	user, err := repo.GetOrCreate(ctx, keycloakID, email, "Новый Пользователь")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, user.KeycloakID)
	assert.Equal(t, keycloakID, *user.KeycloakID)
	assert.Equal(t, email, user.Email)
	assert.Contains(t, []string(user.Roles), "user")
}

// TestUserRepository_GetOrCreate_IdempotentByKeycloakID проверяет, что
// повторный GetOrCreate с тем же keycloak_id возвращает ту же запись.
func TestUserRepository_GetOrCreate_IdempotentByKeycloakID(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	email := "idem-" + uuid.NewString()[:8] + "@test-getorcreate.local"
	keycloakID := uuid.New()

	first, err := repo.GetOrCreate(ctx, keycloakID, email, "User One")
	require.NoError(t, err)

	second, err := repo.GetOrCreate(ctx, keycloakID, email, "User One")
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID, "запись должна быть той же")
}
