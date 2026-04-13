package grpcclient

import (
	"context"
	"net"
	"testing"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const eventFlagCreated = "flag.created"

type mockPublisherServer struct {
	pb.UnimplementedPublisherServiceServer
	lastEvent *pb.Event
}

func (m *mockPublisherServer) Publish(_ context.Context, req *pb.PublishRequest) (*pb.PublishResponse, error) {
	m.lastEvent = req.Event
	return &pb.PublishResponse{}, nil
}

func (m *mockPublisherServer) HealthCheck(_ context.Context, _ *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{Healthy: true, Message: "ok"}, nil
}

func startMockPublisherServer(t *testing.T) (*grpc.ClientConn, *mockPublisherServer) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	mock := &mockPublisherServer{}
	srv := grpc.NewServer()
	pb.RegisterPublisherServiceServer(srv, mock)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.GracefulStop)

	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn, mock
}

func TestPublisherClient_Publish(t *testing.T) {
	conn, mock := startMockPublisherServer(t)
	client := NewPublisherClient(conn)

	event := &pb.Event{
		EventId:     "evt-1",
		EventType:   eventFlagCreated,
		TenantId:    "tenant-1",
		Timestamp:   1700000000,
		PayloadJson: `{"key":"dark-mode"}`,
	}
	if err := client.Publish(context.Background(), event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.lastEvent == nil {
		t.Fatal("expected server to receive event")
	}
	if mock.lastEvent.EventId != "evt-1" {
		t.Errorf("EventId = %q, want %q", mock.lastEvent.EventId, "evt-1")
	}
	if mock.lastEvent.EventType != eventFlagCreated {
		t.Errorf("EventType = %q, want %q", mock.lastEvent.EventType, eventFlagCreated)
	}
}

func TestPublisherClient_Close(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	pb.RegisterPublisherServiceServer(srv, &mockPublisherServer{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.GracefulStop)

	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	client := NewPublisherClient(conn)
	if err := client.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
