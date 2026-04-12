package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

const (
	pathFlags                = "/api/v1/flags"
	pathFlagByKey            = "/api/v1/flags/my-flag"
	flagKeyMyFlag            = "my-flag"
	fmtWantStatus            = "status = %d, want %d"
	fmtDecode                = "decode: %v"
	fmtKeyWant               = "key = %q, want %q"
	routeGetFlagByKey        = "GET /flags/{flagKey}"
	routePutFlagByKey        = "PUT /flags/{flagKey}"
	pathFlagsPrefix          = "/flags/"
	fmtPerPageWant           = "perPage = %d, want %d"
	fmtRulesLenWant1         = "rules len = %d, want 1"
	msgExpectedDefaultResult = "expected defaultResult, got nil"
	fmtDefaultResultVariant  = "defaultResult.variant = %q, want %q"
	emailAdmin               = "admin@test.com"
	emailEditor              = "editor@test.com"
)

var errStore = errors.New("store failure")

type mockFlagManager struct {
	getFunc    func(ctx context.Context, tenantID, flagKey string) (*pb.FlagDefinition, error)
	listFunc   func(ctx context.Context, tenantID string, page, perPage int) ([]*pb.FlagDefinition, int, error)
	createFunc func(ctx context.Context, tenantID string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error)
	updateFunc func(ctx context.Context, tenantID string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error)
	deleteFunc func(ctx context.Context, tenantID, flagKey string) error
}

func (m *mockFlagManager) GetFlag(ctx context.Context, tenantID, flagKey string) (*pb.FlagDefinition, error) {
	return m.getFunc(ctx, tenantID, flagKey)
}

func (m *mockFlagManager) ListFlags(ctx context.Context, tenantID string, page, perPage int) ([]*pb.FlagDefinition, int, error) {
	return m.listFunc(ctx, tenantID, page, perPage)
}

func (m *mockFlagManager) CreateFlag(ctx context.Context, tenantID string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error) {
	return m.createFunc(ctx, tenantID, flag)
}

func (m *mockFlagManager) UpdateFlag(ctx context.Context, tenantID string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error) {
	return m.updateFunc(ctx, tenantID, flag)
}

func (m *mockFlagManager) DeleteFlag(ctx context.Context, tenantID, flagKey string) error {
	return m.deleteFunc(ctx, tenantID, flagKey)
}

func sampleFlag() *pb.FlagDefinition {
	return &pb.FlagDefinition{
		Key:         flagKeyMyFlag,
		Type:        "boolean",
		Semantics:   "flag",
		Enabled:     true,
		Description: "test flag",
		CreatedAt:   1700000000,
		UpdatedAt:   1700000000,
		DefaultResult: &pb.EvalResult{
			Enabled: true,
			Variant: "on",
		},
	}
}

func TestHandleListFlags(t *testing.T) {
	fm := &mockFlagManager{
		listFunc: func(_ context.Context, _ string, _, _ int) ([]*pb.FlagDefinition, int, error) {
			return []*pb.FlagDefinition{sampleFlag()}, 1, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, pathFlags+"?page=1&perPage=10", nil)
	w := httptest.NewRecorder()
	HandleListFlags(fm).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusOK)
	}

	var resp struct {
		Data       []flagResponse     `json:"data"`
		Pagination PaginationResponse `json:"pagination"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].Key != flagKeyMyFlag {
		t.Errorf(fmtKeyWant, resp.Data[0].Key, flagKeyMyFlag)
	}
	if resp.Pagination.Total != 1 {
		t.Errorf("total = %d, want 1", resp.Pagination.Total)
	}
	if resp.Pagination.TotalPages != 1 {
		t.Errorf("totalPages = %d, want 1", resp.Pagination.TotalPages)
	}
}

func TestHandleGetFlag_Found(t *testing.T) {
	fm := &mockFlagManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.FlagDefinition, error) {
			return sampleFlag(), nil
		},
	}

	mux := http.NewServeMux()
	mux.Handle(routeGetFlagByKey, HandleGetFlag(fm))

	req := httptest.NewRequest(http.MethodGet, pathFlagsPrefix+flagKeyMyFlag, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusOK)
	}

	var resp flagResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if resp.Key != flagKeyMyFlag {
		t.Errorf(fmtKeyWant, resp.Key, flagKeyMyFlag)
	}
}

func TestHandleGetFlag_NotFound(t *testing.T) {
	fm := &mockFlagManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.FlagDefinition, error) {
			return nil, nil
		},
	}

	mux := http.NewServeMux()
	mux.Handle(routeGetFlagByKey, HandleGetFlag(fm))

	req := httptest.NewRequest(http.MethodGet, pathFlagsPrefix+"nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusNotFound)
	}
}

func TestHandleCreateFlag_Valid(t *testing.T) {
	fm := &mockFlagManager{
		createFunc: func(_ context.Context, _ string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error) {
			flag.CreatedAt = 1700000000
			return flag, nil
		},
	}

	body := `{"key":"new-flag","type":"boolean","semantics":"flag","enabled":true}`
	req := httptest.NewRequest(http.MethodPost, pathFlags, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	HandleCreateFlag(fm).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusCreated)
	}

	var resp flagResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if resp.Key != "new-flag" {
		t.Errorf(fmtKeyWant, resp.Key, "new-flag")
	}
}

func TestHandleCreateFlag_MissingKey(t *testing.T) {
	fm := &mockFlagManager{}

	body := `{"type":"boolean","semantics":"flag"}`
	req := httptest.NewRequest(http.MethodPost, pathFlags, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	HandleCreateFlag(fm).ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleCreateFlag_MissingType(t *testing.T) {
	fm := &mockFlagManager{}

	body := `{"key":"new-flag","semantics":"flag"}`
	req := httptest.NewRequest(http.MethodPost, pathFlags, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	HandleCreateFlag(fm).ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleCreateFlag_MissingSemantics(t *testing.T) {
	fm := &mockFlagManager{}

	body := `{"key":"new-flag","type":"boolean"}`
	req := httptest.NewRequest(http.MethodPost, pathFlags, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	HandleCreateFlag(fm).ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleUpdateFlag_Valid(t *testing.T) {
	fm := &mockFlagManager{
		updateFunc: func(_ context.Context, _ string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error) {
			flag.UpdatedAt = 1700000001
			return flag, nil
		},
	}

	body := `{"type":"boolean","semantics":"flag","enabled":false,"description":"updated"}`
	mux := http.NewServeMux()
	mux.Handle(routePutFlagByKey, HandleUpdateFlag(fm))

	req := httptest.NewRequest(http.MethodPut, pathFlagsPrefix+flagKeyMyFlag, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusOK)
	}

	var resp flagResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if resp.Key != flagKeyMyFlag {
		t.Errorf(fmtKeyWant, resp.Key, flagKeyMyFlag)
	}
}

func TestHandleDeleteFlag_Valid(t *testing.T) {
	fm := &mockFlagManager{
		deleteFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
	}

	mux := http.NewServeMux()
	mux.Handle("DELETE /flags/{flagKey}", HandleDeleteFlag(fm))

	req := httptest.NewRequest(http.MethodDelete, pathFlagsPrefix+flagKeyMyFlag, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusNoContent)
	}
}

func TestHandleListFlags_Error(t *testing.T) {
	fm := &mockFlagManager{
		listFunc: func(_ context.Context, _ string, _, _ int) ([]*pb.FlagDefinition, int, error) {
			return nil, 0, errStore
		},
	}

	req := httptest.NewRequest(http.MethodGet, pathFlags, nil)
	w := httptest.NewRecorder()
	HandleListFlags(fm).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusInternalServerError)
	}
}

func TestHandleGetFlag_Error(t *testing.T) {
	fm := &mockFlagManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.FlagDefinition, error) {
			return nil, errStore
		},
	}

	mux := http.NewServeMux()
	mux.Handle(routeGetFlagByKey, HandleGetFlag(fm))

	req := httptest.NewRequest(http.MethodGet, pathFlagsPrefix+flagKeyMyFlag, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusInternalServerError)
	}
}

func TestHandleCreateFlag_InvalidJSON(t *testing.T) {
	fm := &mockFlagManager{}

	req := httptest.NewRequest(http.MethodPost, pathFlags, bytes.NewBufferString(`{bad`))
	w := httptest.NewRecorder()
	HandleCreateFlag(fm).ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleCreateFlag_WithRulesAndDefault(t *testing.T) {
	fm := &mockFlagManager{
		createFunc: func(_ context.Context, _ string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error) {
			flag.CreatedAt = 1700000000
			return flag, nil
		},
	}

	body := `{"key":"feat-x","type":"variant","semantics":"deterministic","enabled":true,` +
		`"description":"test","createdBy":"` + emailAdmin + `",` +
		`"rules":[{"conditions":[{"attribute":"env","operator":"equals","valueJson":"\"prod\""}],"rolloutPercentage":50,"variant":"v1"}],` +
		`"defaultResult":{"enabled":false,"variant":"control"}}`
	req := httptest.NewRequest(http.MethodPost, pathFlags, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	HandleCreateFlag(fm).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusCreated)
	}

	var resp flagResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if resp.Key != "feat-x" {
		t.Errorf(fmtKeyWant, resp.Key, "feat-x")
	}
	if len(resp.Rules) != 1 {
		t.Fatalf(fmtRulesLenWant1, len(resp.Rules))
	}
	if resp.Rules[0].Conditions[0].Attribute != "env" {
		t.Errorf("condition attribute = %q, want %q", resp.Rules[0].Conditions[0].Attribute, "env")
	}
	if resp.Rules[0].RolloutPercentage != 50 {
		t.Errorf("rollout = %d, want 50", resp.Rules[0].RolloutPercentage)
	}
	if resp.DefaultResult == nil {
		t.Fatal(msgExpectedDefaultResult)
	}
	if resp.DefaultResult.Variant != "control" {
		t.Errorf(fmtDefaultResultVariant, resp.DefaultResult.Variant, "control")
	}
	if resp.CreatedBy != emailAdmin {
		t.Errorf("createdBy = %q, want %q", resp.CreatedBy, emailAdmin)
	}
}

func TestHandleUpdateFlag_WithRulesAndDefault(t *testing.T) {
	fm := &mockFlagManager{
		updateFunc: func(_ context.Context, _ string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error) {
			flag.UpdatedAt = 1700000001
			return flag, nil
		},
	}

	body := `{"type":"variant","semantics":"deterministic","enabled":true,` +
		`"description":"updated","updatedBy":"` + emailEditor + `",` +
		`"rules":[{"conditions":[{"attribute":"region","operator":"in","valueJson":"[\"us\",\"eu\"]"}],"rolloutPercentage":75,"variant":"v2"}],` +
		`"defaultResult":{"enabled":true,"variant":"fallback"}}`
	mux := http.NewServeMux()
	mux.Handle(routePutFlagByKey, HandleUpdateFlag(fm))

	req := httptest.NewRequest(http.MethodPut, pathFlagsPrefix+flagKeyMyFlag, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusOK)
	}

	var resp flagResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if len(resp.Rules) != 1 {
		t.Fatalf(fmtRulesLenWant1, len(resp.Rules))
	}
	if resp.Rules[0].Variant != "v2" {
		t.Errorf("rule variant = %q, want %q", resp.Rules[0].Variant, "v2")
	}
	if resp.DefaultResult == nil {
		t.Fatal(msgExpectedDefaultResult)
	}
	if resp.DefaultResult.Variant != "fallback" {
		t.Errorf(fmtDefaultResultVariant, resp.DefaultResult.Variant, "fallback")
	}
	if resp.UpdatedBy != emailEditor {
		t.Errorf("updatedBy = %q, want %q", resp.UpdatedBy, emailEditor)
	}
}

func TestHandleCreateFlag_StoreError(t *testing.T) {
	fm := &mockFlagManager{
		createFunc: func(_ context.Context, _ string, _ *pb.FlagDefinition) (*pb.FlagDefinition, error) {
			return nil, errStore
		},
	}

	body := `{"key":"fail-flag","type":"boolean","semantics":"flag"}`
	req := httptest.NewRequest(http.MethodPost, pathFlags, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	HandleCreateFlag(fm).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusInternalServerError)
	}
}

func TestHandleUpdateFlag_StoreError(t *testing.T) {
	fm := &mockFlagManager{
		updateFunc: func(_ context.Context, _ string, _ *pb.FlagDefinition) (*pb.FlagDefinition, error) {
			return nil, errStore
		},
	}

	body := `{"type":"boolean","semantics":"flag","enabled":false}`
	mux := http.NewServeMux()
	mux.Handle(routePutFlagByKey, HandleUpdateFlag(fm))

	req := httptest.NewRequest(http.MethodPut, pathFlagsPrefix+flagKeyMyFlag, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusInternalServerError)
	}
}

func TestHandleUpdateFlag_InvalidJSON(t *testing.T) {
	fm := &mockFlagManager{}

	mux := http.NewServeMux()
	mux.Handle(routePutFlagByKey, HandleUpdateFlag(fm))

	req := httptest.NewRequest(http.MethodPut, pathFlagsPrefix+flagKeyMyFlag, bytes.NewBufferString(`{bad`))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleDeleteFlag_Error(t *testing.T) {
	fm := &mockFlagManager{
		deleteFunc: func(_ context.Context, _, _ string) error {
			return errStore
		},
	}

	mux := http.NewServeMux()
	mux.Handle("DELETE /flags/{flagKey}", HandleDeleteFlag(fm))

	req := httptest.NewRequest(http.MethodDelete, pathFlagsPrefix+flagKeyMyFlag, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusInternalServerError)
	}
}

func TestHandleListFlags_WithRulesAndTimestamps(t *testing.T) {
	fm := &mockFlagManager{
		listFunc: func(_ context.Context, _ string, _, _ int) ([]*pb.FlagDefinition, int, error) {
			return []*pb.FlagDefinition{
				{
					Key: flagKeyMyFlag, Type: "variant", Semantics: "deterministic",
					Enabled: true, Description: "rich flag",
					CreatedAt: 1700000000, UpdatedAt: 1700000001,
					CreatedBy: emailAdmin, UpdatedBy: emailEditor,
					Rules: []*pb.Rule{
						{
							Conditions: []*pb.Condition{
								{Attribute: "env", Operator: "equals", ValueJson: `"prod"`},
							},
							RolloutPercentage: 100,
							Variant:           "v1",
						},
					},
					DefaultResult: &pb.EvalResult{Enabled: true, Variant: "on"},
				},
			}, 1, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, pathFlags+"?page=1&perPage=10", nil)
	w := httptest.NewRecorder()
	HandleListFlags(fm).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusOK)
	}

	var resp struct {
		Data []flagResponse `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(resp.Data))
	}
	f := resp.Data[0]
	if f.CreatedAt == nil {
		t.Fatal("expected createdAt, got nil")
	}
	if f.UpdatedAt == nil {
		t.Fatal("expected updatedAt, got nil")
	}
	if len(f.Rules) != 1 {
		t.Fatalf(fmtRulesLenWant1, len(f.Rules))
	}
	if f.Rules[0].Conditions[0].Attribute != "env" {
		t.Errorf("condition attribute = %q, want %q", f.Rules[0].Conditions[0].Attribute, "env")
	}
	if f.DefaultResult == nil {
		t.Fatal(msgExpectedDefaultResult)
	}
	if f.DefaultResult.Variant != "on" {
		t.Errorf(fmtDefaultResultVariant, f.DefaultResult.Variant, "on")
	}
}

func TestParsePagination_Defaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	page, perPage := ParsePagination(req)
	if page != defaultPage {
		t.Errorf("page = %d, want %d", page, defaultPage)
	}
	if perPage != defaultPerPage {
		t.Errorf(fmtPerPageWant, perPage, defaultPerPage)
	}
}

func TestParsePagination_ClampMax(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?perPage=999", nil)
	_, perPage := ParsePagination(req)
	if perPage != maxPerPage {
		t.Errorf(fmtPerPageWant, perPage, maxPerPage)
	}
}

func TestParsePagination_Custom(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=3&perPage=50", nil)
	page, perPage := ParsePagination(req)
	if page != 3 {
		t.Errorf("page = %d, want 3", page)
	}
	if perPage != 50 {
		t.Errorf("perPage = %d, want 50", perPage)
	}
}

func TestParsePagination_InvalidValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=-1&perPage=abc", nil)
	page, perPage := ParsePagination(req)
	if page != defaultPage {
		t.Errorf("page = %d, want %d", page, defaultPage)
	}
	if perPage != defaultPerPage {
		t.Errorf(fmtPerPageWant, perPage, defaultPerPage)
	}
}
