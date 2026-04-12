package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orlandoburli/feature-bacon/internal/engine"
)

type stubStore struct{}

func (s *stubStore) GetFlag(_, flagKey string) (*engine.FlagDefinition, error) {
	if flagKey == "test-flag" {
		return &engine.FlagDefinition{
			Key:     "test-flag",
			Type:    engine.FlagTypeBoolean,
			Enabled: true,
			DefaultResult: engine.EvalResult{
				Enabled: true,
				Variant: "on",
			},
		}, nil
	}
	return nil, nil
}

func (s *stubStore) ListFlagKeys(_ string) ([]string, error) {
	return []string{"test-flag"}, nil
}

func TestNewRouter_Healthz(t *testing.T) {
	eng := engine.New(&stubStore{})
	router := NewRouter(eng)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if w.Header().Get("X-Bacon-Version") == "" {
		t.Error("expected X-Bacon-Version header")
	}
	if w.Header().Get("X-Request-Id") == "" {
		t.Error("expected X-Request-Id header")
	}
}

func TestNewRouter_Readyz(t *testing.T) {
	eng := engine.New(&stubStore{})
	router := NewRouter(eng)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestNewRouter_Evaluate(t *testing.T) {
	eng := engine.New(&stubStore{})
	router := NewRouter(eng)

	body := `{"flagKey":"test-flag","context":{"subjectId":"user-1"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/evaluate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var result engine.EvaluationResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.FlagKey != "test-flag" {
		t.Errorf("flagKey = %q, want %q", result.FlagKey, "test-flag")
	}
}

func TestNewRouter_EvaluateBatch(t *testing.T) {
	eng := engine.New(&stubStore{})
	router := NewRouter(eng)

	body := `{"flagKeys":["test-flag"],"context":{"subjectId":"user-1"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/evaluate/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestNewRouter_CorrelationID_Echo(t *testing.T) {
	eng := engine.New(&stubStore{})
	router := NewRouter(eng)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Request-Id", "custom-id-42")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Request-Id") != "custom-id-42" {
		t.Errorf("X-Request-Id = %q, want %q", w.Header().Get("X-Request-Id"), "custom-id-42")
	}
}

func TestNewRouter_MethodNotAllowed(t *testing.T) {
	eng := engine.New(&stubStore{})
	router := NewRouter(eng)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/evaluate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Error("expected non-200 for GET on POST-only route")
	}
}
