package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/orlandoburli/feature-bacon/internal/api/problem"
	"github.com/orlandoburli/feature-bacon/internal/auth"
)

const (
	schemeAPIKey = "ApiKey"
	schemeBearer = "Bearer"
)

// AuthDeps holds dependencies needed by the auth middleware.
type AuthDeps struct {
	KeyStore     auth.KeyFinder
	JWTValidator *auth.JWTValidator
	JWTEnabled   bool
}

// Auth returns middleware that authenticates requests via API key or JWT.
// When disabled is true the middleware is a no-op (sidecar bypass).
func Auth(deps AuthDeps, disabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if disabled {
				next.ServeHTTP(w, r)
				return
			}

			header := r.Header.Get("Authorization")
			if header == "" {
				problem.Write(w, problem.Unauthorized("missing authorization header", r.URL.Path))
				return
			}

			scheme, token, ok := parseAuthHeader(header)
			if !ok {
				problem.Write(w, problem.Unauthorized("malformed authorization header", r.URL.Path))
				return
			}

			switch scheme {
			case schemeAPIKey:
				handleAPIKey(w, r, next, deps.KeyStore, token)
			case schemeBearer:
				handleBearer(w, r, next, deps, token)
			default:
				problem.Write(w, problem.Unauthorized("unsupported authorization scheme", r.URL.Path))
			}
		})
	}
}

func handleAPIKey(w http.ResponseWriter, r *http.Request, next http.Handler, store auth.KeyFinder, rawKey string) {
	key, err := auth.AuthenticateAPIKey(store, rawKey)
	if err != nil {
		problem.Write(w, problem.Unauthorized(err.Error(), r.URL.Path))
		return
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, tenantIDKey, key.TenantID)
	ctx = context.WithValue(ctx, scopeKey, key.Scope)
	next.ServeHTTP(w, r.WithContext(ctx))
}

func handleBearer(w http.ResponseWriter, r *http.Request, next http.Handler, deps AuthDeps, token string) {
	if !deps.JWTEnabled || deps.JWTValidator == nil {
		problem.Write(w, problem.Unauthorized("JWT authentication is not configured", r.URL.Path))
		return
	}

	result, err := deps.JWTValidator.Validate(token)
	if err != nil {
		problem.Write(w, problem.Unauthorized(err.Error(), r.URL.Path))
		return
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, tenantIDKey, result.TenantID)
	if result.Scope.Valid() {
		ctx = context.WithValue(ctx, scopeKey, result.Scope)
	}
	next.ServeHTTP(w, r.WithContext(ctx))
}

func parseAuthHeader(header string) (scheme, token string, ok bool) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
