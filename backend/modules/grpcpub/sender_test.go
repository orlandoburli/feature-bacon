package grpcpub

import (
	"context"
	"errors"
	"testing"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/pubserver"
	"google.golang.org/grpc"
)

const (
	fmtUnexpected = "unexpected error: %v"
	emptyJSON     = "{}"
)

var errMock = errors.New("mock error")

type nopCloser struct{}

func (nopCloser) Close() error { return nil }

type mockPubClient struct {
	publishErr error
	healthResp *pb.HealthCheckResponse
	healthErr  error
}

func (m *mockPubClient) Publish(_ context.Context, _ *pb.PublishRequest, _ ...grpc.CallOption) (*pb.PublishResponse, error) {
	return &pb.PublishResponse{}, m.publishErr
}

func (m *mockPubClient) PublishBatch(_ context.Context, _ *pb.PublishBatchRequest, _ ...grpc.CallOption) (*pb.PublishBatchResponse, error) {
	return &pb.PublishBatchResponse{}, m.publishErr
}

func (m *mockPubClient) HealthCheck(_ context.Context, _ *pb.HealthCheckRequest, _ ...grpc.CallOption) (*pb.HealthCheckResponse, error) {
	return m.healthResp, m.healthErr
}

func newTestSender(mock *mockPubClient) *Sender {
	return &Sender{client: mock, conn: nopCloser{}}
}

func TestSend_Success(t *testing.T) {
	sender := newTestSender(&mockPubClient{})
	err := sender.Send(&pb.Event{EventId: "e1", EventType: "flag.created", PayloadJson: emptyJSON})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
}

func TestSend_Error(t *testing.T) {
	sender := newTestSender(&mockPubClient{publishErr: errMock})
	err := sender.Send(&pb.Event{EventId: "e1", EventType: "flag.created", PayloadJson: emptyJSON})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHealthy_OK(t *testing.T) {
	sender := newTestSender(&mockPubClient{
		healthResp: &pb.HealthCheckResponse{Healthy: true, Message: pubserver.HealthyMessage},
	})
	ok, msg := sender.Healthy(context.Background())
	if !ok {
		t.Error("expected healthy = true")
	}
	if msg != pubserver.HealthyMessage {
		t.Errorf("message = %q, want %q", msg, pubserver.HealthyMessage)
	}
}

func TestHealthy_Unhealthy(t *testing.T) {
	sender := newTestSender(&mockPubClient{
		healthResp: &pb.HealthCheckResponse{Healthy: false, Message: "target down"},
	})
	ok, msg := sender.Healthy(context.Background())
	if ok {
		t.Error("expected healthy = false")
	}
	if msg != "target down" {
		t.Errorf("message = %q, want %q", msg, "target down")
	}
}

func TestHealthy_Error(t *testing.T) {
	sender := newTestSender(&mockPubClient{healthErr: errMock})
	ok, msg := sender.Healthy(context.Background())
	if ok {
		t.Error("expected healthy = false")
	}
	if msg == "" {
		t.Error("expected non-empty message")
	}
}

func TestClose(t *testing.T) {
	sender := newTestSender(&mockPubClient{})
	if err := sender.Close(); err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
}
