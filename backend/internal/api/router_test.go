package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/api/handlers"
	"github.com/orlandoburli/feature-bacon/internal/auth"
	"github.com/orlandoburli/feature-bacon/internal/engine"
)

const (
	flagKeyTestFlag    = "test-flag"
	fmtStatusWant      = "status = %d, want %d"
	headerRequestID    = "X-Request-Id"
	requestIDCustom42  = "custom-id-42"
	pathEvalEndpoint   = "/api/v1/evaluate"
	headerContentType  = "Content-Type"
	contentTypeJSON    = "application/json"
	pathAPIFlags       = "/api/v1/flags"
	pathAPIExperiments = "/api/v1/experiments"
	expKeyOnboarding   = "onboarding"
)

type stubFlagManager struct{}

func (s *stubFlagManager) GetFlag(_ context.Context, _, flagKey string) (*pb.FlagDefinition, error) {
	if flagKey == flagKeyTestFlag {
		return &pb.FlagDefinition{Key: flagKeyTestFlag, Type: "boolean", Semantics: "flag", Enabled: true}, nil
	}
	return nil, nil
}

func (s *stubFlagManager) ListFlags(_ context.Context, _ string, _, _ int) ([]*pb.FlagDefinition, int, error) {
	return []*pb.FlagDefinition{{Key: flagKeyTestFlag}}, 1, nil
}

func (s *stubFlagManager) CreateFlag(_ context.Context, _ string, f *pb.FlagDefinition) (*pb.FlagDefinition, error) {
	return f, nil
}

func (s *stubFlagManager) UpdateFlag(_ context.Context, _ string, f *pb.FlagDefinition) (*pb.FlagDefinition, error) {
	return f, nil
}

func (s *stubFlagManager) DeleteFlag(_ context.Context, _, _ string) error { return nil }

var _ handlers.FlagManager = (*stubFlagManager)(nil)

type stubExperimentManager struct{}

func (s *stubExperimentManager) GetExperiment(_ context.Context, _, key string) (*pb.Experiment, error) {
	if key == expKeyOnboarding {
		return &pb.Experiment{
			Key: expKeyOnboarding, Name: "Onboarding", Status: "draft",
			StickyAssignment: true,
		}, nil
	}
	return nil, nil
}

func (s *stubExperimentManager) ListExperiments(_ context.Context, _ string, _, _ int) ([]*pb.Experiment, int, error) {
	return []*pb.Experiment{{Key: expKeyOnboarding, Status: "draft"}}, 1, nil
}

func (s *stubExperimentManager) CreateExperiment(_ context.Context, _ string, e *pb.Experiment) (*pb.Experiment, error) {
	e.Status = "draft"
	return e, nil
}

func (s *stubExperimentManager) UpdateExperiment(_ context.Context, _ string, e *pb.Experiment) (*pb.Experiment, error) {
	return e, nil
}

var _ handlers.ExperimentManager = (*stubExperimentManager)(nil)

func testRouter(eng *engine.Engine) http.Handler {
	return NewRouter(RouterConfig{
		Engine:            eng,
		AuthDisabled:      true,
		KeyStore:          auth.NewMemKeyStore(),
		FlagManager:       &stubFlagManager{},
		ExperimentManager: &stubExperimentManager{},
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
	eng := engine.New(&stubStore{}, nil)
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
	eng := engine.New(&stubStore{}, nil)
	router := testRouter(eng)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusOK)
	}
}

func TestNewRouter_Evaluate(t *testing.T) {
	eng := engine.New(&stubStore{}, nil)
	router := testRouter(eng)

	body := `{"flagKey":"` + flagKeyTestFlag + `","context":{"subjectId":"user-1"}}`
	req := httptest.NewRequest(http.MethodPost, pathEvalEndpoint, bytes.NewBufferString(body))
	req.Header.Set(headerContentType, contentTypeJSON)
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
	eng := engine.New(&stubStore{}, nil)
	router := testRouter(eng)

	body := `{"flagKeys":["` + flagKeyTestFlag + `"],"context":{"subjectId":"user-1"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/evaluate/batch", bytes.NewBufferString(body))
	req.Header.Set(headerContentType, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusOK)
	}
}

func TestNewRouter_CorrelationID_Echo(t *testing.T) {
	eng := engine.New(&stubStore{}, nil)
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
	eng := engine.New(&stubStore{}, nil)
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
	eng := engine.New(&stubStore{}, nil)
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
	eng := engine.New(&stubStore{}, nil)
	router := testRouter(eng)

	req := httptest.NewRequest(http.MethodGet, pathEvalEndpoint, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Error("expected non-200 for GET on POST-only route")
	}
}

func TestNewRouter_ListFlags(t *testing.T) {
	eng := engine.New(&stubStore{}, nil)
	router := testRouter(eng)

	req := httptest.NewRequest(http.MethodGet, pathAPIFlags, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusOK)
	}
}

func TestNewRouter_GetFlag(t *testing.T) {
	eng := engine.New(&stubStore{}, nil)
	router := testRouter(eng)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/flags/"+flagKeyTestFlag, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusOK)
	}
}

func TestNewRouter_CreateFlag(t *testing.T) {
	eng := engine.New(&stubStore{}, nil)
	router := testRouter(eng)

	body := `{"key":"new-flag","type":"boolean","semantics":"flag","enabled":true}`
	req := httptest.NewRequest(http.MethodPost, pathAPIFlags, bytes.NewBufferString(body))
	req.Header.Set(headerContentType, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusCreated)
	}
}

func TestNewRouter_DeleteFlag(t *testing.T) {
	eng := engine.New(&stubStore{}, nil)
	router := testRouter(eng)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/flags/"+flagKeyTestFlag, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusNoContent)
	}
}

func TestNewRouter_ListExperiments(t *testing.T) {
	eng := engine.New(&stubStore{}, nil)
	router := testRouter(eng)

	req := httptest.NewRequest(http.MethodGet, pathAPIExperiments, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusOK)
	}
}

func TestNewRouter_GetExperiment(t *testing.T) {
	eng := engine.New(&stubStore{}, nil)
	router := testRouter(eng)

	req := httptest.NewRequest(http.MethodGet, pathAPIExperiments+"/"+expKeyOnboarding, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusOK)
	}
}

func TestNewRouter_CreateExperiment(t *testing.T) {
	eng := engine.New(&stubStore{}, nil)
	router := testRouter(eng)

	body := `{"key":"new-exp","name":"New Experiment"}`
	req := httptest.NewRequest(http.MethodPost, pathAPIExperiments, bytes.NewBufferString(body))
	req.Header.Set(headerContentType, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusCreated)
	}
}

func TestNewRouter_StartExperiment(t *testing.T) {
	eng := engine.New(&stubStore{}, nil)
	router := testRouter(eng)

	req := httptest.NewRequest(http.MethodPost, pathAPIExperiments+"/"+expKeyOnboarding+"/start", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusOK)
	}
}

func TestNewRouter_ReadOnlyMode_Experiments(t *testing.T) {
	eng := engine.New(&stubStore{}, nil)
	router := NewRouter(RouterConfig{
		Engine:            eng,
		AuthDisabled:      true,
		KeyStore:          auth.NewMemKeyStore(),
		FlagManager:       &stubFlagManager{},
		ExperimentManager: &stubExperimentManager{},
		ReadOnly:          true,
	})

	body := `{"key":"new-exp","name":"New Experiment"}`
	req := httptest.NewRequest(http.MethodPost, pathAPIExperiments, bytes.NewBufferString(body))
	req.Header.Set(headerContentType, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusConflict)
	}
}

func TestNewRouter_ReadOnlyMode(t *testing.T) {
	eng := engine.New(&stubStore{}, nil)
	router := NewRouter(RouterConfig{
		Engine:       eng,
		AuthDisabled: true,
		KeyStore:     auth.NewMemKeyStore(),
		FlagManager:  &stubFlagManager{},
		ReadOnly:     true,
	})

	body := `{"key":"new-flag","type":"boolean","semantics":"flag","enabled":true}`
	req := httptest.NewRequest(http.MethodPost, pathAPIFlags, bytes.NewBufferString(body))
	req.Header.Set(headerContentType, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf(fmtStatusWant, w.Code, http.StatusConflict)
	}
}
