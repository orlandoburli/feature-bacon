package pubserver

import (
	"context"
	"errors"
	"testing"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

const (
	fmtUnexpected     = "unexpected error: %v"
	emptyJSON         = "{}"
	msgConnectionLost = "connection lost"
)

var errMock = errors.New("mock error")

type mockSender struct {
	sendErr    error
	sendCalls  int
	healthy    bool
	healthMsg  string
	closeCalls int
}

func (m *mockSender) Send(_ *pb.Event) error {
	m.sendCalls++
	return m.sendErr
}

func (m *mockSender) Healthy(_ context.Context) (bool, string) {
	return m.healthy, m.healthMsg
}

func (m *mockSender) Close() error {
	m.closeCalls++
	return nil
}

func newTestServer(healthy bool) (*mockSender, *Server) {
	ms := &mockSender{healthy: healthy, healthMsg: HealthyMessage}
	return ms, New(ms)
}

func TestPublish_Success(t *testing.T) {
	ms, srv := newTestServer(true)
	_, err := srv.Publish(context.Background(), &pb.PublishRequest{
		Event: &pb.Event{EventId: "e1", EventType: "flag.created", PayloadJson: emptyJSON},
	})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
	if ms.sendCalls != 1 {
		t.Errorf("sendCalls = %d, want 1", ms.sendCalls)
	}
}

func TestPublish_NilEvent(t *testing.T) {
	ms, srv := newTestServer(true)
	_, err := srv.Publish(context.Background(), &pb.PublishRequest{})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
	if ms.sendCalls != 0 {
		t.Errorf("sendCalls = %d, want 0", ms.sendCalls)
	}
}

func TestPublish_Error(t *testing.T) {
	ms, srv := newTestServer(true)
	ms.sendErr = errMock
	_, err := srv.Publish(context.Background(), &pb.PublishRequest{
		Event: &pb.Event{EventId: "e1", PayloadJson: emptyJSON},
	})
	if !errors.Is(err, errMock) {
		t.Fatalf("got %v, want %v", err, errMock)
	}
}

func TestPublishBatch_Success(t *testing.T) {
	ms, srv := newTestServer(true)
	_, err := srv.PublishBatch(context.Background(), &pb.PublishBatchRequest{
		Events: []*pb.Event{
			{EventId: "e1", PayloadJson: emptyJSON},
			{EventId: "e2", PayloadJson: emptyJSON},
		},
	})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
	if ms.sendCalls != 2 {
		t.Errorf("sendCalls = %d, want 2", ms.sendCalls)
	}
}

func TestPublishBatch_Empty(t *testing.T) {
	ms, srv := newTestServer(true)
	_, err := srv.PublishBatch(context.Background(), &pb.PublishBatchRequest{})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
	if ms.sendCalls != 0 {
		t.Errorf("sendCalls = %d, want 0", ms.sendCalls)
	}
}

func TestPublishBatch_PartialError(t *testing.T) {
	srv := New(&countingSender{failAt: 2})
	_, err := srv.PublishBatch(context.Background(), &pb.PublishBatchRequest{
		Events: []*pb.Event{
			{EventId: "e1", PayloadJson: emptyJSON},
			{EventId: "e2", PayloadJson: emptyJSON},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

type countingSender struct {
	calls  int
	failAt int
}

func (c *countingSender) Send(_ *pb.Event) error {
	c.calls++
	if c.calls >= c.failAt {
		return errMock
	}
	return nil
}

func (c *countingSender) Healthy(_ context.Context) (bool, string) { return true, HealthyMessage }
func (c *countingSender) Close() error                             { return nil }

func TestHealthCheck_Healthy(t *testing.T) {
	_, srv := newTestServer(true)
	resp, err := srv.HealthCheck(context.Background(), &pb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
	if !resp.Healthy {
		t.Error("expected healthy = true")
	}
	if resp.Message != HealthyMessage {
		t.Errorf("message = %q, want %q", resp.Message, HealthyMessage)
	}
}

func TestHealthCheck_Unhealthy(t *testing.T) {
	ms := &mockSender{healthy: false, healthMsg: msgConnectionLost}
	srv := New(ms)
	resp, err := srv.HealthCheck(context.Background(), &pb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
	if resp.Healthy {
		t.Error("expected healthy = false")
	}
	if resp.Message != msgConnectionLost {
		t.Errorf("message = %q, want %q", resp.Message, msgConnectionLost)
	}
}

func TestMarshalEvent(t *testing.T) {
	data, err := MarshalEvent(&pb.Event{EventId: "e1", EventType: "test"})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty data")
	}
}
