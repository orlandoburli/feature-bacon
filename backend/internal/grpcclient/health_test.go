package grpcclient

import (
	"context"
	"net"
	"testing"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestPersistenceHealthChecker_Healthy(t *testing.T) {
	conn := startMockServer(t)
	client := NewPersistenceClient(conn)
	checker := NewPersistenceHealthChecker(client)

	name, health := checker.CheckHealth(context.Background())

	if name != "persistence" {
		t.Errorf("name = %q, want %q", name, "persistence")
	}
	if health.Status != "ok" {
		t.Errorf("status = %q, want %q", health.Status, "ok")
	}
	if health.LatencyMs < 0 {
		t.Errorf("latency = %d, want >= 0", health.LatencyMs)
	}
}

func TestPersistenceHealthChecker_Unhealthy(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := lis.Addr().String()
	_ = lis.Close()

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client := NewPersistenceClient(conn)
	checker := NewPersistenceHealthChecker(client)

	name, health := checker.CheckHealth(context.Background())

	if name != "persistence" {
		t.Errorf("name = %q, want %q", name, "persistence")
	}
	if health.Status != "error" {
		t.Errorf("status = %q, want %q", health.Status, "error")
	}
	if health.Message == "" {
		t.Error("expected non-empty error message")
	}
}

func TestPublisherHealthChecker_Healthy(t *testing.T) {
	conn, _ := startMockPublisherServer(t)
	client := NewPublisherClient(conn)
	checker := NewPublisherHealthChecker(client)

	name, health := checker.CheckHealth(context.Background())

	if name != "publisher" {
		t.Errorf("name = %q, want %q", name, "publisher")
	}
	if health.Status != "ok" {
		t.Errorf("status = %q, want %q", health.Status, "ok")
	}
	if health.LatencyMs < 0 {
		t.Errorf("latency = %d, want >= 0", health.LatencyMs)
	}
}

type unhealthyPublisherServer struct {
	pb.UnimplementedPublisherServiceServer
}

func (u *unhealthyPublisherServer) HealthCheck(_ context.Context, _ *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{Healthy: false, Message: "kafka unavailable"}, nil
}

func TestPublisherHealthChecker_Degraded(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv := grpc.NewServer()
	pb.RegisterPublisherServiceServer(srv, &unhealthyPublisherServer{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.GracefulStop)

	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client := NewPublisherClient(conn)
	checker := NewPublisherHealthChecker(client)

	name, health := checker.CheckHealth(context.Background())

	if name != "publisher" {
		t.Errorf("name = %q, want %q", name, "publisher")
	}
	if health.Status != "degraded" {
		t.Errorf("status = %q, want %q", health.Status, "degraded")
	}
	if health.Message != "kafka unavailable" {
		t.Errorf("message = %q, want %q", health.Message, "kafka unavailable")
	}
}

func TestPublisherHealthChecker_Unhealthy(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := lis.Addr().String()
	_ = lis.Close()

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client := NewPublisherClient(conn)
	checker := NewPublisherHealthChecker(client)

	name, health := checker.CheckHealth(context.Background())

	if name != "publisher" {
		t.Errorf("name = %q, want %q", name, "publisher")
	}
	if health.Status != "error" {
		t.Errorf("status = %q, want %q", health.Status, "error")
	}
	if health.Message == "" {
		t.Error("expected non-empty error message")
	}
}
