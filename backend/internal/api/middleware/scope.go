package middleware

import (
	"net/http"

	"github.com/orlandoburli/feature-bacon/internal/api/problem"
	"github.com/orlandoburli/feature-bacon/internal/auth"
)

// RequireScope returns middleware that enforces the given scope on the request.
// If auth is disabled, scope checks are skipped.
func RequireScope(required auth.Scope, authDisabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authDisabled {
				next.ServeHTTP(w, r)
				return
			}

			scope := ScopeFromRequest(r)

			// management scope can access everything
			if scope == auth.ScopeManagement {
				next.ServeHTTP(w, r)
				return
			}

			if scope != required {
				problem.Write(w, problem.Forbidden("insufficient scope for this endpoint", r.URL.Path))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
