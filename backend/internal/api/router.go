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
	authMW := middleware.Auth(middleware.AuthDeps{
		KeyStore:     cfg.KeyStore,
		JWTValidator: cfg.JWTValidator,
		JWTEnabled:   cfg.JWTEnabled,
	}, cfg.AuthDisabled)
	tenantMW := middleware.TenantResolver(cfg.AuthDisabled)
	evalChain := applyScope(auth.ScopeEvaluation, cfg.AuthDisabled)

	apiMux := http.NewServeMux()
	apiMux.Handle("POST /api/v1/evaluate", evalChain(handlers.HandleEvaluate(cfg.Engine)))
	apiMux.Handle("POST /api/v1/evaluate/batch", evalChain(handlers.HandleEvaluateBatch(cfg.Engine)))

	var protectedAPI http.Handler = apiMux
	protectedAPI = tenantMW(protectedAPI)
	protectedAPI = authMW(protectedAPI)

	root := http.NewServeMux()
	root.HandleFunc("GET /healthz", handlers.HandleHealthz())
	root.HandleFunc("GET /readyz", handlers.HandleReadyz())
	root.Handle("/api/", protectedAPI)

	var h http.Handler = root
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
