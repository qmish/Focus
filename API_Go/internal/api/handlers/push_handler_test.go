package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func newPushTestRepo(t *testing.T) *repository.PushTokenRepository {
	t.Helper()
	dsn := "host=localhost port=5432 user=focus password=focus dbname=focus_test sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		t.Skipf("Skipping DB-dependent test: %v", err)
	}
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.PushToken{}))
	t.Cleanup(func() {
		db.Exec("DELETE FROM push_tokens WHERE endpoint LIKE 'http-test-%'")
		db.Exec("DELETE FROM users WHERE email LIKE '%@test-pushhandler.local'")
	})
	return repository.NewPushTokenRepository(db)
}

func contextWithUser(uid uuid.UUID) context.Context {
	claims := &auth.SessionClaims{UserID: uid.String()}
	return context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
}

func TestPushHandler_GetVAPIDPublicKey_NotConfigured(t *testing.T) {
	h := NewPushHandler(nil, "")
	req := httptest.NewRequest(http.MethodGet, "/push/vapid-public-key", nil)
	rr := httptest.NewRecorder()
	h.GetVAPIDPublicKey(rr, req)
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
}

func TestPushHandler_GetVAPIDPublicKey_OK(t *testing.T) {
	h := NewPushHandler(nil, "BLM-test-key")
	req := httptest.NewRequest(http.MethodGet, "/push/vapid-public-key", nil)
	rr := httptest.NewRecorder()
	h.GetVAPIDPublicKey(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "BLM-test-key")
}

func TestPushHandler_Register_Unauthorized(t *testing.T) {
	h := NewPushHandler(nil, "")
	req := httptest.NewRequest(http.MethodPost, "/push/register",
		strings.NewReader(`{"endpoint":"http-test-x","p256dh":"x","auth":"x"}`))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestPushHandler_Register_InvalidBody(t *testing.T) {
	h := NewPushHandler(nil, "")
	req := httptest.NewRequest(http.MethodPost, "/push/register", strings.NewReader(`{`))
	req = req.WithContext(contextWithUser(uuid.New()))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestPushHandler_Register_MissingEndpoint(t *testing.T) {
	h := NewPushHandler(nil, "")
	req := httptest.NewRequest(http.MethodPost, "/push/register", strings.NewReader(`{"endpoint":""}`))
	req = req.WithContext(contextWithUser(uuid.New()))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestPushHandler_Register_WebPush_RequiresKeys(t *testing.T) {
	h := NewPushHandler(nil, "")
	req := httptest.NewRequest(http.MethodPost, "/push/register",
		strings.NewReader(`{"platform":"web","endpoint":"http-test-x"}`))
	req = req.WithContext(contextWithUser(uuid.New()))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestPushHandler_Register_UnknownPlatform(t *testing.T) {
	h := NewPushHandler(nil, "")
	req := httptest.NewRequest(http.MethodPost, "/push/register",
		strings.NewReader(`{"platform":"telegram","endpoint":"x"}`))
	req = req.WithContext(contextWithUser(uuid.New()))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestPushHandler_Register_OK_WithDB(t *testing.T) {
	repo := newPushTestRepo(t)
	h := NewPushHandler(repo, "")

	// Подготавливаем пользователя в БД, на которого можно вешать токен.
	uid := uuid.New()
	dsn := "host=localhost port=5432 user=focus password=focus dbname=focus_test sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	require.NoError(t, err)
	require.NoError(t, db.Create(&models.User{
		ID: uid, Email: "ph-" + uid.String() + "@test-pushhandler.local",
		Name: "PT Handler User", Roles: models.StringArray{"user"}, IsActive: true,
	}).Error)

	body := `{"platform":"web","endpoint":"http-test-` + uid.String() + `","p256dh":"p","auth":"a"}`
	req := httptest.NewRequest(http.MethodPost, "/push/register", strings.NewReader(body))
	req = req.WithContext(contextWithUser(uid))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Unregister
	bodyU := `{"endpoint":"http-test-` + uid.String() + `"}`
	reqU := httptest.NewRequest(http.MethodPost, "/push/unregister", strings.NewReader(bodyU))
	reqU = reqU.WithContext(contextWithUser(uid))
	rrU := httptest.NewRecorder()
	h.Unregister(rrU, reqU)
	assert.Equal(t, http.StatusNoContent, rrU.Code)
}

func TestPushHandler_Unregister_Unauthorized(t *testing.T) {
	h := NewPushHandler(nil, "")
	req := httptest.NewRequest(http.MethodPost, "/push/unregister", strings.NewReader(`{"endpoint":"x"}`))
	rr := httptest.NewRecorder()
	h.Unregister(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestPushHandler_Unregister_MissingEndpoint(t *testing.T) {
	h := NewPushHandler(nil, "")
	req := httptest.NewRequest(http.MethodPost, "/push/unregister", strings.NewReader(`{"endpoint":""}`))
	req = req.WithContext(contextWithUser(uuid.New()))
	rr := httptest.NewRecorder()
	h.Unregister(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
