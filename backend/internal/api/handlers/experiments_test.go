package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

const (
	pathExperiments             = "/api/v1/experiments"
	pathExperimentByKey         = "/api/v1/experiments/onboarding"
	experimentKeyOnboarding     = "onboarding"
	routeGetExperimentByKey     = "GET /experiments/{experimentKey}"
	routePutExperimentByKey     = "PUT /experiments/{experimentKey}"
	routePostExperimentStart    = "POST /experiments/{experimentKey}/start"
	routePostExperimentPause    = "POST /experiments/{experimentKey}/pause"
	routePostExperimentComplete = "POST /experiments/{experimentKey}/complete"
	pathExperimentsPrefix       = "/experiments/"
	msgExperimentNotFound       = "experiment not found"
	fmtCannotTransition         = "cannot transition from %q to %q"
	fmtVariantsLenWant2         = "variants len = %d, want 2"
	fmtAllocationLenWant2       = "allocation len = %d, want 2"
	fmtStatusFieldWant          = "status = %q, want %q"
	suffixStart                 = "/start"
	suffixPause                 = "/pause"
	suffixComplete              = "/complete"
)

type mockExperimentManager struct {
	getFunc    func(ctx context.Context, tenantID, experimentKey string) (*pb.Experiment, error)
	listFunc   func(ctx context.Context, tenantID string, page, perPage int) ([]*pb.Experiment, int, error)
	createFunc func(ctx context.Context, tenantID string, exp *pb.Experiment) (*pb.Experiment, error)
	updateFunc func(ctx context.Context, tenantID string, exp *pb.Experiment) (*pb.Experiment, error)
}

func (m *mockExperimentManager) GetExperiment(ctx context.Context, tenantID, experimentKey string) (*pb.Experiment, error) {
	return m.getFunc(ctx, tenantID, experimentKey)
}

func (m *mockExperimentManager) ListExperiments(ctx context.Context, tenantID string, page, perPage int) ([]*pb.Experiment, int, error) {
	return m.listFunc(ctx, tenantID, page, perPage)
}

func (m *mockExperimentManager) CreateExperiment(ctx context.Context, tenantID string, exp *pb.Experiment) (*pb.Experiment, error) {
	return m.createFunc(ctx, tenantID, exp)
}

func (m *mockExperimentManager) UpdateExperiment(ctx context.Context, tenantID string, exp *pb.Experiment) (*pb.Experiment, error) {
	return m.updateFunc(ctx, tenantID, exp)
}

func sampleExperiment(status string) *pb.Experiment {
	return &pb.Experiment{
		Key:              experimentKeyOnboarding,
		Name:             "Onboarding Flow",
		Status:           status,
		StickyAssignment: true,
		Variants: []*pb.Variant{
			{Key: "control", Description: "Original flow"},
			{Key: "variant-a", Description: "New flow"},
		},
		Allocation: []*pb.Allocation{
			{VariantKey: "control", Percentage: 50},
			{VariantKey: "variant-a", Percentage: 50},
		},
		CreatedAt: 1700000000,
		UpdatedAt: 1700000000,
	}
}

func TestHandleListExperiments(t *testing.T) {
	em := &mockExperimentManager{
		listFunc: func(_ context.Context, _ string, _, _ int) ([]*pb.Experiment, int, error) {
			return []*pb.Experiment{sampleExperiment(statusDraft)}, 1, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, pathExperiments+"?page=1&perPage=10", nil)
	w := httptest.NewRecorder()
	HandleListExperiments(em).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusOK)
	}

	var resp struct {
		Data       []experimentResponse `json:"data"`
		Pagination PaginationResponse   `json:"pagination"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].Key != experimentKeyOnboarding {
		t.Errorf(fmtKeyWant, resp.Data[0].Key, experimentKeyOnboarding)
	}
	if resp.Pagination.Total != 1 {
		t.Errorf("total = %d, want 1", resp.Pagination.Total)
	}
	if resp.Pagination.TotalPages != 1 {
		t.Errorf("totalPages = %d, want 1", resp.Pagination.TotalPages)
	}
}

func TestHandleListExperiments_Error(t *testing.T) {
	em := &mockExperimentManager{
		listFunc: func(_ context.Context, _ string, _, _ int) ([]*pb.Experiment, int, error) {
			return nil, 0, errStore
		},
	}

	req := httptest.NewRequest(http.MethodGet, pathExperiments, nil)
	w := httptest.NewRecorder()
	HandleListExperiments(em).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusInternalServerError)
	}
}

func TestHandleGetExperiment_Found(t *testing.T) {
	em := &mockExperimentManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.Experiment, error) {
			return sampleExperiment(statusDraft), nil
		},
	}

	mux := http.NewServeMux()
	mux.Handle(routeGetExperimentByKey, HandleGetExperiment(em))

	req := httptest.NewRequest(http.MethodGet, pathExperimentsPrefix+experimentKeyOnboarding, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusOK)
	}

	var resp experimentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if resp.Key != experimentKeyOnboarding {
		t.Errorf(fmtKeyWant, resp.Key, experimentKeyOnboarding)
	}
	if len(resp.Variants) != 2 {
		t.Fatalf(fmtVariantsLenWant2, len(resp.Variants))
	}
	if len(resp.Allocation) != 2 {
		t.Fatalf(fmtAllocationLenWant2, len(resp.Allocation))
	}
}

func TestHandleGetExperiment_NotFound(t *testing.T) {
	em := &mockExperimentManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.Experiment, error) {
			return nil, nil
		},
	}

	mux := http.NewServeMux()
	mux.Handle(routeGetExperimentByKey, HandleGetExperiment(em))

	req := httptest.NewRequest(http.MethodGet, pathExperimentsPrefix+"nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusNotFound)
	}
}

func TestHandleGetExperiment_Error(t *testing.T) {
	em := &mockExperimentManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.Experiment, error) {
			return nil, errStore
		},
	}

	mux := http.NewServeMux()
	mux.Handle(routeGetExperimentByKey, HandleGetExperiment(em))

	req := httptest.NewRequest(http.MethodGet, pathExperimentsPrefix+experimentKeyOnboarding, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusInternalServerError)
	}
}

func TestHandleCreateExperiment_Valid(t *testing.T) {
	em := &mockExperimentManager{
		createFunc: func(_ context.Context, _ string, exp *pb.Experiment) (*pb.Experiment, error) {
			exp.CreatedAt = 1700000000
			exp.Status = statusDraft
			return exp, nil
		},
	}

	body := `{"key":"onboarding","name":"Onboarding Flow","stickyAssignment":true,` +
		`"variants":[{"key":"control","description":"Original"},{"key":"variant-a","description":"New"}],` +
		`"allocation":[{"variantKey":"control","percentage":50},{"variantKey":"variant-a","percentage":50}]}`
	req := httptest.NewRequest(http.MethodPost, pathExperiments, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	HandleCreateExperiment(em).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusCreated)
	}

	var resp experimentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if resp.Key != experimentKeyOnboarding {
		t.Errorf(fmtKeyWant, resp.Key, experimentKeyOnboarding)
	}
	if len(resp.Variants) != 2 {
		t.Fatalf(fmtVariantsLenWant2, len(resp.Variants))
	}
	if len(resp.Allocation) != 2 {
		t.Fatalf(fmtAllocationLenWant2, len(resp.Allocation))
	}
	if !resp.StickyAssignment {
		t.Error("expected stickyAssignment = true")
	}
}

func TestHandleCreateExperiment_MissingKey(t *testing.T) {
	em := &mockExperimentManager{}

	body := `{"name":"Onboarding Flow"}`
	req := httptest.NewRequest(http.MethodPost, pathExperiments, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	HandleCreateExperiment(em).ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleCreateExperiment_InvalidJSON(t *testing.T) {
	em := &mockExperimentManager{}

	req := httptest.NewRequest(http.MethodPost, pathExperiments, bytes.NewBufferString(`{bad`))
	w := httptest.NewRecorder()
	HandleCreateExperiment(em).ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleCreateExperiment_StoreError(t *testing.T) {
	em := &mockExperimentManager{
		createFunc: func(_ context.Context, _ string, _ *pb.Experiment) (*pb.Experiment, error) {
			return nil, errStore
		},
	}

	body := `{"key":"fail-exp","name":"Fail"}`
	req := httptest.NewRequest(http.MethodPost, pathExperiments, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	HandleCreateExperiment(em).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusInternalServerError)
	}
}

func TestHandleUpdateExperiment_Valid(t *testing.T) {
	em := &mockExperimentManager{
		updateFunc: func(_ context.Context, _ string, exp *pb.Experiment) (*pb.Experiment, error) {
			exp.UpdatedAt = 1700000001
			return exp, nil
		},
	}

	body := `{"name":"Updated Flow","stickyAssignment":false,` +
		`"variants":[{"key":"control","description":"Orig"},{"key":"variant-b","description":"Alt"}],` +
		`"allocation":[{"variantKey":"control","percentage":30},{"variantKey":"variant-b","percentage":70}]}`
	mux := http.NewServeMux()
	mux.Handle(routePutExperimentByKey, HandleUpdateExperiment(em))

	req := httptest.NewRequest(http.MethodPut, pathExperimentsPrefix+experimentKeyOnboarding, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusOK)
	}

	var resp experimentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if resp.Key != experimentKeyOnboarding {
		t.Errorf(fmtKeyWant, resp.Key, experimentKeyOnboarding)
	}
}

func TestHandleUpdateExperiment_InvalidJSON(t *testing.T) {
	em := &mockExperimentManager{}

	mux := http.NewServeMux()
	mux.Handle(routePutExperimentByKey, HandleUpdateExperiment(em))

	req := httptest.NewRequest(http.MethodPut, pathExperimentsPrefix+experimentKeyOnboarding, bytes.NewBufferString(`{bad`))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleUpdateExperiment_StoreError(t *testing.T) {
	em := &mockExperimentManager{
		updateFunc: func(_ context.Context, _ string, _ *pb.Experiment) (*pb.Experiment, error) {
			return nil, errStore
		},
	}

	body := `{"name":"Updated"}`
	mux := http.NewServeMux()
	mux.Handle(routePutExperimentByKey, HandleUpdateExperiment(em))

	req := httptest.NewRequest(http.MethodPut, pathExperimentsPrefix+experimentKeyOnboarding, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusInternalServerError)
	}
}

func TestHandleStartExperiment_FromDraft(t *testing.T) {
	em := lifecycleMock(statusDraft, statusRunning)

	mux := http.NewServeMux()
	mux.Handle(routePostExperimentStart, HandleStartExperiment(em))

	req := httptest.NewRequest(http.MethodPost, pathExperimentsPrefix+experimentKeyOnboarding+suffixStart, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusOK)
	}

	var resp experimentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if resp.Status != statusRunning {
		t.Errorf(fmtStatusFieldWant, resp.Status, statusRunning)
	}
}

func TestHandleStartExperiment_FromPaused(t *testing.T) {
	em := lifecycleMock(statusPaused, statusRunning)

	mux := http.NewServeMux()
	mux.Handle(routePostExperimentStart, HandleStartExperiment(em))

	req := httptest.NewRequest(http.MethodPost, pathExperimentsPrefix+experimentKeyOnboarding+suffixStart, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusOK)
	}
}

func TestHandleStartExperiment_FromRunning_Conflict(t *testing.T) {
	em := &mockExperimentManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.Experiment, error) {
			return sampleExperiment(statusRunning), nil
		},
	}

	mux := http.NewServeMux()
	mux.Handle(routePostExperimentStart, HandleStartExperiment(em))

	req := httptest.NewRequest(http.MethodPost, pathExperimentsPrefix+experimentKeyOnboarding+suffixStart, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusConflict)
	}
}

func TestHandleStartExperiment_NotFound(t *testing.T) {
	em := &mockExperimentManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.Experiment, error) {
			return nil, nil
		},
	}

	mux := http.NewServeMux()
	mux.Handle(routePostExperimentStart, HandleStartExperiment(em))

	req := httptest.NewRequest(http.MethodPost, pathExperimentsPrefix+experimentKeyOnboarding+suffixStart, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusNotFound)
	}
}

func TestHandlePauseExperiment_FromRunning(t *testing.T) {
	em := lifecycleMock(statusRunning, statusPaused)

	mux := http.NewServeMux()
	mux.Handle(routePostExperimentPause, HandlePauseExperiment(em))

	req := httptest.NewRequest(http.MethodPost, pathExperimentsPrefix+experimentKeyOnboarding+suffixPause, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusOK)
	}

	var resp experimentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if resp.Status != statusPaused {
		t.Errorf(fmtStatusFieldWant, resp.Status, statusPaused)
	}
}

func TestHandlePauseExperiment_FromDraft_Conflict(t *testing.T) {
	em := &mockExperimentManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.Experiment, error) {
			return sampleExperiment(statusDraft), nil
		},
	}

	mux := http.NewServeMux()
	mux.Handle(routePostExperimentPause, HandlePauseExperiment(em))

	req := httptest.NewRequest(http.MethodPost, pathExperimentsPrefix+experimentKeyOnboarding+suffixPause, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusConflict)
	}
}

func TestHandlePauseExperiment_NotFound(t *testing.T) {
	em := &mockExperimentManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.Experiment, error) {
			return nil, nil
		},
	}

	mux := http.NewServeMux()
	mux.Handle(routePostExperimentPause, HandlePauseExperiment(em))

	req := httptest.NewRequest(http.MethodPost, pathExperimentsPrefix+experimentKeyOnboarding+suffixPause, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusNotFound)
	}
}

func TestHandleCompleteExperiment_FromRunning(t *testing.T) {
	em := lifecycleMock(statusRunning, statusCompleted)

	mux := http.NewServeMux()
	mux.Handle(routePostExperimentComplete, HandleCompleteExperiment(em))

	req := httptest.NewRequest(http.MethodPost, pathExperimentsPrefix+experimentKeyOnboarding+suffixComplete, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusOK)
	}

	var resp experimentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if resp.Status != statusCompleted {
		t.Errorf(fmtStatusFieldWant, resp.Status, statusCompleted)
	}
}

func TestHandleCompleteExperiment_FromPaused(t *testing.T) {
	em := lifecycleMock(statusPaused, statusCompleted)

	mux := http.NewServeMux()
	mux.Handle(routePostExperimentComplete, HandleCompleteExperiment(em))

	req := httptest.NewRequest(http.MethodPost, pathExperimentsPrefix+experimentKeyOnboarding+suffixComplete, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusOK)
	}
}

func TestHandleCompleteExperiment_FromDraft_Conflict(t *testing.T) {
	em := &mockExperimentManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.Experiment, error) {
			return sampleExperiment(statusDraft), nil
		},
	}

	mux := http.NewServeMux()
	mux.Handle(routePostExperimentComplete, HandleCompleteExperiment(em))

	req := httptest.NewRequest(http.MethodPost, pathExperimentsPrefix+experimentKeyOnboarding+suffixComplete, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusConflict)
	}
}

func TestHandleCompleteExperiment_NotFound(t *testing.T) {
	em := &mockExperimentManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.Experiment, error) {
			return nil, nil
		},
	}

	mux := http.NewServeMux()
	mux.Handle(routePostExperimentComplete, HandleCompleteExperiment(em))

	req := httptest.NewRequest(http.MethodPost, pathExperimentsPrefix+experimentKeyOnboarding+suffixComplete, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusNotFound)
	}
}

func TestHandleStartExperiment_GetError(t *testing.T) {
	em := &mockExperimentManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.Experiment, error) {
			return nil, errStore
		},
	}

	mux := http.NewServeMux()
	mux.Handle(routePostExperimentStart, HandleStartExperiment(em))

	req := httptest.NewRequest(http.MethodPost, pathExperimentsPrefix+experimentKeyOnboarding+suffixStart, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusInternalServerError)
	}
}

func TestHandleStartExperiment_UpdateError(t *testing.T) {
	em := &mockExperimentManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.Experiment, error) {
			return sampleExperiment(statusDraft), nil
		},
		updateFunc: func(_ context.Context, _ string, _ *pb.Experiment) (*pb.Experiment, error) {
			return nil, errStore
		},
	}

	mux := http.NewServeMux()
	mux.Handle(routePostExperimentStart, HandleStartExperiment(em))

	req := httptest.NewRequest(http.MethodPost, pathExperimentsPrefix+experimentKeyOnboarding+suffixStart, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusInternalServerError)
	}
}

func lifecycleMock(currentStatus, targetStatus string) *mockExperimentManager {
	return &mockExperimentManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.Experiment, error) {
			return sampleExperiment(currentStatus), nil
		},
		updateFunc: func(_ context.Context, _ string, exp *pb.Experiment) (*pb.Experiment, error) {
			exp.UpdatedAt = 1700000001
			exp.Status = targetStatus
			return exp, nil
		},
	}
}
