package grpcclient

import (
	"context"
	"fmt"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

func (c *PersistenceClient) GetExperimentManaged(ctx context.Context, tenantID, experimentKey string) (*pb.Experiment, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	resp, err := c.client.GetExperiment(ctx, &pb.GetExperimentRequest{
		Tenant:        &pb.TenantScope{TenantId: tenantID},
		ExperimentKey: experimentKey,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc GetExperiment: %w", err)
	}
	return resp.Experiment, nil
}

func (c *PersistenceClient) ListExperimentsManaged(ctx context.Context, tenantID string, page, perPage int) ([]*pb.Experiment, int, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	resp, err := c.client.ListExperiments(ctx, &pb.ListExperimentsRequest{
		Tenant:     &pb.TenantScope{TenantId: tenantID},
		Pagination: &pb.PageRequest{Page: int32(page), PerPage: int32(perPage)},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("grpc ListExperiments: %w", err)
	}

	total := 0
	if resp.Pagination != nil {
		total = int(resp.Pagination.Total)
	}
	return resp.Experiments, total, nil
}

func (c *PersistenceClient) CreateExperimentManaged(ctx context.Context, tenantID string, exp *pb.Experiment) (*pb.Experiment, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	resp, err := c.client.CreateExperiment(ctx, &pb.CreateExperimentRequest{
		Tenant:     &pb.TenantScope{TenantId: tenantID},
		Experiment: exp,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc CreateExperiment: %w", err)
	}
	return resp.Experiment, nil
}

func (c *PersistenceClient) UpdateExperimentManaged(ctx context.Context, tenantID string, exp *pb.Experiment) (*pb.Experiment, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	resp, err := c.client.UpdateExperiment(ctx, &pb.UpdateExperimentRequest{
		Tenant:     &pb.TenantScope{TenantId: tenantID},
		Experiment: exp,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc UpdateExperiment: %w", err)
	}
	return resp.Experiment, nil
}
