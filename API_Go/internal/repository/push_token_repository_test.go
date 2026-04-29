package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func newPushTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "host=localhost port=5432 user=focus password=focus dbname=focus_test sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		t.Skipf("Skipping DB-dependent test: %v", err)
	}
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.PushToken{}))
	t.Cleanup(func() {
		db.Exec("DELETE FROM push_tokens WHERE endpoint LIKE 'test-pt-%'")
		db.Exec("DELETE FROM users WHERE email LIKE '%@test-pushtoken.local'")
	})
	return db
}

func makeUser(t *testing.T, db *gorm.DB) *models.User {
	t.Helper()
	u := &models.User{
		ID:       uuid.New(),
		Email:    "user-" + uuid.NewString()[:8] + "@test-pushtoken.local",
		Name:     "PT User",
		Roles:    models.StringArray{"user"},
		IsActive: true,
	}
	require.NoError(t, db.Create(u).Error)
	return u
}

func TestPushTokenRepository_UpsertAndGet(t *testing.T) {
	db := newPushTestDB(t)
	repo := NewPushTokenRepository(db)
	ctx := context.Background()

	u := makeUser(t, db)
	endpoint := "test-pt-" + uuid.NewString()[:8]
	tok := models.NewWebPushToken(u.ID, endpoint, "p1", "a1")
	tok.UserAgent = "Mozilla/5.0"

	require.NoError(t, repo.Upsert(ctx, tok))
	require.NotEqual(t, uuid.Nil, tok.ID)

	got, err := repo.GetByEndpoint(ctx, endpoint)
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.UserID)
	assert.Equal(t, models.PushPlatformWeb, got.Platform)
	assert.Equal(t, "p1", got.P256DHKey)
	assert.Equal(t, "a1", got.AuthKey)
	assert.Equal(t, "Mozilla/5.0", got.UserAgent)
}

func TestPushTokenRepository_Upsert_UpdatesExisting(t *testing.T) {
	db := newPushTestDB(t)
	repo := NewPushTokenRepository(db)
	ctx := context.Background()

	u1 := makeUser(t, db)
	u2 := makeUser(t, db)
	endpoint := "test-pt-" + uuid.NewString()[:8]

	require.NoError(t, repo.Upsert(ctx, models.NewWebPushToken(u1.ID, endpoint, "old-p", "old-a")))
	// Тот же endpoint, но другой пользователь и обновлённые ключи.
	require.NoError(t, repo.Upsert(ctx, models.NewWebPushToken(u2.ID, endpoint, "new-p", "new-a")))

	got, err := repo.GetByEndpoint(ctx, endpoint)
	require.NoError(t, err)
	assert.Equal(t, u2.ID, got.UserID, "endpoint должен быть переназначен новому пользователю")
	assert.Equal(t, "new-p", got.P256DHKey)
	assert.Equal(t, "new-a", got.AuthKey)
}

func TestPushTokenRepository_GetByEndpoint_NotFound(t *testing.T) {
	db := newPushTestDB(t)
	repo := NewPushTokenRepository(db)
	ctx := context.Background()

	_, err := repo.GetByEndpoint(ctx, "test-pt-nonexistent-"+uuid.NewString()[:8])
	assert.ErrorIs(t, err, ErrPushTokenNotFound)
}

func TestPushTokenRepository_ListByUser(t *testing.T) {
	db := newPushTestDB(t)
	repo := NewPushTokenRepository(db)
	ctx := context.Background()

	u := makeUser(t, db)
	require.NoError(t, repo.Upsert(ctx, models.NewWebPushToken(u.ID, "test-pt-l1-"+uuid.NewString()[:8], "p", "a")))
	require.NoError(t, repo.Upsert(ctx, models.NewMobilePushToken(u.ID, models.PushPlatformFCM, "test-pt-l2-"+uuid.NewString()[:8])))

	tokens, err := repo.ListByUser(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, tokens, 2)
}

func TestPushTokenRepository_ListByUsers_Bulk(t *testing.T) {
	db := newPushTestDB(t)
	repo := NewPushTokenRepository(db)
	ctx := context.Background()

	u1 := makeUser(t, db)
	u2 := makeUser(t, db)
	require.NoError(t, repo.Upsert(ctx, models.NewWebPushToken(u1.ID, "test-pt-b1-"+uuid.NewString()[:8], "p", "a")))
	require.NoError(t, repo.Upsert(ctx, models.NewWebPushToken(u2.ID, "test-pt-b2-"+uuid.NewString()[:8], "p", "a")))

	tokens, err := repo.ListByUsers(ctx, []uuid.UUID{u1.ID, u2.ID})
	require.NoError(t, err)
	assert.Len(t, tokens, 2)
}

func TestPushTokenRepository_DeleteByUserAndEndpoint(t *testing.T) {
	db := newPushTestDB(t)
	repo := NewPushTokenRepository(db)
	ctx := context.Background()

	u := makeUser(t, db)
	other := makeUser(t, db)
	endpoint := "test-pt-del-" + uuid.NewString()[:8]
	require.NoError(t, repo.Upsert(ctx, models.NewWebPushToken(u.ID, endpoint, "p", "a")))

	// Чужой пользователь не может удалить
	require.NoError(t, repo.DeleteByUserAndEndpoint(ctx, other.ID, endpoint))
	_, err := repo.GetByEndpoint(ctx, endpoint)
	assert.NoError(t, err, "токен не должен быть удалён чужим пользователем")

	// Владелец может удалить
	require.NoError(t, repo.DeleteByUserAndEndpoint(ctx, u.ID, endpoint))
	_, err = repo.GetByEndpoint(ctx, endpoint)
	assert.ErrorIs(t, err, ErrPushTokenNotFound)
}

func TestPushTokenRepository_TouchLastUsed(t *testing.T) {
	db := newPushTestDB(t)
	repo := NewPushTokenRepository(db)
	ctx := context.Background()

	u := makeUser(t, db)
	endpoint := "test-pt-touch-" + uuid.NewString()[:8]
	tok := models.NewWebPushToken(u.ID, endpoint, "p", "a")
	require.NoError(t, repo.Upsert(ctx, tok))

	require.NoError(t, repo.TouchLastUsed(ctx, tok.ID))

	got, err := repo.GetByEndpoint(ctx, endpoint)
	require.NoError(t, err)
	require.NotNil(t, got.LastUsedAt)
}
