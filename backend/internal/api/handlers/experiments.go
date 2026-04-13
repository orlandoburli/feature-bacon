package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/api/problem"
)

const (
	statusDraft     = "draft"
	statusRunning   = "running"
	statusPaused    = "paused"
	statusCompleted = "completed"
)

type ExperimentManager interface {
	GetExperiment(ctx context.Context, tenantID, experimentKey string) (*pb.Experiment, error)
	ListExperiments(ctx context.Context, tenantID string, page, perPage int) ([]*pb.Experiment, int, error)
	CreateExperiment(ctx context.Context, tenantID string, exp *pb.Experiment) (*pb.Experiment, error)
	UpdateExperiment(ctx context.Context, tenantID string, exp *pb.Experiment) (*pb.Experiment, error)
}

type experimentResponse struct {
	Key              string               `json:"key"`
	Name             string               `json:"name"`
	Status           string               `json:"status"`
	StickyAssignment bool                 `json:"stickyAssignment"`
	Variants         []variantResponse    `json:"variants"`
	Allocation       []allocationResponse `json:"allocation"`
	CreatedAt        *time.Time           `json:"createdAt,omitempty"`
	UpdatedAt        *time.Time           `json:"updatedAt,omitempty"`
}

type variantResponse struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

type allocationResponse struct {
	VariantKey string `json:"variantKey"`
	Percentage int32  `json:"percentage"`
}

type experimentCreateRequest struct {
	Key              string              `json:"key"`
	Name             string              `json:"name"`
	StickyAssignment bool                `json:"stickyAssignment"`
	Variants         []variantRequest    `json:"variants"`
	Allocation       []allocationRequest `json:"allocation"`
}

type experimentUpdateRequest struct {
	Key              string              `json:"key"`
	Name             string              `json:"name"`
	StickyAssignment bool                `json:"stickyAssignment"`
	Variants         []variantRequest    `json:"variants"`
	Allocation       []allocationRequest `json:"allocation"`
}

type variantRequest struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

type allocationRequest struct {
	VariantKey string `json:"variantKey"`
	Percentage int32  `json:"percentage"`
}

func HandleListExperiments(em ExperimentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := tenantFromContext(r)
		page, perPage := ParsePagination(r)

		experiments, total, err := em.ListExperiments(r.Context(), tenantID, page, perPage)
		if err != nil {
			problem.Write(w, problem.InternalError(err.Error(), r.URL.Path))
			return
		}

		totalPages := 0
		if perPage > 0 {
			totalPages = (total + perPage - 1) / perPage
		}

		data := make([]experimentResponse, len(experiments))
		for i, e := range experiments {
			data[i] = protoToExperimentResponse(e)
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

func HandleGetExperiment(em ExperimentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exp, ok := getExperimentOr404(em, w, r)
		if !ok {
			return
		}

		resp := protoToExperimentResponse(exp)
		w.Header().Set(headerContentType, contentTypeJSON)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func HandleCreateExperiment(em ExperimentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req experimentCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			problem.Write(w, problem.ValidationError("invalid JSON body", r.URL.Path))
			return
		}

		if req.Key == "" {
			problem.Write(w, problem.ValidationError("key is required", r.URL.Path))
			return
		}

		tenantID := tenantFromContext(r)
		pbExp := experimentRequestToProto(req.Key, req.Name, req.StickyAssignment, req.Variants, req.Allocation)

		created, err := em.CreateExperiment(r.Context(), tenantID, pbExp)
		if err != nil {
			problem.Write(w, problem.InternalError(err.Error(), r.URL.Path))
			return
		}

		resp := protoToExperimentResponse(created)
		w.Header().Set(headerContentType, contentTypeJSON)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func HandleUpdateExperiment(em ExperimentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		experimentKey := r.PathValue("experimentKey")

		var req experimentUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			problem.Write(w, problem.ValidationError("invalid JSON body", r.URL.Path))
			return
		}

		req.Key = experimentKey
		tenantID := tenantFromContext(r)
		pbExp := experimentRequestToProto(req.Key, req.Name, req.StickyAssignment, req.Variants, req.Allocation)

		updated, err := em.UpdateExperiment(r.Context(), tenantID, pbExp)
		if err != nil {
			problem.Write(w, problem.InternalError(err.Error(), r.URL.Path))
			return
		}

		resp := protoToExperimentResponse(updated)
		w.Header().Set(headerContentType, contentTypeJSON)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func HandleStartExperiment(em ExperimentManager) http.HandlerFunc {
	return handleLifecycleTransition(em, statusRunning, statusDraft, statusPaused)
}

func HandlePauseExperiment(em ExperimentManager) http.HandlerFunc {
	return handleLifecycleTransition(em, statusPaused, statusRunning)
}

func HandleCompleteExperiment(em ExperimentManager) http.HandlerFunc {
	return handleLifecycleTransition(em, statusCompleted, statusRunning, statusPaused)
}

func handleLifecycleTransition(em ExperimentManager, target string, allowedFrom ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exp, ok := getExperimentOr404(em, w, r)
		if !ok {
			return
		}

		if !isAllowedTransition(exp.Status, allowedFrom) {
			detail := fmt.Sprintf("cannot transition from %q to %q", exp.Status, target)
			problem.Write(w, problem.Conflict(detail, r.URL.Path))
			return
		}

		exp.Status = target
		updated, err := em.UpdateExperiment(r.Context(), tenantFromContext(r), exp)
		if err != nil {
			problem.Write(w, problem.InternalError(err.Error(), r.URL.Path))
			return
		}

		resp := protoToExperimentResponse(updated)
		w.Header().Set(headerContentType, contentTypeJSON)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func getExperimentOr404(em ExperimentManager, w http.ResponseWriter, r *http.Request) (*pb.Experiment, bool) {
	tenantID := tenantFromContext(r)
	experimentKey := r.PathValue("experimentKey")

	exp, err := em.GetExperiment(r.Context(), tenantID, experimentKey)
	if err != nil {
		problem.Write(w, problem.InternalError(err.Error(), r.URL.Path))
		return nil, false
	}
	if exp == nil {
		problem.Write(w, problem.NotFound("experiment not found", r.URL.Path))
		return nil, false
	}
	return exp, true
}

func isAllowedTransition(current string, allowed []string) bool {
	for _, s := range allowed {
		if current == s {
			return true
		}
	}
	return false
}

func protoToExperimentResponse(e *pb.Experiment) experimentResponse {
	resp := experimentResponse{
		Key:              e.Key,
		Name:             e.Name,
		Status:           e.Status,
		StickyAssignment: e.StickyAssignment,
		Variants:         make([]variantResponse, 0, len(e.Variants)),
		Allocation:       make([]allocationResponse, 0, len(e.Allocation)),
	}

	if e.CreatedAt != 0 {
		t := time.Unix(e.CreatedAt, 0).UTC()
		resp.CreatedAt = &t
	}
	if e.UpdatedAt != 0 {
		t := time.Unix(e.UpdatedAt, 0).UTC()
		resp.UpdatedAt = &t
	}

	for _, v := range e.Variants {
		resp.Variants = append(resp.Variants, variantResponse{
			Key:         v.Key,
			Description: v.Description,
		})
	}

	for _, a := range e.Allocation {
		resp.Allocation = append(resp.Allocation, allocationResponse{
			VariantKey: a.VariantKey,
			Percentage: a.Percentage,
		})
	}

	return resp
}

func experimentRequestToProto(key, name string, sticky bool, variants []variantRequest, allocation []allocationRequest) *pb.Experiment {
	exp := &pb.Experiment{
		Key:              key,
		Name:             name,
		StickyAssignment: sticky,
	}

	for _, v := range variants {
		exp.Variants = append(exp.Variants, &pb.Variant{
			Key:         v.Key,
			Description: v.Description,
		})
	}

	for _, a := range allocation {
		exp.Allocation = append(exp.Allocation, &pb.Allocation{
			VariantKey: a.VariantKey,
			Percentage: a.Percentage,
		})
	}

	return exp
}
