package middleware

import (
	"net/http"

	"github.com/orlandoburli/feature-bacon/internal/auth"
)

type authContextKey string

const (
	tenantIDKey authContextKey = "tenantID"
	scopeKey    authContextKey = "scope"
)

// TenantIDFromRequest extracts the tenant ID set by the auth middleware.
func TenantIDFromRequest(r *http.Request) string {
	if id, ok := r.Context().Value(tenantIDKey).(string); ok {
		return id
	}
	return ""
}

// ScopeFromRequest extracts the auth scope set by the auth middleware.
func ScopeFromRequest(r *http.Request) auth.Scope {
	if s, ok := r.Context().Value(scopeKey).(auth.Scope); ok {
		return s
	}
	return ""
}
