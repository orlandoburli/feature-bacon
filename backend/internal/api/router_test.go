package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orlandoburli/feature-bacon/internal/auth"
	"github.com/orlandoburli/feature-bacon/internal/engine"
)

const (
	flagKeyTestFlag   = "test-flag"
	fmtStatusWant     = "status = %d, want %d"
	headerRequestID   = "X-Request-Id"
	requestIDCustom42 = "custom-id-42"
	pathEvalEndpoint  = "/api/v1/evaluate"
)

func testRouter(eng *engine.Engine) http.Handler {
	return NewRouter(RouterConfig{
		Engine:       eng,
		AuthDisabled: true,
		KeyStore:     auth.NewMemKeyStore(),
	})
}

type stubStore struct{}

func (s *stubStore) GetFlag(_, flagKey string) (*engine.FlagDefinition, error) {
	if flagKey == flagKeyTestFlag {
		return &engine.FlagDefinition{
			Key:     flagKeyTestFlag,
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
	return []string{flagKeyTestFlag}, nil
}

func TestNewRouter_Healthz(t *testing.T) {
	eng := engine.New(&stubStore{})
	router := testRouter(eng)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusOK)
	}

	if w.Header().Get("X-Bacon-Version") == "" {
		t.Error("expected X-Bacon-Version header")
	}
	if w.Header().Get(headerRequestID) == "" {
		t.Error("expected " + headerRequestID + " header")
	}
}

func TestNewRouter_Readyz(t *testing.T) {
	eng := engine.New(&stubStore{})
	router := testRouter(eng)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusOK)
	}
}

func TestNewRouter_Evaluate(t *testing.T) {
	eng := engine.New(&stubStore{})
	router := testRouter(eng)

	body := `{"flagKey":"` + flagKeyTestFlag + `","context":{"subjectId":"user-1"}}`
	req := httptest.NewRequest(http.MethodPost, pathEvalEndpoint, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusOK)
	}

	var result engine.EvaluationResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.FlagKey != flagKeyTestFlag {
		t.Errorf("flagKey = %q, want %q", result.FlagKey, flagKeyTestFlag)
	}
}

func TestNewRouter_EvaluateBatch(t *testing.T) {
	eng := engine.New(&stubStore{})
	router := testRouter(eng)

	body := `{"flagKeys":["` + flagKeyTestFlag + `"],"context":{"subjectId":"user-1"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/evaluate/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusOK)
	}
}

func TestNewRouter_CorrelationID_Echo(t *testing.T) {
	eng := engine.New(&stubStore{})
	router := testRouter(eng)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(headerRequestID, requestIDCustom42)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get(headerRequestID) != requestIDCustom42 {
		t.Errorf("%s = %q, want %q", headerRequestID, w.Header().Get(headerRequestID), requestIDCustom42)
	}
}

func TestNewRouter_AuthEnabled_Unauthorized(t *testing.T) {
	eng := engine.New(&stubStore{})
	router := NewRouter(RouterConfig{
		Engine:       eng,
		AuthDisabled: false,
		KeyStore:     auth.NewMemKeyStore(),
	})

	body := `{"flagKey":"test-flag","context":{"subjectId":"u1"}}`
	req := httptest.NewRequest(http.MethodPost, pathEvalEndpoint, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusUnauthorized)
	}
}

func TestNewRouter_AuthEnabled_ValidKey(t *testing.T) {
	eng := engine.New(&stubStore{})
	store := auth.NewMemKeyStore()
	rawKey := "ba_eval_routertest"
	store.Add(&auth.APIKey{
		ID:       "rk1",
		TenantID: "_default",
		KeyHash:  auth.HashKey(rawKey),
		Scope:    auth.ScopeEvaluation,
	})
	router := NewRouter(RouterConfig{
		Engine:       eng,
		AuthDisabled: false,
		KeyStore:     store,
	})

	body := `{"flagKey":"` + flagKeyTestFlag + `","context":{"subjectId":"u1"}}`
	req := httptest.NewRequest(http.MethodPost, pathEvalEndpoint, bytes.NewBufferString(body))
	req.Header.Set("Authorization", "ApiKey "+rawKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusOK)
	}
}

func TestNewRouter_MethodNotAllowed(t *testing.T) {
	eng := engine.New(&stubStore{})
	router := testRouter(eng)

	req := httptest.NewRequest(http.MethodGet, pathEvalEndpoint, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Error("expected non-200 for GET on POST-only route")
	}
}
