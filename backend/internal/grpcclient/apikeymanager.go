package grpcclient

import (
	"context"
	"fmt"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

func (c *PersistenceClient) ListAPIKeysManaged(ctx context.Context, tenantID string, page, perPage int) ([]*pb.APIKey, int, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	resp, err := c.client.ListAPIKeys(ctx, &pb.ListAPIKeysRequest{
		Tenant:     &pb.TenantScope{TenantId: tenantID},
		Pagination: &pb.PageRequest{Page: int32(page), PerPage: int32(perPage)},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("grpc ListAPIKeys: %w", err)
	}

	total := 0
	if resp.Pagination != nil {
		total = int(resp.Pagination.Total)
	}
	return resp.ApiKeys, total, nil
}

func (c *PersistenceClient) CreateAPIKeyManaged(ctx context.Context, tenantID string, key *pb.APIKey) (*pb.APIKey, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	resp, err := c.client.CreateAPIKey(ctx, &pb.CreateAPIKeyRequest{
		Tenant: &pb.TenantScope{TenantId: tenantID},
		ApiKey: key,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc CreateAPIKey: %w", err)
	}
	return resp.ApiKey, nil
}

func (c *PersistenceClient) RevokeAPIKeyManaged(ctx context.Context, tenantID, keyID string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err := c.client.RevokeAPIKey(ctx, &pb.RevokeAPIKeyRequest{
		Tenant: &pb.TenantScope{TenantId: tenantID},
		KeyId:  keyID,
	})
	if err != nil {
		return fmt.Errorf("grpc RevokeAPIKey: %w", err)
	}
	return nil
}
