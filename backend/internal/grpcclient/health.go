package grpcclient

import (
	"context"
	"time"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/api/handlers"
)

const healthCheckTimeout = 2 * time.Second

type PersistenceHealthChecker struct {
	client *PersistenceClient
}

func NewPersistenceHealthChecker(c *PersistenceClient) *PersistenceHealthChecker {
	return &PersistenceHealthChecker{client: c}
}

func (h *PersistenceHealthChecker) CheckHealth(ctx context.Context) (string, handlers.ModuleHealth) {
	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	start := time.Now()
	_, err := h.client.client.ListFlags(ctx, &pb.ListFlagsRequest{
		Tenant:     &pb.TenantScope{TenantId: "_health"},
		Pagination: &pb.PageRequest{Page: 1, PerPage: 1},
	})
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return "persistence", handlers.ModuleHealth{
			Status:    "error",
			LatencyMs: latency,
			Message:   err.Error(),
		}
	}
	return "persistence", handlers.ModuleHealth{
		Status:    "ok",
		LatencyMs: latency,
	}
}

type PublisherHealthChecker struct {
	client *PublisherClient
}

func NewPublisherHealthChecker(c *PublisherClient) *PublisherHealthChecker {
	return &PublisherHealthChecker{client: c}
}

func (h *PublisherHealthChecker) CheckHealth(ctx context.Context) (string, handlers.ModuleHealth) {
	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	start := time.Now()
	resp, err := h.client.client.HealthCheck(ctx, &pb.HealthCheckRequest{})
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return "publisher", handlers.ModuleHealth{
			Status:    "error",
			LatencyMs: latency,
			Message:   err.Error(),
		}
	}
	if !resp.Healthy {
		return "publisher", handlers.ModuleHealth{
			Status:    "degraded",
			LatencyMs: latency,
			Message:   resp.Message,
		}
	}
	return "publisher", handlers.ModuleHealth{
		Status:    "ok",
		LatencyMs: latency,
	}
}
