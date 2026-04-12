package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/api/problem"
)

const (
	contentTypeJSON   = "application/json"
	headerContentType = "Content-Type"
)

type FlagManager interface {
	GetFlag(ctx context.Context, tenantID, flagKey string) (*pb.FlagDefinition, error)
	ListFlags(ctx context.Context, tenantID string, page, perPage int) ([]*pb.FlagDefinition, int, error)
	CreateFlag(ctx context.Context, tenantID string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error)
	UpdateFlag(ctx context.Context, tenantID string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error)
	DeleteFlag(ctx context.Context, tenantID, flagKey string) error
}

type flagResponse struct {
	Key           string              `json:"key"`
	Type          string              `json:"type"`
	Semantics     string              `json:"semantics"`
	Enabled       bool                `json:"enabled"`
	Description   string              `json:"description"`
	Rules         []ruleResponse      `json:"rules"`
	DefaultResult *evalResultResponse `json:"defaultResult,omitempty"`
	CreatedBy     string              `json:"createdBy"`
	UpdatedBy     string              `json:"updatedBy"`
	CreatedAt     *time.Time          `json:"createdAt,omitempty"`
	UpdatedAt     *time.Time          `json:"updatedAt,omitempty"`
}

type ruleResponse struct {
	Conditions        []conditionResponse `json:"conditions"`
	RolloutPercentage int32               `json:"rolloutPercentage"`
	Variant           string              `json:"variant"`
}

type conditionResponse struct {
	Attribute string `json:"attribute"`
	Operator  string `json:"operator"`
	ValueJSON string `json:"valueJson"`
}

type evalResultResponse struct {
	Enabled bool   `json:"enabled"`
	Variant string `json:"variant"`
}

type flagCreateRequest struct {
	Key           string             `json:"key"`
	Type          string             `json:"type"`
	Semantics     string             `json:"semantics"`
	Enabled       bool               `json:"enabled"`
	Description   string             `json:"description"`
	Rules         []ruleRequest      `json:"rules"`
	DefaultResult *evalResultRequest `json:"defaultResult,omitempty"`
	CreatedBy     string             `json:"createdBy"`
}

type flagUpdateRequest struct {
	Key           string             `json:"key"`
	Type          string             `json:"type"`
	Semantics     string             `json:"semantics"`
	Enabled       bool               `json:"enabled"`
	Description   string             `json:"description"`
	Rules         []ruleRequest      `json:"rules"`
	DefaultResult *evalResultRequest `json:"defaultResult,omitempty"`
	UpdatedBy     string             `json:"updatedBy"`
}

type ruleRequest struct {
	Conditions        []conditionRequest `json:"conditions"`
	RolloutPercentage int32              `json:"rolloutPercentage"`
	Variant           string             `json:"variant"`
}

type conditionRequest struct {
	Attribute string `json:"attribute"`
	Operator  string `json:"operator"`
	ValueJSON string `json:"valueJson"`
}

type evalResultRequest struct {
	Enabled bool   `json:"enabled"`
	Variant string `json:"variant"`
}

func HandleListFlags(fm FlagManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := tenantFromContext(r)
		page, perPage := ParsePagination(r)

		flags, total, err := fm.ListFlags(r.Context(), tenantID, page, perPage)
		if err != nil {
			problem.Write(w, problem.InternalError(err.Error(), r.URL.Path))
			return
		}

		totalPages := 0
		if perPage > 0 {
			totalPages = (total + perPage - 1) / perPage
		}

		data := make([]flagResponse, len(flags))
		for i, f := range flags {
			data[i] = protoToFlagResponse(f)
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

func HandleGetFlag(fm FlagManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := tenantFromContext(r)
		flagKey := r.PathValue("flagKey")

		flag, err := fm.GetFlag(r.Context(), tenantID, flagKey)
		if err != nil {
			problem.Write(w, problem.InternalError(err.Error(), r.URL.Path))
			return
		}
		if flag == nil {
			problem.Write(w, problem.NotFound("flag not found", r.URL.Path))
			return
		}

		resp := protoToFlagResponse(flag)
		w.Header().Set(headerContentType, contentTypeJSON)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func HandleCreateFlag(fm FlagManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req flagCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			problem.Write(w, problem.ValidationError("invalid JSON body", r.URL.Path))
			return
		}

		if req.Key == "" {
			problem.Write(w, problem.ValidationError("key is required", r.URL.Path))
			return
		}
		if req.Type == "" {
			problem.Write(w, problem.ValidationError("type is required", r.URL.Path))
			return
		}
		if req.Semantics == "" {
			problem.Write(w, problem.ValidationError("semantics is required", r.URL.Path))
			return
		}

		tenantID := tenantFromContext(r)
		pbFlag := createRequestToProto(&req)

		created, err := fm.CreateFlag(r.Context(), tenantID, pbFlag)
		if err != nil {
			problem.Write(w, problem.InternalError(err.Error(), r.URL.Path))
			return
		}

		resp := protoToFlagResponse(created)
		w.Header().Set(headerContentType, contentTypeJSON)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func HandleUpdateFlag(fm FlagManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flagKey := r.PathValue("flagKey")

		var req flagUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			problem.Write(w, problem.ValidationError("invalid JSON body", r.URL.Path))
			return
		}

		req.Key = flagKey
		tenantID := tenantFromContext(r)
		pbFlag := updateRequestToProto(&req)

		updated, err := fm.UpdateFlag(r.Context(), tenantID, pbFlag)
		if err != nil {
			problem.Write(w, problem.InternalError(err.Error(), r.URL.Path))
			return
		}

		resp := protoToFlagResponse(updated)
		w.Header().Set(headerContentType, contentTypeJSON)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func HandleDeleteFlag(fm FlagManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := tenantFromContext(r)
		flagKey := r.PathValue("flagKey")

		if err := fm.DeleteFlag(r.Context(), tenantID, flagKey); err != nil {
			problem.Write(w, problem.InternalError(err.Error(), r.URL.Path))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func protoToFlagResponse(f *pb.FlagDefinition) flagResponse {
	resp := flagResponse{
		Key:         f.Key,
		Type:        f.Type,
		Semantics:   f.Semantics,
		Enabled:     f.Enabled,
		Description: f.Description,
		CreatedBy:   f.CreatedBy,
		UpdatedBy:   f.UpdatedBy,
		Rules:       make([]ruleResponse, 0, len(f.Rules)),
	}

	if f.CreatedAt != 0 {
		t := time.Unix(f.CreatedAt, 0).UTC()
		resp.CreatedAt = &t
	}
	if f.UpdatedAt != 0 {
		t := time.Unix(f.UpdatedAt, 0).UTC()
		resp.UpdatedAt = &t
	}

	if f.DefaultResult != nil {
		resp.DefaultResult = &evalResultResponse{
			Enabled: f.DefaultResult.Enabled,
			Variant: f.DefaultResult.Variant,
		}
	}

	for _, r := range f.Rules {
		rule := ruleResponse{
			RolloutPercentage: r.RolloutPercentage,
			Variant:           r.Variant,
			Conditions:        make([]conditionResponse, 0, len(r.Conditions)),
		}
		for _, c := range r.Conditions {
			rule.Conditions = append(rule.Conditions, conditionResponse{
				Attribute: c.Attribute,
				Operator:  c.Operator,
				ValueJSON: c.ValueJson,
			})
		}
		resp.Rules = append(resp.Rules, rule)
	}

	return resp
}

func rulesToProto(rules []ruleRequest) []*pb.Rule {
	out := make([]*pb.Rule, 0, len(rules))
	for _, r := range rules {
		rule := &pb.Rule{
			RolloutPercentage: r.RolloutPercentage,
			Variant:           r.Variant,
		}
		for _, c := range r.Conditions {
			rule.Conditions = append(rule.Conditions, &pb.Condition{
				Attribute: c.Attribute,
				Operator:  c.Operator,
				ValueJson: c.ValueJSON,
			})
		}
		out = append(out, rule)
	}
	return out
}

func evalResultToProto(er *evalResultRequest) *pb.EvalResult {
	if er == nil {
		return nil
	}
	return &pb.EvalResult{Enabled: er.Enabled, Variant: er.Variant}
}

func createRequestToProto(req *flagCreateRequest) *pb.FlagDefinition {
	return &pb.FlagDefinition{
		Key:           req.Key,
		Type:          req.Type,
		Semantics:     req.Semantics,
		Enabled:       req.Enabled,
		Description:   req.Description,
		CreatedBy:     req.CreatedBy,
		DefaultResult: evalResultToProto(req.DefaultResult),
		Rules:         rulesToProto(req.Rules),
	}
}

func updateRequestToProto(req *flagUpdateRequest) *pb.FlagDefinition {
	return &pb.FlagDefinition{
		Key:           req.Key,
		Type:          req.Type,
		Semantics:     req.Semantics,
		Enabled:       req.Enabled,
		Description:   req.Description,
		UpdatedBy:     req.UpdatedBy,
		DefaultResult: evalResultToProto(req.DefaultResult),
		Rules:         rulesToProto(req.Rules),
	}
}
