package jitsi

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// UserContext контекст пользователя для JWT
type UserContext struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Moderator bool   `json:"moderator"`
	AvatarURL string `json:"avatar,omitempty"`
}

// JitsiClaims claims для Jitsi JWT
type JitsiClaims struct {
	Context struct {
		User UserContext `json:"user"`
	} `json:"context"`
	Room string `json:"room"`
	jwt.RegisteredClaims
}

// Config конфигурация Jitsi JWT
type Config struct {
	BaseURL       string
	AppID         string
	AppSecret     string
	Issuer        string
	Audience      string
	TokenLifetime time.Duration
}

// TokenGenerator генератор токенов для Jitsi
type TokenGenerator struct {
	config Config
}

// NewTokenGenerator создаёт новый TokenGenerator
func NewTokenGenerator(baseURL, appID, appSecret, issuer, audience string, tokenLifetime time.Duration) *TokenGenerator {
	return &TokenGenerator{
		config: Config{
			BaseURL:       baseURL,
			AppID:         appID,
			AppSecret:     appSecret,
			Issuer:        issuer,
			Audience:      audience,
			TokenLifetime: tokenLifetime,
		},
	}
}

// GenerateToken генерирует JWT токен для доступа к комнате Jitsi
func (g *TokenGenerator) GenerateToken(roomName string, user UserContext) (string, error) {
	now := time.Now()
	exp := now.Add(g.config.TokenLifetime)

	claims := JitsiClaims{}
	claims.Context.User = user
	claims.Room = roomName
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    g.config.Issuer,
		Audience:  jwt.ClaimStrings{g.config.Audience},
		ExpiresAt: jwt.NewNumericDate(exp),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(g.config.AppSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GenerateTokenForUser генерирует токен для пользователя
func (g *TokenGenerator) GenerateTokenForUser(roomName, userID, userName, userEmail string, isModerator bool) (string, error) {
	userCtx := UserContext{
		ID:        userID,
		Name:      userName,
		Email:     userEmail,
		Moderator: isModerator,
	}

	return g.GenerateToken(roomName, userCtx)
}

// GenerateRoomURL генерирует полный URL для входа в комнату
func (g *TokenGenerator) GenerateRoomURL(roomName string, user UserContext) (string, string, error) {
	token, err := g.GenerateToken(roomName, user)
	if err != nil {
		return "", "", err
	}

	url := g.config.BaseURL + "/" + roomName + "?jwt=" + token
	return url, token, nil
}

// ValidateToken проверяет JWT токен
func (g *TokenGenerator) ValidateToken(tokenString string) (*JitsiClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JitsiClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(g.config.AppSecret), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	claims, ok := token.Claims.(*JitsiClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid claims")
	}

	return claims, nil
}

// GetTokenLifetime возвращает время жизни токена
func (g *TokenGenerator) GetTokenLifetime() time.Duration {
	return g.config.TokenLifetime
}

// GetBaseURL возвращает базовый URL Jitsi
func (g *TokenGenerator) BaseURL() string {
	return g.config.BaseURL
}

// SetBaseURL устанавливает базовый URL Jitsi
func (g *TokenGenerator) SetBaseURL(url string) {
	g.config.BaseURL = url
}
