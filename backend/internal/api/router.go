package api

import (
	"net/http"

	"github.com/orlandoburli/feature-bacon/internal/api/handlers"
	"github.com/orlandoburli/feature-bacon/internal/api/middleware"
	"github.com/orlandoburli/feature-bacon/internal/auth"
	"github.com/orlandoburli/feature-bacon/internal/engine"
)

// RouterConfig holds dependencies for building the HTTP router.
type RouterConfig struct {
	Engine       *engine.Engine
	AuthDisabled bool
	KeyStore     auth.KeyFinder
	JWTValidator *auth.JWTValidator
	JWTEnabled   bool
}

func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	evalChain := applyScope(auth.ScopeEvaluation, cfg.AuthDisabled)
	mux.Handle("POST /api/v1/evaluate", evalChain(handlers.HandleEvaluate(cfg.Engine)))
	mux.Handle("POST /api/v1/evaluate/batch", evalChain(handlers.HandleEvaluateBatch(cfg.Engine)))
	mux.HandleFunc("GET /healthz", handlers.HandleHealthz())
	mux.HandleFunc("GET /readyz", handlers.HandleReadyz())

	authMW := middleware.Auth(middleware.AuthDeps{
		KeyStore:     cfg.KeyStore,
		JWTValidator: cfg.JWTValidator,
		JWTEnabled:   cfg.JWTEnabled,
	}, cfg.AuthDisabled)
	tenantMW := middleware.TenantResolver(cfg.AuthDisabled)

	var h http.Handler = mux
	h = tenantMW(h)
	h = authMW(h)
	h = middleware.Correlation(h)
	h = middleware.VersionHeader(h)
	return h
}

func applyScope(scope auth.Scope, authDisabled bool) func(http.HandlerFunc) http.Handler {
	mw := middleware.RequireScope(scope, authDisabled)
	return func(hf http.HandlerFunc) http.Handler {
		return mw(hf)
	}
}
