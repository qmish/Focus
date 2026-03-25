package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateState(t *testing.T) {
	state, err := GenerateState()

	require.NoError(t, err)
	assert.NotEmpty(t, state)
	assert.Len(t, state, 44) // base64 encoded 32 bytes
}

func TestGenerateStateUniqueness(t *testing.T) {
	state1, err := GenerateState()
	require.NoError(t, err)

	state2, err := GenerateState()
	require.NoError(t, err)

	assert.NotEqual(t, state1, state2)
}

func TestValidateState(t *testing.T) {
	t.Run("valid state", func(t *testing.T) {
		err := ValidateState("test-state", "test-state")
		assert.NoError(t, err)
	})

	t.Run("invalid state", func(t *testing.T) {
		err := ValidateState("test-state", "wrong-state")
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidState, err)
	})

	t.Run("empty state", func(t *testing.T) {
		err := ValidateState("", "")
		assert.NoError(t, err)
	})
}

func TestGenerateSessionJWT(t *testing.T) {
	secret := []byte("test-secret-key-12345")
	lifetime := 1 * time.Hour

	userInfo := &UserInfo{
		Sub:           "user-123",
		Email:         "test@example.com",
		Name:          "Test User",
		EmailVerified: true,
		Roles:         []string{"user", "moderator"},
	}

	sessionID := "session-456"

	token, err := GenerateSessionJWT(userInfo, sessionID, secret, lifetime)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Contains(t, token, ".") // JWT format
}

func TestValidateSessionJWT(t *testing.T) {
	secret := []byte("test-secret-key-12345")
	lifetime := 1 * time.Hour

	userInfo := &UserInfo{
		Sub:   "user-123",
		Email: "test@example.com",
		Name:  "Test User",
		Roles: []string{"user"},
	}

	sessionID := "session-456"

	token, err := GenerateSessionJWT(userInfo, sessionID, secret, lifetime)
	require.NoError(t, err)

	claims, err := ValidateSessionJWT(token, secret)

	require.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, "Test User", claims.Name)
	assert.Equal(t, "session-456", claims.SessionID)
	assert.Contains(t, claims.Roles, "user")
}

func TestValidateSessionJWTInvalid(t *testing.T) {
	secret := []byte("test-secret-key-12345")
	wrongSecret := []byte("wrong-secret")

	userInfo := &UserInfo{
		Sub:   "user-123",
		Email: "test@example.com",
		Name:  "Test User",
	}

	token, err := GenerateSessionJWT(userInfo, "session-123", secret, 1*time.Hour)
	require.NoError(t, err)

	// Проверка с неправильным секретом
	claims, err := ValidateSessionJWT(token, wrongSecret)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateSessionJWTExpired(t *testing.T) {
	secret := []byte("test-secret-key-12345")
	shortLifetime := 1 * time.Second

	userInfo := &UserInfo{
		Sub:   "user-123",
		Email: "test@example.com",
		Name:  "Test User",
	}

	token, err := GenerateSessionJWT(userInfo, "session-123", secret, shortLifetime)
	require.NoError(t, err)

	// Ждём истечения времени
	time.Sleep(2 * time.Second)

	claims, err := ValidateSessionJWT(token, secret)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateSessionJWTMalformed(t *testing.T) {
	secret := []byte("test-secret-key-12345")

	claims, err := ValidateSessionJWT("invalid.token.here", secret)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestSessionClaimsStructure(t *testing.T) {
	secret := []byte("test-secret-key-12345")
	lifetime := 1 * time.Hour

	userInfo := &UserInfo{
		Sub:   "user-123",
		Email: "test@example.com",
		Name:  "Test User",
		Roles: []string{"user", "admin"},
		Scope: "focus.read focus.write",
	}

	sessionID := "session-456"

	token, err := GenerateSessionJWT(userInfo, sessionID, secret, lifetime)
	require.NoError(t, err)

	claims, err := ValidateSessionJWT(token, secret)
	require.NoError(t, err)

	// Проверка структуры claims
	assert.Equal(t, "focus-api", claims.Issuer)
	assert.Contains(t, claims.Audience, "focus-frontend")
	assert.Equal(t, "user-123", claims.KeycloakID)
	assert.Equal(t, "session-456", claims.SessionID)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
	assert.Len(t, claims.Roles, 2)
	assert.Contains(t, claims.AllScopes(), "focus.read")
	assert.Contains(t, claims.AllScopes(), "focus.write")
}

func TestGetUserClaimsFromContext(t *testing.T) {
	expectedClaims := &SessionClaims{
		UserID: "user-123",
		Email:  "test@example.com",
	}

	ctx := context.WithValue(context.Background(), ContextKeyUserClaims, expectedClaims)

	claims := GetUserClaimsFromContext(ctx)

	assert.NotNil(t, claims)
	assert.Equal(t, expectedClaims.UserID, claims.UserID)
	assert.Equal(t, expectedClaims.Email, claims.Email)
}

func TestGetUserClaimsFromContextNil(t *testing.T) {
	ctx := context.Background()

	claims := GetUserClaimsFromContext(ctx)

	assert.Nil(t, claims)
}

func TestRequireRole(t *testing.T) {
	secret := []byte("test-secret-key-12345")

	userInfo := &UserInfo{
		Sub:   "user-123",
		Email: "test@example.com",
		Name:  "Test User",
		Roles: []string{"moderator"},
	}

	token, err := GenerateSessionJWT(userInfo, "session-123", secret, 1*time.Hour)
	require.NoError(t, err)

	t.Run("user has required role", func(t *testing.T) {
		// Создаём handler с правильным контекстом
		claims, _ := ValidateSessionJWT(token, secret)
		ctx := context.WithValue(context.Background(), ContextKeyUserClaims, claims)

		handler := RequireRole("moderator")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req, _ := http.NewRequest("GET", "/test", nil)
		req = req.WithContext(ctx)

		rr := &testResponseWriter{}
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.statusCode)
	})

	t.Run("user doesn't have required role", func(t *testing.T) {
		claims, _ := ValidateSessionJWT(token, secret)
		ctx := context.WithValue(context.Background(), ContextKeyUserClaims, claims)

		handler := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req, _ := http.NewRequest("GET", "/test", nil)
		req = req.WithContext(ctx)

		rr := &testResponseWriter{}
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.statusCode)
	})

	t.Run("no claims in context", func(t *testing.T) {
		handler := RequireRole("user")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req, _ := http.NewRequest("GET", "/test", nil)

		rr := &testResponseWriter{}
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.statusCode)
	})
}

func TestAllScopesMergesScopeFormats(t *testing.T) {
	userInfo := &UserInfo{
		Scope:  "focus.admin focus.read",
		Scopes: []string{"focus.read", "focus.calendar"},
	}
	all := userInfo.AllScopes()
	assert.Len(t, all, 3)
	assert.Contains(t, all, "focus.admin")
	assert.Contains(t, all, "focus.read")
	assert.Contains(t, all, "focus.calendar")
}

func TestRequireAccess(t *testing.T) {
	t.Run("allows by role when any role matches", func(t *testing.T) {
		claims := &SessionClaims{
			Roles: []string{"admin"},
		}
		ctx := context.WithValue(context.Background(), ContextKeyUserClaims, claims)
		handler := RequireAccess(AccessRule{
			AnyRoles: []string{"admin"},
		})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req = req.WithContext(ctx)
		rr := &testResponseWriter{}
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.statusCode)
	})

	t.Run("allows by scope when any scope matches", func(t *testing.T) {
		claims := &SessionClaims{
			Scope: "focus.admin focus.read",
		}
		ctx := context.WithValue(context.Background(), ContextKeyUserClaims, claims)
		handler := RequireAccess(AccessRule{
			AnyScopes: []string{"focus.admin"},
		})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req = req.WithContext(ctx)
		rr := &testResponseWriter{}
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.statusCode)
	})

	t.Run("requires all scopes when configured", func(t *testing.T) {
		claims := &SessionClaims{
			Scopes: []string{"focus.read"},
		}
		ctx := context.WithValue(context.Background(), ContextKeyUserClaims, claims)
		handler := RequireAccess(AccessRule{
			AllScopes: []string{"focus.read", "focus.write"},
		})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req = req.WithContext(ctx)
		rr := &testResponseWriter{}
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.statusCode)
	})
}

// testResponseWriter тестовая реализация http.ResponseWriter
type testResponseWriter struct {
	statusCode int
	header     http.Header
	body       []byte
}

func (t *testResponseWriter) Header() http.Header {
	if t.header == nil {
		t.header = make(http.Header)
	}
	return t.header
}

func (t *testResponseWriter) Write(data []byte) (int, error) {
	t.body = data
	return len(data), nil
}

func (t *testResponseWriter) WriteHeader(statusCode int) {
	t.statusCode = statusCode
}
