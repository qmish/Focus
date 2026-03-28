package auth

import (
	"net/http"
	"strings"
)

// ABACRequest describes action/resource/context for policy evaluation.
type ABACRequest struct {
	Action     string
	Resource   string
	Attributes map[string]string
}

// ABACPolicyEngine evaluates ABAC decisions.
type ABACPolicyEngine interface {
	Allow(claims *SessionClaims, req ABACRequest) bool
}

// DefaultABACEngine is a simple ABAC engine over roles/scopes/context.
type DefaultABACEngine struct{}

// NewDefaultABACEngine creates default ABAC policy engine.
func NewDefaultABACEngine() *DefaultABACEngine {
	return &DefaultABACEngine{}
}

// Allow returns true when request is allowed by ABAC policies.
func (e *DefaultABACEngine) Allow(claims *SessionClaims, req ABACRequest) bool {
	if claims == nil {
		return false
	}
	if hasRole(claims.Roles, "admin") {
		return true
	}

	requiredScopes := map[string][]string{
		"user.ban":       {"focus.admin.user.ban", "focus.admin"},
		"user.unban":     {"focus.admin.user.unban", "focus.admin"},
		"conference.end": {"focus.admin.conference.end", "focus.admin"},
	}

	candidates, ok := requiredScopes[strings.TrimSpace(req.Action)]
	if !ok {
		return false
	}
	userScopes := claims.AllScopes()
	for _, scope := range candidates {
		if hasScope(userScopes, scope) {
			return true
		}
	}
	return false
}

// RequireABAC checks access using ABAC engine and action/resource context.
func RequireABAC(engine ABACPolicyEngine, action string, resourceResolver func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserClaimsFromContext(r.Context())
			if claims == nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if engine == nil {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			resource := ""
			if resourceResolver != nil {
				resource = resourceResolver(r)
			}
			if !engine.Allow(claims, ABACRequest{
				Action:   action,
				Resource: resource,
				Attributes: map[string]string{
					"method": r.Method,
					"path":   r.URL.Path,
				},
			}) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// HasRole checks whether the session claims contain the given role.
// For the "admin" role it also accepts the "focus.admin" scope.
func HasRole(claims *SessionClaims, role string) bool {
	if claims == nil {
		return false
	}
	for _, r := range claims.Roles {
		if r == role {
			return true
		}
	}
	if role == "admin" {
		for _, scope := range claims.AllScopes() {
			if scope == "focus.admin" {
				return true
			}
		}
	}
	return false
}

func hasRole(roles []string, required string) bool {
	for _, role := range roles {
		if role == required {
			return true
		}
	}
	return false
}

func hasScope(scopes []string, required string) bool {
	for _, scope := range scopes {
		if scope == required {
			return true
		}
	}
	return false
}
