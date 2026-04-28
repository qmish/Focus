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

// JitsiClaims claims для Jitsi JWT.
//
// ВАЖНО: поле Audience сериализуется как одиночная строка, а не массив,
// потому что lua-jwt-библиотека Jitsi (luajwtjitsi.lib.lua) сравнивает
// claim с принимаемым audience через прямое равенство строк
// (verify_claim: claim == accepted) и не поддерживает массив. Стандартный
// jwt.RegisteredClaims использует ClaimStrings, который всегда даёт массив,
// поэтому используем собственные поля.
type JitsiClaims struct {
	Context struct {
		User UserContext `json:"user"`
	} `json:"context"`
	Room      string           `json:"room"`
	Issuer    string           `json:"iss,omitempty"`
	Subject   string           `json:"sub,omitempty"`
	Audience  string           `json:"aud,omitempty"`
	ExpiresAt *jwt.NumericDate `json:"exp,omitempty"`
	NotBefore *jwt.NumericDate `json:"nbf,omitempty"`
	IssuedAt  *jwt.NumericDate `json:"iat,omitempty"`
}

// GetExpirationTime реализует интерфейс jwt.Claims.
func (c JitsiClaims) GetExpirationTime() (*jwt.NumericDate, error) { return c.ExpiresAt, nil }

// GetIssuedAt реализует интерфейс jwt.Claims.
func (c JitsiClaims) GetIssuedAt() (*jwt.NumericDate, error) { return c.IssuedAt, nil }

// GetNotBefore реализует интерфейс jwt.Claims.
func (c JitsiClaims) GetNotBefore() (*jwt.NumericDate, error) { return c.NotBefore, nil }

// GetIssuer реализует интерфейс jwt.Claims.
func (c JitsiClaims) GetIssuer() (string, error) { return c.Issuer, nil }

// GetSubject реализует интерфейс jwt.Claims.
func (c JitsiClaims) GetSubject() (string, error) { return c.Subject, nil }

// GetAudience реализует интерфейс jwt.Claims. Для совместимости со
// стандартом возвращает audience как ClaimStrings, хотя в JSON хранится
// одиночная строка.
func (c JitsiClaims) GetAudience() (jwt.ClaimStrings, error) {
	if c.Audience == "" {
		return nil, nil
	}
	return jwt.ClaimStrings{c.Audience}, nil
}

// Config конфигурация Jitsi JWT
type Config struct {
	BaseURL       string
	AppID         string
	AppSecret     string
	Issuer        string
	Audience      string
	Subject       string
	TokenLifetime time.Duration
}

// TokenGenerator генератор токенов для Jitsi
type TokenGenerator struct {
	config Config
}

// defaultSubject значение sub claim по умолчанию.
// "*" разрешает любой XMPP-домен и совместим с настройкой
// prosody enable_domain_verification = false. Используется,
// если конкретный домен (например, meet.jitsi) не задан явно.
const defaultSubject = "*"

// NewTokenGenerator создаёт новый TokenGenerator.
// subject задаёт значение JWT claim "sub". Если передана пустая строка —
// используется wildcard "*".
func NewTokenGenerator(baseURL, appID, appSecret, issuer, audience, subject string, tokenLifetime time.Duration) *TokenGenerator {
	if subject == "" {
		subject = defaultSubject
	}
	return &TokenGenerator{
		config: Config{
			BaseURL:       baseURL,
			AppID:         appID,
			AppSecret:     appSecret,
			Issuer:        issuer,
			Audience:      audience,
			Subject:       subject,
			TokenLifetime: tokenLifetime,
		},
	}
}

// GenerateToken генерирует JWT токен для доступа к комнате Jitsi
func (g *TokenGenerator) GenerateToken(roomName string, user UserContext) (string, error) {
	now := time.Now()
	exp := now.Add(g.config.TokenLifetime)

	subject := g.config.Subject
	if subject == "" {
		subject = defaultSubject
	}
	claims := JitsiClaims{
		Room:      roomName,
		Issuer:    g.config.Issuer,
		Subject:   subject,
		Audience:  g.config.Audience,
		ExpiresAt: jwt.NewNumericDate(exp),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
	}
	claims.Context.User = user

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
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
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
