package grpcclient

import (
	"context"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

// FlagManagerAdapter adapts PersistenceClient to the handlers.FlagManager interface.
type FlagManagerAdapter struct {
	client *PersistenceClient
}

func NewFlagManagerAdapter(c *PersistenceClient) *FlagManagerAdapter {
	return &FlagManagerAdapter{client: c}
}

func (a *FlagManagerAdapter) GetFlag(ctx context.Context, tenantID, flagKey string) (*pb.FlagDefinition, error) {
	return a.client.GetFlagManaged(ctx, tenantID, flagKey)
}

func (a *FlagManagerAdapter) ListFlags(ctx context.Context, tenantID string, page, perPage int) ([]*pb.FlagDefinition, int, error) {
	return a.client.ListFlagsManaged(ctx, tenantID, page, perPage)
}

func (a *FlagManagerAdapter) CreateFlag(ctx context.Context, tenantID string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error) {
	return a.client.CreateFlagManaged(ctx, tenantID, flag)
}

func (a *FlagManagerAdapter) UpdateFlag(ctx context.Context, tenantID string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error) {
	return a.client.UpdateFlagManaged(ctx, tenantID, flag)
}

func (a *FlagManagerAdapter) DeleteFlag(ctx context.Context, tenantID, flagKey string) error {
	return a.client.DeleteFlagManaged(ctx, tenantID, flagKey)
}

// ExperimentManagerAdapter adapts PersistenceClient to the handlers.ExperimentManager interface.
type ExperimentManagerAdapter struct {
	client *PersistenceClient
}

func NewExperimentManagerAdapter(c *PersistenceClient) *ExperimentManagerAdapter {
	return &ExperimentManagerAdapter{client: c}
}

func (a *ExperimentManagerAdapter) GetExperiment(ctx context.Context, tenantID, experimentKey string) (*pb.Experiment, error) {
	return a.client.GetExperimentManaged(ctx, tenantID, experimentKey)
}

func (a *ExperimentManagerAdapter) ListExperiments(ctx context.Context, tenantID string, page, perPage int) ([]*pb.Experiment, int, error) {
	return a.client.ListExperimentsManaged(ctx, tenantID, page, perPage)
}

func (a *ExperimentManagerAdapter) CreateExperiment(ctx context.Context, tenantID string, exp *pb.Experiment) (*pb.Experiment, error) {
	return a.client.CreateExperimentManaged(ctx, tenantID, exp)
}

func (a *ExperimentManagerAdapter) UpdateExperiment(ctx context.Context, tenantID string, exp *pb.Experiment) (*pb.Experiment, error) {
	return a.client.UpdateExperimentManaged(ctx, tenantID, exp)
}

// APIKeyManagerAdapter adapts PersistenceClient to the handlers.APIKeyManager interface.
type APIKeyManagerAdapter struct {
	client *PersistenceClient
}

func NewAPIKeyManagerAdapter(c *PersistenceClient) *APIKeyManagerAdapter {
	return &APIKeyManagerAdapter{client: c}
}

func (a *APIKeyManagerAdapter) ListAPIKeys(ctx context.Context, tenantID string, page, perPage int) ([]*pb.APIKey, int, error) {
	return a.client.ListAPIKeysManaged(ctx, tenantID, page, perPage)
}

func (a *APIKeyManagerAdapter) CreateAPIKey(ctx context.Context, tenantID string, key *pb.APIKey) (*pb.APIKey, error) {
	return a.client.CreateAPIKeyManaged(ctx, tenantID, key)
}

func (a *APIKeyManagerAdapter) RevokeAPIKey(ctx context.Context, tenantID, keyID string) error {
	return a.client.RevokeAPIKeyManaged(ctx, tenantID, keyID)
}
