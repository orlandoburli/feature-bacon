package api

import (
	"net/http"

	"github.com/orlandoburli/feature-bacon/internal/api/handlers"
	"github.com/orlandoburli/feature-bacon/internal/api/middleware"
	"github.com/orlandoburli/feature-bacon/internal/engine"
)

func NewRouter(eng *engine.Engine) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/evaluate", handlers.HandleEvaluate(eng))
	mux.HandleFunc("POST /api/v1/evaluate/batch", handlers.HandleEvaluateBatch(eng))
	mux.HandleFunc("GET /healthz", handlers.HandleHealthz())
	mux.HandleFunc("GET /readyz", handlers.HandleReadyz())

	var h http.Handler = mux
	h = middleware.Correlation(h)
	h = middleware.VersionHeader(h)
	return h
}
