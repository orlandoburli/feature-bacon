package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/orlandoburli/feature-bacon/internal/api/problem"
	"github.com/orlandoburli/feature-bacon/internal/engine"
)

type evaluateRequest struct {
	FlagKey string          `json:"flagKey"`
	Context evalContextBody `json:"context"`
}

type evaluateBatchRequest struct {
	FlagKeys []string        `json:"flagKeys"`
	Context  evalContextBody `json:"context"`
}

type evalContextBody struct {
	SubjectID   string         `json:"subjectId"`
	Environment string         `json:"environment"`
	Attributes  map[string]any `json:"attributes"`
}

func HandleEvaluate(eng *engine.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req evaluateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			problem.Write(w, problem.ValidationError("invalid JSON body", r.URL.Path))
			return
		}

		if req.FlagKey == "" {
			problem.Write(w, problem.ValidationError("flagKey is required", r.URL.Path))
			return
		}

		tenantID := tenantFromContext(r)

		ctx := engine.EvaluationContext{
			TenantID:    tenantID,
			SubjectID:   req.Context.SubjectID,
			Environment: req.Context.Environment,
			Attributes:  req.Context.Attributes,
		}

		result := eng.Evaluate(req.FlagKey, ctx)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}

func HandleEvaluateBatch(eng *engine.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req evaluateBatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			problem.Write(w, problem.ValidationError("invalid JSON body", r.URL.Path))
			return
		}

		if len(req.FlagKeys) == 0 {
			problem.Write(w, problem.ValidationError("flagKeys is required and must not be empty", r.URL.Path))
			return
		}

		tenantID := tenantFromContext(r)

		ctx := engine.EvaluationContext{
			TenantID:    tenantID,
			SubjectID:   req.Context.SubjectID,
			Environment: req.Context.Environment,
			Attributes:  req.Context.Attributes,
		}

		results := eng.EvaluateBatch(req.FlagKeys, ctx)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
	}
}

func tenantFromContext(r *http.Request) string {
	if id, ok := r.Context().Value(TenantIDKey).(string); ok && id != "" {
		return id
	}
	return "_default"
}
