package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultABACEngineAllow(t *testing.T) {
	engine := NewDefaultABACEngine()

	t.Run("admin role allowed", func(t *testing.T) {
		claims := &SessionClaims{Roles: []string{"admin"}}
		assert.True(t, engine.Allow(claims, ABACRequest{Action: "conference.end"}))
	})

	t.Run("scope based allow", func(t *testing.T) {
		claims := &SessionClaims{Scopes: []string{"focus.admin.user.ban"}}
		assert.True(t, engine.Allow(claims, ABACRequest{Action: "user.ban"}))
	})

	t.Run("deny unknown action", func(t *testing.T) {
		claims := &SessionClaims{Scopes: []string{"focus.admin"}}
		assert.False(t, engine.Allow(claims, ABACRequest{Action: "unknown.action"}))
	})

	t.Run("deny without role and scope", func(t *testing.T) {
		claims := &SessionClaims{Roles: []string{"user"}}
		assert.False(t, engine.Allow(claims, ABACRequest{Action: "conference.end"}))
	})
}

func TestRequireABAC(t *testing.T) {
	engine := NewDefaultABACEngine()
	handler := RequireABAC(engine, "conference.end", func(r *http.Request) string {
		return "conference:" + r.URL.Path
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/conferences/1/end", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("forbidden without permissions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/conferences/1/end", nil)
		req = req.WithContext(context.WithValue(req.Context(), ContextKeyUserClaims, &SessionClaims{
			Roles: []string{"moderator"},
		}))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("allowed with scoped claim", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/conferences/1/end", nil)
		req = req.WithContext(context.WithValue(req.Context(), ContextKeyUserClaims, &SessionClaims{
			Scopes: []string{"focus.admin.conference.end"},
		}))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
