package gcppubsub

import (
	"context"
	"errors"
	"testing"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/pubserver"
)

const (
	fmtUnexpected = "unexpected error: %v"
	emptyJSON     = "{}"
)

var errMock = errors.New("mock error")

type mockPublisher struct {
	publishErr error
	stopped    bool
}

func (m *mockPublisher) publish(_ context.Context, _ []byte) (string, error) {
	return "msg-id", m.publishErr
}

func (m *mockPublisher) stop() {
	m.stopped = true
}

func newTestSender(mock *mockPublisher) *Sender {
	return &Sender{pub: mock}
}

func TestSend_Success(t *testing.T) {
	sender := newTestSender(&mockPublisher{})
	err := sender.Send(&pb.Event{EventId: "e1", EventType: "flag.created", PayloadJson: emptyJSON})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
}

func TestSend_Error(t *testing.T) {
	sender := newTestSender(&mockPublisher{publishErr: errMock})
	err := sender.Send(&pb.Event{EventId: "e1", EventType: "flag.created", PayloadJson: emptyJSON})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHealthy(t *testing.T) {
	sender := newTestSender(&mockPublisher{})
	ok, msg := sender.Healthy(context.Background())
	if !ok {
		t.Error("expected healthy = true")
	}
	if msg != pubserver.HealthyMessage {
		t.Errorf("message = %q, want %q", msg, pubserver.HealthyMessage)
	}
}

func TestClose(t *testing.T) {
	mock := &mockPublisher{}
	sender := newTestSender(mock)
	if err := sender.Close(); err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
	if !mock.stopped {
		t.Error("expected Stop to be called")
	}
}
