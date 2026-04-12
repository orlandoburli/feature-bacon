package middleware

import (
	"context"
	"net/http"

	"github.com/orlandoburli/feature-bacon/internal/api/handlers"
	"github.com/orlandoburli/feature-bacon/internal/api/problem"
)

const defaultTenantID = "_default"

// TenantResolver propagates the tenant ID from the auth middleware into
// the handlers.TenantIDKey context value used by handlers.
// In sidecar mode (authDisabled=true), the default tenant is always used.
func TenantResolver(authDisabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tenantID string
			if authDisabled {
				tenantID = defaultTenantID
			} else {
				tenantID = TenantIDFromRequest(r)
			}

			if tenantID == "" {
				problem.Write(w, problem.Unauthorized("unable to resolve tenant", r.URL.Path))
				return
			}

			ctx := context.WithValue(r.Context(), handlers.TenantIDKey, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
