package grpcclient

import (
	"context"
	"net"
	"testing"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	fmtNameWant         = "name = %q, want %q"
	fmtStatusWant       = "status = %q, want %q"
	addrLocalhost       = "127.0.0.1:0"
	fmtListen           = "listen: %v"
	fmtDial             = "dial: %v"
	msgKafkaUnavailable = "kafka unavailable"
)

func TestPersistenceHealthChecker_Healthy(t *testing.T) {
	conn := startMockServer(t)
	client := NewPersistenceClient(conn)
	checker := NewPersistenceHealthChecker(client)

	name, health := checker.CheckHealth(context.Background())

	if name != "persistence" {
		t.Errorf(fmtNameWant, name, "persistence")
	}
	if health.Status != "ok" {
		t.Errorf(fmtStatusWant, health.Status, "ok")
	}
	if health.LatencyMs < 0 {
		t.Errorf("latency = %d, want >= 0", health.LatencyMs)
	}
}

func TestPersistenceHealthChecker_Unhealthy(t *testing.T) {
	lis, err := net.Listen("tcp", addrLocalhost)
	if err != nil {
		t.Fatalf(fmtListen, err)
	}
	addr := lis.Addr().String()
	_ = lis.Close()

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf(fmtDial, err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client := NewPersistenceClient(conn)
	checker := NewPersistenceHealthChecker(client)

	name, health := checker.CheckHealth(context.Background())

	if name != "persistence" {
		t.Errorf(fmtNameWant, name, "persistence")
	}
	if health.Status != "error" {
		t.Errorf(fmtStatusWant, health.Status, "error")
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
		t.Errorf(fmtNameWant, name, "publisher")
	}
	if health.Status != "ok" {
		t.Errorf(fmtStatusWant, health.Status, "ok")
	}
	if health.LatencyMs < 0 {
		t.Errorf("latency = %d, want >= 0", health.LatencyMs)
	}
}

type unhealthyPublisherServer struct {
	pb.UnimplementedPublisherServiceServer
}

func (u *unhealthyPublisherServer) HealthCheck(_ context.Context, _ *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{Healthy: false, Message: msgKafkaUnavailable}, nil
}

func TestPublisherHealthChecker_Degraded(t *testing.T) {
	lis, err := net.Listen("tcp", addrLocalhost)
	if err != nil {
		t.Fatalf(fmtListen, err)
	}

	srv := grpc.NewServer()
	pb.RegisterPublisherServiceServer(srv, &unhealthyPublisherServer{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.GracefulStop)

	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf(fmtDial, err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client := NewPublisherClient(conn)
	checker := NewPublisherHealthChecker(client)

	name, health := checker.CheckHealth(context.Background())

	if name != "publisher" {
		t.Errorf(fmtNameWant, name, "publisher")
	}
	if health.Status != "degraded" {
		t.Errorf(fmtStatusWant, health.Status, "degraded")
	}
	if health.Message != msgKafkaUnavailable {
		t.Errorf("message = %q, want %q", health.Message, msgKafkaUnavailable)
	}
}

func TestPublisherHealthChecker_Unhealthy(t *testing.T) {
	lis, err := net.Listen("tcp", addrLocalhost)
	if err != nil {
		t.Fatalf(fmtListen, err)
	}
	addr := lis.Addr().String()
	_ = lis.Close()

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf(fmtDial, err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client := NewPublisherClient(conn)
	checker := NewPublisherHealthChecker(client)

	name, health := checker.CheckHealth(context.Background())

	if name != "publisher" {
		t.Errorf(fmtNameWant, name, "publisher")
	}
	if health.Status != "error" {
		t.Errorf(fmtStatusWant, health.Status, "error")
	}
	if health.Message == "" {
		t.Error("expected non-empty error message")
	}
}
