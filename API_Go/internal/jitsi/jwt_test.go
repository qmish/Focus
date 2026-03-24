package jitsi

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTokenGenerator(t *testing.T) {
	baseURL := "https://meet.company.com"
	appID := "jitsi"
	appSecret := "secret"
	issuer := "jitsi"
	audience := "jitsi"
	tokenLifetime := 8 * time.Hour

	gen := NewTokenGenerator(baseURL, appID, appSecret, issuer, audience, tokenLifetime)

	require.NotNil(t, gen)
	assert.Equal(t, baseURL, gen.config.BaseURL)
	assert.Equal(t, appID, gen.config.AppID)
	assert.Equal(t, appSecret, gen.config.AppSecret)
	assert.Equal(t, issuer, gen.config.Issuer)
	assert.Equal(t, audience, gen.config.Audience)
	assert.Equal(t, tokenLifetime, gen.config.TokenLifetime)
}

func TestGenerateToken(t *testing.T) {
	gen := NewTokenGenerator(
		"https://meet.company.com",
		"jitsi",
		"test-secret-key-12345",
		"jitsi",
		"jitsi",
		8*time.Hour,
	)

	roomName := "test-room"
	user := UserContext{
		ID:        uuid.New().String(),
		Name:      "Test User",
		Email:     "test@example.com",
		Moderator: true,
	}

	token, err := gen.GenerateToken(roomName, user)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Contains(t, token, ".") // JWT имеет формат xxx.yyy.zzz
}

func TestGenerateTokenForUser(t *testing.T) {
	gen := NewTokenGenerator(
		"https://meet.company.com",
		"jitsi",
		"test-secret-key-12345",
		"jitsi",
		"jitsi",
		8*time.Hour,
	)

	roomName := "meeting-room"
	userID := uuid.New().String()
	userName := "John Doe"
	userEmail := "john@example.com"
	isModerator := false

	token, err := gen.GenerateTokenForUser(roomName, userID, userName, userEmail, isModerator)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Проверяем валидацию токена
	claims, err := gen.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, roomName, claims.Room)
	assert.Equal(t, userID, claims.Context.User.ID)
	assert.Equal(t, userName, claims.Context.User.Name)
	assert.Equal(t, userEmail, claims.Context.User.Email)
	assert.Equal(t, isModerator, claims.Context.User.Moderator)
}

func TestGenerateTokenForModerator(t *testing.T) {
	gen := NewTokenGenerator(
		"https://meet.company.com",
		"jitsi",
		"test-secret-key-12345",
		"jitsi",
		"jitsi",
		8*time.Hour,
	)

	roomName := "moderated-room"
	userID := uuid.New().String()
	userName := "Admin User"
	userEmail := "admin@example.com"
	isModerator := true

	token, err := gen.GenerateTokenForUser(roomName, userID, userName, userEmail, isModerator)

	require.NoError(t, err)

	claims, err := gen.ValidateToken(token)
	require.NoError(t, err)
	assert.True(t, claims.Context.User.Moderator)
}

func TestValidateToken(t *testing.T) {
	gen := NewTokenGenerator(
		"https://meet.company.com",
		"jitsi",
		"test-secret-key-12345",
		"jitsi",
		"jitsi",
		8*time.Hour,
	)

	roomName := "test-room"
	user := UserContext{
		ID:        uuid.New().String(),
		Name:      "Test User",
		Email:     "test@example.com",
		Moderator: false,
	}

	token, err := gen.GenerateToken(roomName, user)
	require.NoError(t, err)

	claims, err := gen.ValidateToken(token)

	require.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, roomName, claims.Room)
	assert.Equal(t, "jitsi", claims.Issuer)
	assert.Equal(t, "jitsi", claims.Audience[0])
	assert.False(t, claims.Context.User.Moderator)
}

func TestValidateTokenInvalid(t *testing.T) {
	gen := NewTokenGenerator(
		"https://meet.company.com",
		"jitsi",
		"test-secret-key-12345",
		"jitsi",
		"jitsi",
		8*time.Hour,
	)

	invalidToken := "invalid.token.here"

	claims, err := gen.ValidateToken(invalidToken)

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateTokenWrongSecret(t *testing.T) {
	gen := NewTokenGenerator(
		"https://meet.company.com",
		"jitsi",
		"test-secret-key-12345",
		"jitsi",
		"jitsi",
		8*time.Hour,
	)

	gen2 := NewTokenGenerator(
		"https://meet.company.com",
		"jitsi",
		"wrong-secret-key",
		"jitsi",
		"jitsi",
		8*time.Hour,
	)

	user := UserContext{
		ID:        uuid.New().String(),
		Name:      "Test User",
		Email:     "test@example.com",
		Moderator: false,
	}

	token, err := gen.GenerateToken("test-room", user)
	require.NoError(t, err)

	claims, err := gen2.ValidateToken(token)

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestGenerateRoomURL(t *testing.T) {
	gen := NewTokenGenerator(
		"https://meet.company.com",
		"jitsi",
		"test-secret-key-12345",
		"jitsi",
		"jitsi",
		8*time.Hour,
	)

	roomName := "conference-room"
	user := UserContext{
		ID:        uuid.New().String(),
		Name:      "Test User",
		Email:     "test@example.com",
		Moderator: true,
	}

	url, token, err := gen.GenerateRoomURL(roomName, user)

	require.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.NotEmpty(t, token)
	assert.Contains(t, url, "https://meet.company.com/"+roomName)
	assert.Contains(t, url, "jwt="+token)
}

func TestGetBaseURL(t *testing.T) {
	baseURL := "https://meet.example.com"
	gen := NewTokenGenerator(
		baseURL,
		"jitsi",
		"secret",
		"jitsi",
		"jitsi",
		8*time.Hour,
	)

	assert.Equal(t, baseURL, gen.BaseURL())
}

func TestSetBaseURL(t *testing.T) {
	gen := NewTokenGenerator(
		"https://meet.company.com",
		"jitsi",
		"secret",
		"jitsi",
		"jitsi",
		8*time.Hour,
	)

	newBaseURL := "https://meet.new-domain.com"
	gen.SetBaseURL(newBaseURL)

	assert.Equal(t, newBaseURL, gen.BaseURL())
}

func TestTokenExpiration(t *testing.T) {
	// Токен с коротким временем жизни для теста
	gen := NewTokenGenerator(
		"https://meet.company.com",
		"jitsi",
		"test-secret-key-12345",
		"jitsi",
		"jitsi",
		1*time.Second,
	)

	user := UserContext{
		ID:        uuid.New().String(),
		Name:      "Test User",
		Email:     "test@example.com",
		Moderator: false,
	}

	token, err := gen.GenerateToken("test-room", user)
	require.NoError(t, err)

	// Токен должен быть валиден сразу
	claims, err := gen.ValidateToken(token)
	require.NoError(t, err)
	assert.NotNil(t, claims)

	// Ждём истечения времени
	time.Sleep(2 * time.Second)

	// Токен должен быть невалиден
	claims, err = gen.ValidateToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJitsiClaimsStructure(t *testing.T) {
	gen := NewTokenGenerator(
		"https://meet.company.com",
		"jitsi",
		"test-secret-key-12345",
		"jitsi",
		"jitsi",
		8*time.Hour,
	)

	roomName := "test-room"
	userID := uuid.New().String()
	userName := "Test User"
	userEmail := "test@example.com"
	isModerator := true

	token, err := gen.GenerateTokenForUser(roomName, userID, userName, userEmail, isModerator)
	require.NoError(t, err)

	claims, err := gen.ValidateToken(token)
	require.NoError(t, err)

	// Проверяем структуру claims
	assert.Equal(t, "jitsi", claims.Issuer)
	assert.Contains(t, claims.Audience, "jitsi")
	assert.Equal(t, roomName, claims.Room)
	assert.Equal(t, userID, claims.Context.User.ID)
	assert.Equal(t, userName, claims.Context.User.Name)
	assert.Equal(t, userEmail, claims.Context.User.Email)
	assert.Equal(t, isModerator, claims.Context.User.Moderator)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
	assert.NotNil(t, claims.NotBefore)
}
