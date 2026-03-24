package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/auth"
	"github.com/stretchr/testify/assert"
)

func TestHasRole(t *testing.T) {
	claims := &auth.SessionClaims{
		Roles: []string{"user", "moderator"},
	}

	assert.True(t, hasRole(claims, "user"))
	assert.True(t, hasRole(claims, "moderator"))
	assert.False(t, hasRole(claims, "admin"))
}

func TestRequireAdmin(t *testing.T) {
	t.Run("user with admin role", func(t *testing.T) {
		claims := &auth.SessionClaims{
			Roles: []string{"admin"},
		}

		ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
		rr := httptest.NewRecorder()

		handler := requireAdmin(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("user without admin role", func(t *testing.T) {
		claims := &auth.SessionClaims{
			Roles: []string{"user"},
		}

		ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
		rr := httptest.NewRecorder()

		handler := requireAdmin(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("no claims in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		handler := requireAdmin(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestAdminHandler(t *testing.T) {
	handler := NewAdminHandler(nil, nil)

	assert.NotNil(t, handler)
}

func TestAdminHandlerListUsersForbidden(t *testing.T) {
	handler := NewAdminHandler(nil, nil)

	// Создаём запрос без admin роли
	claims := &auth.SessionClaims{
		Roles: []string{"user"},
	}

	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/users", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ListUsers(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAdminHandlerGetUserInvalidID(t *testing.T) {
	handler := NewAdminHandler(nil, nil)

	claims := &auth.SessionClaims{
		Roles: []string{"admin"},
	}

	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/users/invalid-id", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetUser(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAdminHandlerGetUserForbidden(t *testing.T) {
	handler := NewAdminHandler(nil, nil)

	claims := &auth.SessionClaims{
		Roles: []string{"user"},
	}

	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/users/"+uuid.New().String(), nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetUser(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAdminHandlerListConferences(t *testing.T) {
	handler := NewAdminHandler(nil, nil)

	claims := &auth.SessionClaims{
		Roles: []string{"admin"},
	}

	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/conferences", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ListConferences(rr, req)

	// Должен вернуть пустой список
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"data":[]`)
}

func TestAdminHandlerEndConference(t *testing.T) {
	handler := NewAdminHandler(nil, nil)

	claims := &auth.SessionClaims{
		Roles: []string{"admin"},
	}

	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("POST", "/api/v1/admin/conferences/test-conference/end", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.EndConference(rr, req)

	// Должен вернуть успех (заглушка)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"ended":true`)
}

func TestAdminHandlerGetStats(t *testing.T) {
	handler := NewAdminHandler(nil, nil)

	claims := &auth.SessionClaims{
		Roles: []string{"admin"},
	}

	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/stats", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetStats(rr, req)

	// Должен вернуть 200 с пустой статистикой
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAdminHandlerGetStatsForbidden(t *testing.T) {
	handler := NewAdminHandler(nil, nil)

	claims := &auth.SessionClaims{
		Roles: []string{"user"},
	}

	ctx := context.WithValue(context.Background(), auth.ContextKeyUserClaims, claims)
	req := httptest.NewRequest("GET", "/api/v1/admin/stats", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetStats(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}
