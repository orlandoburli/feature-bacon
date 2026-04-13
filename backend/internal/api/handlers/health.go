package handlers

import (
	"context"
	"encoding/json"
	"net/http"
)

type ModuleHealth struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
	Message   string `json:"message,omitempty"`
}

type HealthChecker interface {
	CheckHealth(ctx context.Context) (name string, health ModuleHealth)
}

func HandleHealthz() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

func HandleReadyz(checkers ...HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		overall := "ready"
		modules := make(map[string]ModuleHealth)

		for _, c := range checkers {
			name, health := c.CheckHealth(r.Context())
			modules[name] = health
			if health.Status != "ok" {
				overall = "not_ready"
			}
		}

		status := http.StatusOK
		if overall != "ready" {
			status = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  overall,
			"modules": modules,
		})
	}
}
