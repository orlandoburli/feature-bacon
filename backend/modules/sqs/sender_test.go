package sqs

import (
	"context"
	"errors"
	"testing"

	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/pubserver"
)

const (
	fmtUnexpected = "unexpected error: %v"
	testQueueURL  = "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue"
	emptyJSON     = "{}"
)

var errMock = errors.New("mock error")

type mockSQSAPI struct {
	sendErr    error
	getAttrErr error
}

func (m *mockSQSAPI) SendMessage(_ context.Context, _ *awssqs.SendMessageInput, _ ...func(*awssqs.Options)) (*awssqs.SendMessageOutput, error) {
	return &awssqs.SendMessageOutput{}, m.sendErr
}

func (m *mockSQSAPI) GetQueueAttributes(_ context.Context, _ *awssqs.GetQueueAttributesInput, _ ...func(*awssqs.Options)) (*awssqs.GetQueueAttributesOutput, error) {
	return &awssqs.GetQueueAttributesOutput{}, m.getAttrErr
}

func newTestSender(mock *mockSQSAPI) *Sender {
	return &Sender{client: mock, queueURL: testQueueURL}
}

func TestSend_Success(t *testing.T) {
	sender := newTestSender(&mockSQSAPI{})
	err := sender.Send(&pb.Event{EventId: "e1", EventType: "flag.created", PayloadJson: emptyJSON})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
}

func TestSend_Error(t *testing.T) {
	sender := newTestSender(&mockSQSAPI{sendErr: errMock})
	err := sender.Send(&pb.Event{EventId: "e1", EventType: "flag.created", PayloadJson: emptyJSON})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHealthy_OK(t *testing.T) {
	sender := newTestSender(&mockSQSAPI{})
	ok, msg := sender.Healthy(context.Background())
	if !ok {
		t.Error("expected healthy = true")
	}
	if msg != pubserver.HealthyMessage {
		t.Errorf("message = %q, want %q", msg, pubserver.HealthyMessage)
	}
}

func TestHealthy_Error(t *testing.T) {
	sender := newTestSender(&mockSQSAPI{getAttrErr: errMock})
	ok, msg := sender.Healthy(context.Background())
	if ok {
		t.Error("expected healthy = false")
	}
	if msg == "" {
		t.Error("expected non-empty message")
	}
}

func TestClose(t *testing.T) {
	sender := newTestSender(&mockSQSAPI{})
	if err := sender.Close(); err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
}
