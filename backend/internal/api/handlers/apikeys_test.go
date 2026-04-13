package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

const (
	pathAPIKeys       = "/api/v1/api-keys"
	routeDeleteAPIKey = "DELETE /api-keys/{keyId}"
	pathAPIKeysPrefix = "/api-keys/"
	apiKeyID1         = "key-1"
	apiKeyName1       = "my-api-key"
	rawKeyPrefixEval  = "ba_eval_"
)

type mockAPIKeyManager struct {
	listFunc   func(ctx context.Context, tenantID string, page, perPage int) ([]*pb.APIKey, int, error)
	createFunc func(ctx context.Context, tenantID string, key *pb.APIKey) (*pb.APIKey, error)
	revokeFunc func(ctx context.Context, tenantID, keyID string) error
}

func (m *mockAPIKeyManager) ListAPIKeys(ctx context.Context, tenantID string, page, perPage int) ([]*pb.APIKey, int, error) {
	return m.listFunc(ctx, tenantID, page, perPage)
}

func (m *mockAPIKeyManager) CreateAPIKey(ctx context.Context, tenantID string, key *pb.APIKey) (*pb.APIKey, error) {
	return m.createFunc(ctx, tenantID, key)
}

func (m *mockAPIKeyManager) RevokeAPIKey(ctx context.Context, tenantID, keyID string) error {
	return m.revokeFunc(ctx, tenantID, keyID)
}

func TestHandleListAPIKeys(t *testing.T) {
	km := &mockAPIKeyManager{
		listFunc: func(_ context.Context, _ string, _, _ int) ([]*pb.APIKey, int, error) {
			return []*pb.APIKey{
				{Id: apiKeyID1, KeyPrefix: "ba_eval_", Scope: scopeEvaluation, Name: apiKeyName1, CreatedAt: 1700000000},
				{Id: "key-2", KeyPrefix: "ba_mgmt_", Scope: scopeManagement, Name: "revoked-key", CreatedAt: 1700000000, RevokedAt: 1700000100},
			}, 2, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, pathAPIKeys+"?page=1&perPage=10", nil)
	w := httptest.NewRecorder()
	HandleListAPIKeys(km).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusOK)
	}

	var resp struct {
		Data       []apiKeyResponse   `json:"data"`
		Pagination PaginationResponse `json:"pagination"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("data len = %d, want 2", len(resp.Data))
	}
	if resp.Data[0].ID != apiKeyID1 {
		t.Errorf("data[0].id = %q, want %q", resp.Data[0].ID, apiKeyID1)
	}
	if resp.Data[0].Status != statusActive {
		t.Errorf("data[0].status = %q, want %q", resp.Data[0].Status, statusActive)
	}
	if resp.Data[0].CreatedAt == nil {
		t.Error("expected data[0].createdAt, got nil")
	}
	if resp.Data[1].Status != statusRevoked {
		t.Errorf("data[1].status = %q, want %q", resp.Data[1].Status, statusRevoked)
	}
	if resp.Pagination.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Pagination.Total)
	}
	if resp.Pagination.TotalPages != 1 {
		t.Errorf("totalPages = %d, want 1", resp.Pagination.TotalPages)
	}
}

func TestHandleListAPIKeys_Error(t *testing.T) {
	km := &mockAPIKeyManager{
		listFunc: func(_ context.Context, _ string, _, _ int) ([]*pb.APIKey, int, error) {
			return nil, 0, errStore
		},
	}

	req := httptest.NewRequest(http.MethodGet, pathAPIKeys, nil)
	w := httptest.NewRecorder()
	HandleListAPIKeys(km).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusInternalServerError)
	}
}

func TestHandleCreateAPIKey_Valid(t *testing.T) {
	km := &mockAPIKeyManager{
		createFunc: func(_ context.Context, _ string, key *pb.APIKey) (*pb.APIKey, error) {
			key.Id = apiKeyID1
			key.CreatedAt = 1700000000
			return key, nil
		},
	}

	body := `{"name":"` + apiKeyName1 + `","scope":"evaluation"}`
	req := httptest.NewRequest(http.MethodPost, pathAPIKeys, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	HandleCreateAPIKey(km).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusCreated)
	}

	var resp apiKeyCreateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if resp.ID != apiKeyID1 {
		t.Errorf("id = %q, want %q", resp.ID, apiKeyID1)
	}
	if !strings.HasPrefix(resp.RawKey, rawKeyPrefixEval) {
		t.Errorf("rawKey = %q, want prefix %q", resp.RawKey, rawKeyPrefixEval)
	}
	expectedLen := len(rawKeyPrefixEval) + rawKeyBytes*2
	if len(resp.RawKey) != expectedLen {
		t.Errorf("rawKey length = %d, want %d", len(resp.RawKey), expectedLen)
	}
	if resp.Scope != scopeEvaluation {
		t.Errorf("scope = %q, want %q", resp.Scope, scopeEvaluation)
	}
	if resp.Name != apiKeyName1 {
		t.Errorf("name = %q, want %q", resp.Name, apiKeyName1)
	}
	if resp.CreatedAt == nil {
		t.Error("expected createdAt, got nil")
	}
}

func TestHandleCreateAPIKey_ManagementScope(t *testing.T) {
	km := &mockAPIKeyManager{
		createFunc: func(_ context.Context, _ string, key *pb.APIKey) (*pb.APIKey, error) {
			key.Id = apiKeyID1
			key.CreatedAt = 1700000000
			return key, nil
		},
	}

	body := `{"name":"mgmt-key","scope":"management"}`
	req := httptest.NewRequest(http.MethodPost, pathAPIKeys, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	HandleCreateAPIKey(km).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusCreated)
	}

	var resp apiKeyCreateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if !strings.HasPrefix(resp.RawKey, "ba_mgmt_") {
		t.Errorf("rawKey = %q, want prefix %q", resp.RawKey, "ba_mgmt_")
	}
}

func TestHandleCreateAPIKey_MissingName(t *testing.T) {
	km := &mockAPIKeyManager{}

	body := `{"scope":"evaluation"}`
	req := httptest.NewRequest(http.MethodPost, pathAPIKeys, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	HandleCreateAPIKey(km).ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleCreateAPIKey_MissingScope(t *testing.T) {
	km := &mockAPIKeyManager{}

	body := `{"name":"test-key"}`
	req := httptest.NewRequest(http.MethodPost, pathAPIKeys, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	HandleCreateAPIKey(km).ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleCreateAPIKey_InvalidScope(t *testing.T) {
	km := &mockAPIKeyManager{}

	body := `{"name":"test-key","scope":"admin"}`
	req := httptest.NewRequest(http.MethodPost, pathAPIKeys, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	HandleCreateAPIKey(km).ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleCreateAPIKey_InvalidJSON(t *testing.T) {
	km := &mockAPIKeyManager{}

	req := httptest.NewRequest(http.MethodPost, pathAPIKeys, bytes.NewBufferString(`{bad`))
	w := httptest.NewRecorder()
	HandleCreateAPIKey(km).ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleCreateAPIKey_StoreError(t *testing.T) {
	km := &mockAPIKeyManager{
		createFunc: func(_ context.Context, _ string, _ *pb.APIKey) (*pb.APIKey, error) {
			return nil, errStore
		},
	}

	body := `{"name":"test-key","scope":"evaluation"}`
	req := httptest.NewRequest(http.MethodPost, pathAPIKeys, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	HandleCreateAPIKey(km).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusInternalServerError)
	}
}

func TestHandleRevokeAPIKey_Valid(t *testing.T) {
	km := &mockAPIKeyManager{
		revokeFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
	}

	mux := http.NewServeMux()
	mux.Handle(routeDeleteAPIKey, HandleRevokeAPIKey(km))

	req := httptest.NewRequest(http.MethodDelete, pathAPIKeysPrefix+apiKeyID1, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusNoContent)
	}
}

func TestHandleRevokeAPIKey_Error(t *testing.T) {
	km := &mockAPIKeyManager{
		revokeFunc: func(_ context.Context, _, _ string) error {
			return errStore
		},
	}

	mux := http.NewServeMux()
	mux.Handle(routeDeleteAPIKey, HandleRevokeAPIKey(km))

	req := httptest.NewRequest(http.MethodDelete, pathAPIKeysPrefix+apiKeyID1, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf(fmtWantStatus, w.Code, http.StatusInternalServerError)
	}
}
