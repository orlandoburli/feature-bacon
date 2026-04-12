package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orlandoburli/feature-bacon/internal/engine"
)

const (
	flagKeyDarkMode   = "dark-mode"
	pathEvaluate      = "/api/v1/evaluate"
	pathEvaluateBatch = "/api/v1/evaluate/batch"
	tenantMyTenant    = "my-tenant"
	fmtStatusWant     = "status = %d, want %d"
	fmtDecode         = "decode: %v"
)

type stubStore struct {
	flags map[string]*engine.FlagDefinition
}

func (s *stubStore) GetFlag(tenantID, flagKey string) (*engine.FlagDefinition, error) {
	return s.flags[tenantID+"/"+flagKey], nil
}

func (s *stubStore) ListFlagKeys(tenantID string) ([]string, error) {
	var keys []string
	prefix := tenantID + "/"
	for k := range s.flags {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			keys = append(keys, k[len(prefix):])
		}
	}
	return keys, nil
}

func newTestEngine() *engine.Engine {
	store := &stubStore{
		flags: map[string]*engine.FlagDefinition{
			"_default/" + flagKeyDarkMode: {
				Key:     flagKeyDarkMode,
				Type:    engine.FlagTypeBoolean,
				Enabled: true,
				DefaultResult: engine.EvalResult{
					Enabled: true,
					Variant: "on",
				},
			},
			"tenant-a/" + flagKeyDarkMode: {
				Key:     flagKeyDarkMode,
				Type:    engine.FlagTypeBoolean,
				Enabled: true,
				DefaultResult: engine.EvalResult{
					Enabled: true,
					Variant: "control",
				},
			},
		},
	}
	return engine.New(store, nil)
}

func TestHandleEvaluate_Success(t *testing.T) {
	eng := newTestEngine()
	body := `{"flagKey":"` + flagKeyDarkMode + `","context":{"subjectId":"user-1","environment":"prod"}}`

	req := httptest.NewRequest(http.MethodPost, pathEvaluate, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandleEvaluate(eng).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusOK)
	}

	var result engine.EvaluationResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf(fmtDecode, err)
	}

	if result.FlagKey != flagKeyDarkMode {
		t.Errorf("flagKey = %q, want %q", result.FlagKey, flagKeyDarkMode)
	}
	if !result.Enabled {
		t.Error("expected enabled=true")
	}
	if result.Variant != "on" {
		t.Errorf("variant = %q, want %q", result.Variant, "on")
	}
}

func TestHandleEvaluate_EmptyFlagKey(t *testing.T) {
	eng := newTestEngine()
	body := `{"flagKey":"","context":{}}`

	req := httptest.NewRequest(http.MethodPost, pathEvaluate, bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	HandleEvaluate(eng).ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusUnprocessableEntity)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
}

func TestHandleEvaluate_InvalidJSON(t *testing.T) {
	eng := newTestEngine()

	req := httptest.NewRequest(http.MethodPost, pathEvaluate, bytes.NewBufferString(`{bad`))
	w := httptest.NewRecorder()

	HandleEvaluate(eng).ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleEvaluate_WithTenantContext(t *testing.T) {
	eng := newTestEngine()
	body := `{"flagKey":"` + flagKeyDarkMode + `","context":{"subjectId":"user-1"}}`

	req := httptest.NewRequest(http.MethodPost, pathEvaluate, bytes.NewBufferString(body))
	ctx := context.WithValue(req.Context(), TenantIDKey, "tenant-a")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	HandleEvaluate(eng).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusOK)
	}

	var result engine.EvaluationResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf(fmtDecode, err)
	}

	if result.Variant != "control" {
		t.Errorf("variant = %q, want %q (tenant-a flag)", result.Variant, "control")
	}
}

func TestHandleEvaluate_FlagNotFound(t *testing.T) {
	eng := newTestEngine()
	body := `{"flagKey":"nonexistent","context":{"subjectId":"user-1"}}`

	req := httptest.NewRequest(http.MethodPost, pathEvaluate, bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	HandleEvaluate(eng).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusOK)
	}

	var result engine.EvaluationResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf(fmtDecode, err)
	}

	if result.Reason != engine.ReasonNotFound {
		t.Errorf("reason = %q, want %q", result.Reason, engine.ReasonNotFound)
	}
}

func TestHandleEvaluateBatch_Success(t *testing.T) {
	eng := newTestEngine()
	body := `{"flagKeys":["` + flagKeyDarkMode + `","nonexistent"],"context":{"subjectId":"user-1"}}`

	req := httptest.NewRequest(http.MethodPost, pathEvaluateBatch, bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	HandleEvaluateBatch(eng).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusOK)
	}

	var resp struct {
		Results []engine.EvaluationResult `json:"results"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}

	if len(resp.Results) != 2 {
		t.Fatalf("results len = %d, want 2", len(resp.Results))
	}
	if resp.Results[0].FlagKey != flagKeyDarkMode {
		t.Errorf("results[0].flagKey = %q, want %q", resp.Results[0].FlagKey, flagKeyDarkMode)
	}
	if resp.Results[1].Reason != engine.ReasonNotFound {
		t.Errorf("results[1].reason = %q, want %q", resp.Results[1].Reason, engine.ReasonNotFound)
	}
}

func TestHandleEvaluateBatch_EmptyFlagKeys(t *testing.T) {
	eng := newTestEngine()
	body := `{"flagKeys":[],"context":{}}`

	req := httptest.NewRequest(http.MethodPost, pathEvaluateBatch, bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	HandleEvaluateBatch(eng).ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleEvaluateBatch_MissingFlagKeys(t *testing.T) {
	eng := newTestEngine()
	body := `{"context":{}}`

	req := httptest.NewRequest(http.MethodPost, pathEvaluateBatch, bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	HandleEvaluateBatch(eng).ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleEvaluateBatch_InvalidJSON(t *testing.T) {
	eng := newTestEngine()

	req := httptest.NewRequest(http.MethodPost, pathEvaluateBatch, bytes.NewBufferString(`not-json`))
	w := httptest.NewRecorder()

	HandleEvaluateBatch(eng).ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusUnprocessableEntity)
	}
}

func TestTenantFromContext_Default(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	got := tenantFromContext(req)
	if got != "_default" {
		t.Errorf("tenantFromContext = %q, want %q", got, "_default")
	}
}

func TestTenantFromContext_Set(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), TenantIDKey, tenantMyTenant)
	req = req.WithContext(ctx)
	got := tenantFromContext(req)
	if got != tenantMyTenant {
		t.Errorf("tenantFromContext = %q, want %q", got, tenantMyTenant)
	}
}
