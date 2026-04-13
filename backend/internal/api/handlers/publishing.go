package handlers

import (
	"context"
	"log/slog"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/publisher"
)

// PublishingFlagManager decorates a FlagManager with event publishing.
// Mutating operations fire events asynchronously; read operations delegate directly.
type PublishingFlagManager struct {
	inner FlagManager
	pub   publisher.Publisher
}

func NewPublishingFlagManager(inner FlagManager, pub publisher.Publisher) *PublishingFlagManager {
	return &PublishingFlagManager{inner: inner, pub: pub}
}

func (m *PublishingFlagManager) GetFlag(ctx context.Context, tenantID, flagKey string) (*pb.FlagDefinition, error) {
	return m.inner.GetFlag(ctx, tenantID, flagKey)
}

func (m *PublishingFlagManager) ListFlags(ctx context.Context, tenantID string, page, perPage int) ([]*pb.FlagDefinition, int, error) {
	return m.inner.ListFlags(ctx, tenantID, page, perPage)
}

func (m *PublishingFlagManager) CreateFlag(ctx context.Context, tenantID string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error) {
	created, err := m.inner.CreateFlag(ctx, tenantID, flag)
	if err != nil {
		return nil, err
	}
	m.publishAsync(ctx, publisher.EventFlagCreated, tenantID, created)
	return created, nil
}

func (m *PublishingFlagManager) UpdateFlag(ctx context.Context, tenantID string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error) {
	updated, err := m.inner.UpdateFlag(ctx, tenantID, flag)
	if err != nil {
		return nil, err
	}
	m.publishAsync(ctx, publisher.EventFlagUpdated, tenantID, updated)
	return updated, nil
}

func (m *PublishingFlagManager) DeleteFlag(ctx context.Context, tenantID, flagKey string) error {
	if err := m.inner.DeleteFlag(ctx, tenantID, flagKey); err != nil {
		return err
	}
	m.publishAsync(ctx, publisher.EventFlagDeleted, tenantID, map[string]string{"key": flagKey})
	return nil
}

func (m *PublishingFlagManager) publishAsync(ctx context.Context, eventType, tenantID string, payload any) {
	go func() {
		event := publisher.NewEvent(eventType, tenantID, payload)
		if err := m.pub.Publish(context.WithoutCancel(ctx), event); err != nil {
			slog.Warn("failed to publish event", "type", eventType, "error", err)
		}
	}()
}

// PublishingExperimentManager decorates an ExperimentManager with event publishing.
// Lifecycle transitions (start/pause/complete) go through UpdateExperiment, so events
// are published for those transitions automatically.
type PublishingExperimentManager struct {
	inner ExperimentManager
	pub   publisher.Publisher
}

func NewPublishingExperimentManager(inner ExperimentManager, pub publisher.Publisher) *PublishingExperimentManager {
	return &PublishingExperimentManager{inner: inner, pub: pub}
}

func (m *PublishingExperimentManager) GetExperiment(ctx context.Context, tenantID, experimentKey string) (*pb.Experiment, error) {
	return m.inner.GetExperiment(ctx, tenantID, experimentKey)
}

func (m *PublishingExperimentManager) ListExperiments(ctx context.Context, tenantID string, page, perPage int) ([]*pb.Experiment, int, error) {
	return m.inner.ListExperiments(ctx, tenantID, page, perPage)
}

func (m *PublishingExperimentManager) CreateExperiment(ctx context.Context, tenantID string, exp *pb.Experiment) (*pb.Experiment, error) {
	created, err := m.inner.CreateExperiment(ctx, tenantID, exp)
	if err != nil {
		return nil, err
	}
	m.publishAsync(ctx, publisher.EventExperimentCreated, tenantID, created)
	return created, nil
}

func (m *PublishingExperimentManager) UpdateExperiment(ctx context.Context, tenantID string, exp *pb.Experiment) (*pb.Experiment, error) {
	updated, err := m.inner.UpdateExperiment(ctx, tenantID, exp)
	if err != nil {
		return nil, err
	}
	m.publishAsync(ctx, publisher.EventExperimentUpdated, tenantID, updated)
	return updated, nil
}

func (m *PublishingExperimentManager) publishAsync(ctx context.Context, eventType, tenantID string, payload any) {
	go func() {
		event := publisher.NewEvent(eventType, tenantID, payload)
		if err := m.pub.Publish(context.WithoutCancel(ctx), event); err != nil {
			slog.Warn("failed to publish event", "type", eventType, "error", err)
		}
	}()
}
