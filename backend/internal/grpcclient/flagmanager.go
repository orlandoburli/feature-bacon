package grpcclient

import (
	"context"
	"fmt"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

func (c *PersistenceClient) GetFlagManaged(ctx context.Context, tenantID, flagKey string) (*pb.FlagDefinition, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	resp, err := c.client.GetFlag(ctx, &pb.GetFlagRequest{
		Tenant:  &pb.TenantScope{TenantId: tenantID},
		FlagKey: flagKey,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc GetFlag: %w", err)
	}
	return resp.Flag, nil
}

func (c *PersistenceClient) ListFlagsManaged(ctx context.Context, tenantID string, page, perPage int) ([]*pb.FlagDefinition, int, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	resp, err := c.client.ListFlags(ctx, &pb.ListFlagsRequest{
		Tenant:     &pb.TenantScope{TenantId: tenantID},
		Pagination: &pb.PageRequest{Page: int32(page), PerPage: int32(perPage)},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("grpc ListFlags: %w", err)
	}

	total := 0
	if resp.Pagination != nil {
		total = int(resp.Pagination.Total)
	}
	return resp.Flags, total, nil
}

func (c *PersistenceClient) CreateFlagManaged(ctx context.Context, tenantID string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	resp, err := c.client.CreateFlag(ctx, &pb.CreateFlagRequest{
		Tenant: &pb.TenantScope{TenantId: tenantID},
		Flag:   flag,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc CreateFlag: %w", err)
	}
	return resp.Flag, nil
}

func (c *PersistenceClient) UpdateFlagManaged(ctx context.Context, tenantID string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	resp, err := c.client.UpdateFlag(ctx, &pb.UpdateFlagRequest{
		Tenant: &pb.TenantScope{TenantId: tenantID},
		Flag:   flag,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc UpdateFlag: %w", err)
	}
	return resp.Flag, nil
}

func (c *PersistenceClient) DeleteFlagManaged(ctx context.Context, tenantID, flagKey string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err := c.client.DeleteFlag(ctx, &pb.DeleteFlagRequest{
		Tenant:  &pb.TenantScope{TenantId: tenantID},
		FlagKey: flagKey,
	})
	if err != nil {
		return fmt.Errorf("grpc DeleteFlag: %w", err)
	}
	return nil
}
