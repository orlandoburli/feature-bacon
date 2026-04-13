package api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/orlandoburli/feature-bacon/internal/api/handlers"
	"github.com/orlandoburli/feature-bacon/internal/api/middleware"
	"github.com/orlandoburli/feature-bacon/internal/auth"
	"github.com/orlandoburli/feature-bacon/internal/engine"
)

// RouterConfig holds dependencies for building the HTTP router.
type RouterConfig struct {
	Engine            *engine.Engine
	AuthDisabled      bool
	KeyStore          auth.KeyFinder
	JWTValidator      *auth.JWTValidator
	JWTEnabled        bool
	ReadOnly          bool
	FlagManager       handlers.FlagManager
	ExperimentManager handlers.ExperimentManager
	APIKeyManager     handlers.APIKeyManager
	HealthCheckers    []handlers.HealthChecker
}

func NewRouter(cfg RouterConfig) http.Handler {
	authMW := middleware.Auth(middleware.AuthDeps{
		KeyStore:     cfg.KeyStore,
		JWTValidator: cfg.JWTValidator,
		JWTEnabled:   cfg.JWTEnabled,
	}, cfg.AuthDisabled)
	tenantMW := middleware.TenantResolver(cfg.AuthDisabled)
	evalChain := applyScope(auth.ScopeEvaluation, cfg.AuthDisabled)

	mgmtChain := applyScope(auth.ScopeManagement, cfg.AuthDisabled)
	readOnlyMW := middleware.ReadOnly(cfg.ReadOnly)

	apiMux := http.NewServeMux()
	apiMux.Handle("POST /api/v1/evaluate", evalChain(handlers.HandleEvaluate(cfg.Engine)))
	apiMux.Handle("POST /api/v1/evaluate/batch", evalChain(handlers.HandleEvaluateBatch(cfg.Engine)))

	if cfg.FlagManager != nil {
		mgmtMux := http.NewServeMux()
		mgmtMux.Handle("GET /api/v1/flags", mgmtChain(handlers.HandleListFlags(cfg.FlagManager)))
		mgmtMux.Handle("GET /api/v1/flags/{flagKey}", mgmtChain(handlers.HandleGetFlag(cfg.FlagManager)))
		mgmtMux.Handle("POST /api/v1/flags", mgmtChain(handlers.HandleCreateFlag(cfg.FlagManager)))
		mgmtMux.Handle("PUT /api/v1/flags/{flagKey}", mgmtChain(handlers.HandleUpdateFlag(cfg.FlagManager)))
		mgmtMux.Handle("DELETE /api/v1/flags/{flagKey}", mgmtChain(handlers.HandleDeleteFlag(cfg.FlagManager)))

		apiMux.Handle("/api/v1/flags", readOnlyMW(mgmtMux))
		apiMux.Handle("/api/v1/flags/", readOnlyMW(mgmtMux))
	}

	if cfg.ExperimentManager != nil {
		expMux := http.NewServeMux()
		expMux.Handle("GET /api/v1/experiments", mgmtChain(handlers.HandleListExperiments(cfg.ExperimentManager)))
		expMux.Handle("GET /api/v1/experiments/{experimentKey}", mgmtChain(handlers.HandleGetExperiment(cfg.ExperimentManager)))
		expMux.Handle("POST /api/v1/experiments", mgmtChain(handlers.HandleCreateExperiment(cfg.ExperimentManager)))
		expMux.Handle("PUT /api/v1/experiments/{experimentKey}", mgmtChain(handlers.HandleUpdateExperiment(cfg.ExperimentManager)))
		expMux.Handle("POST /api/v1/experiments/{experimentKey}/start", mgmtChain(handlers.HandleStartExperiment(cfg.ExperimentManager)))
		expMux.Handle("POST /api/v1/experiments/{experimentKey}/pause", mgmtChain(handlers.HandlePauseExperiment(cfg.ExperimentManager)))
		expMux.Handle("POST /api/v1/experiments/{experimentKey}/complete", mgmtChain(handlers.HandleCompleteExperiment(cfg.ExperimentManager)))

		apiMux.Handle("/api/v1/experiments", readOnlyMW(expMux))
		apiMux.Handle("/api/v1/experiments/", readOnlyMW(expMux))
	}

	if cfg.APIKeyManager != nil {
		akMux := http.NewServeMux()
		akMux.Handle("GET /api/v1/api-keys", mgmtChain(handlers.HandleListAPIKeys(cfg.APIKeyManager)))
		akMux.Handle("POST /api/v1/api-keys", mgmtChain(handlers.HandleCreateAPIKey(cfg.APIKeyManager)))
		akMux.Handle("DELETE /api/v1/api-keys/{keyId}", mgmtChain(handlers.HandleRevokeAPIKey(cfg.APIKeyManager)))

		apiMux.Handle("/api/v1/api-keys", readOnlyMW(akMux))
		apiMux.Handle("/api/v1/api-keys/", readOnlyMW(akMux))
	}

	var protectedAPI http.Handler = apiMux
	protectedAPI = tenantMW(protectedAPI)
	protectedAPI = authMW(protectedAPI)

	root := http.NewServeMux()
	root.HandleFunc("GET /healthz", handlers.HandleHealthz())
	root.HandleFunc("GET /readyz", handlers.HandleReadyz(cfg.HealthCheckers...))
	root.Handle("GET /metrics", promhttp.Handler())
	root.Handle("/api/", protectedAPI)

	var h http.Handler = root
	h = middleware.RequestLogger(h)
	h = middleware.Metrics(h)
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
