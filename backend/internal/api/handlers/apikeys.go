package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/api/problem"
	"github.com/orlandoburli/feature-bacon/internal/auth"
)

const (
	statusActive    = "active"
	statusRevoked   = "revoked"
	scopeEvaluation = "evaluation"
	scopeManagement = "management"
	rawKeyBytes     = 16
)

var scopeAbbrev = map[string]string{
	scopeEvaluation: "eval",
	scopeManagement: "mgmt",
}

type APIKeyManager interface {
	ListAPIKeys(ctx context.Context, tenantID string, page, perPage int) ([]*pb.APIKey, int, error)
	CreateAPIKey(ctx context.Context, tenantID string, key *pb.APIKey) (*pb.APIKey, error)
	RevokeAPIKey(ctx context.Context, tenantID, keyID string) error
}

type apiKeyResponse struct {
	ID        string     `json:"id"`
	Prefix    string     `json:"prefix"`
	Scope     string     `json:"scope"`
	Name      string     `json:"name"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	Status    string     `json:"status"`
}

type apiKeyCreateRequest struct {
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

type apiKeyCreateResponse struct {
	ID        string     `json:"id"`
	RawKey    string     `json:"rawKey"`
	Prefix    string     `json:"prefix"`
	Scope     string     `json:"scope"`
	Name      string     `json:"name"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
}

func HandleListAPIKeys(km APIKeyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := tenantFromContext(r)
		page, perPage := ParsePagination(r)

		keys, total, err := km.ListAPIKeys(r.Context(), tenantID, page, perPage)
		if err != nil {
			problem.Write(w, problem.InternalError(err.Error(), r.URL.Path))
			return
		}

		totalPages := 0
		if perPage > 0 {
			totalPages = (total + perPage - 1) / perPage
		}

		data := make([]apiKeyResponse, len(keys))
		for i, k := range keys {
			data[i] = protoToAPIKeyResponse(k)
		}

		w.Header().Set(headerContentType, contentTypeJSON)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": data,
			"pagination": PaginationResponse{
				Page:       page,
				PerPage:    perPage,
				Total:      total,
				TotalPages: totalPages,
			},
		})
	}
}

func HandleCreateAPIKey(km APIKeyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req apiKeyCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			problem.Write(w, problem.ValidationError("invalid JSON body", r.URL.Path))
			return
		}

		if req.Name == "" {
			problem.Write(w, problem.ValidationError("name is required", r.URL.Path))
			return
		}
		if req.Scope == "" {
			problem.Write(w, problem.ValidationError("scope is required", r.URL.Path))
			return
		}
		if !isValidScope(req.Scope) {
			problem.Write(w, problem.ValidationError("scope must be evaluation or management", r.URL.Path))
			return
		}

		rawKey, err := generateRawKey(req.Scope)
		if err != nil {
			problem.Write(w, problem.InternalError(err.Error(), r.URL.Path))
			return
		}

		tenantID := tenantFromContext(r)
		pbKey := &pb.APIKey{
			KeyHash:   auth.HashKey(rawKey),
			KeyPrefix: auth.Prefix(rawKey),
			Scope:     req.Scope,
			Name:      req.Name,
		}

		created, err := km.CreateAPIKey(r.Context(), tenantID, pbKey)
		if err != nil {
			problem.Write(w, problem.InternalError(err.Error(), r.URL.Path))
			return
		}

		resp := apiKeyCreateResponse{
			ID:     created.Id,
			RawKey: rawKey,
			Prefix: created.KeyPrefix,
			Scope:  created.Scope,
			Name:   created.Name,
		}
		if created.CreatedAt != 0 {
			t := time.Unix(created.CreatedAt, 0).UTC()
			resp.CreatedAt = &t
		}

		w.Header().Set(headerContentType, contentTypeJSON)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func HandleRevokeAPIKey(km APIKeyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := tenantFromContext(r)
		keyID := r.PathValue("keyId")

		if err := km.RevokeAPIKey(r.Context(), tenantID, keyID); err != nil {
			problem.Write(w, problem.InternalError(err.Error(), r.URL.Path))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func protoToAPIKeyResponse(k *pb.APIKey) apiKeyResponse {
	resp := apiKeyResponse{
		ID:     k.Id,
		Prefix: k.KeyPrefix,
		Scope:  k.Scope,
		Name:   k.Name,
		Status: statusActive,
	}
	if k.CreatedAt != 0 {
		t := time.Unix(k.CreatedAt, 0).UTC()
		resp.CreatedAt = &t
	}
	if k.RevokedAt != 0 {
		resp.Status = statusRevoked
	}
	return resp
}

func isValidScope(scope string) bool {
	return scope == scopeEvaluation || scope == scopeManagement
}

func generateRawKey(scope string) (string, error) {
	b := make([]byte, rawKeyBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}
	return fmt.Sprintf("ba_%s_%s", scopeAbbrev[scope], hex.EncodeToString(b)), nil
}
