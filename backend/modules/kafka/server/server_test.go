package server

import (
	"context"
	"testing"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/pubserver"
)

const (
	topicTest     = "test-topic"
	fmtUnexpected = "unexpected error: %v"
	emptyJSON     = "{}"
)

func newTestSender(t *testing.T) (*mocks.SyncProducer, *kafkaSender) {
	t.Helper()
	mp := mocks.NewSyncProducer(t, nil)
	return mp, &kafkaSender{producer: mp, topic: topicTest}
}

func TestSend_Success(t *testing.T) {
	mp, sender := newTestSender(t)
	mp.ExpectSendMessageAndSucceed()

	err := sender.Send(&pb.Event{EventId: "e1", EventType: "flag.created", PayloadJson: emptyJSON})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
}

func TestSend_ProducerError(t *testing.T) {
	mp, sender := newTestSender(t)
	mp.ExpectSendMessageAndFail(sarama.ErrOutOfBrokers)

	err := sender.Send(&pb.Event{EventId: "e1", PayloadJson: emptyJSON})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHealthy(t *testing.T) {
	_, sender := newTestSender(t)
	ok, msg := sender.Healthy(context.Background())
	if !ok {
		t.Error("expected healthy = true")
	}
	if msg != pubserver.HealthyMessage {
		t.Errorf("message = %q, want %q", msg, pubserver.HealthyMessage)
	}
}

func TestNew_ReturnsServer(t *testing.T) {
	mp := mocks.NewSyncProducer(t, nil)
	mp.ExpectSendMessageAndSucceed()

	srv := New(mp, topicTest)
	_, err := srv.Publish(context.Background(), &pb.PublishRequest{
		Event: &pb.Event{EventId: "e1", EventType: "flag.created", PayloadJson: emptyJSON},
	})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
}

func TestNew_NilEvent(t *testing.T) {
	mp := mocks.NewSyncProducer(t, nil)
	srv := New(mp, topicTest)
	_, err := srv.Publish(context.Background(), &pb.PublishRequest{})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
}

func TestNew_BatchSuccess(t *testing.T) {
	mp := mocks.NewSyncProducer(t, nil)
	mp.ExpectSendMessageAndSucceed()
	mp.ExpectSendMessageAndSucceed()

	srv := New(mp, topicTest)
	_, err := srv.PublishBatch(context.Background(), &pb.PublishBatchRequest{
		Events: []*pb.Event{
			{EventId: "e1", PayloadJson: emptyJSON},
			{EventId: "e2", PayloadJson: emptyJSON},
		},
	})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
}

func TestNew_BatchPartialError(t *testing.T) {
	mp := mocks.NewSyncProducer(t, nil)
	mp.ExpectSendMessageAndSucceed()
	mp.ExpectSendMessageAndFail(sarama.ErrOutOfBrokers)

	srv := New(mp, topicTest)
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

func TestNew_HealthCheck(t *testing.T) {
	mp := mocks.NewSyncProducer(t, nil)
	srv := New(mp, topicTest)
	resp, err := srv.HealthCheck(context.Background(), &pb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
	if !resp.Healthy {
		t.Error("expected healthy = true")
	}
	if resp.Message != pubserver.HealthyMessage {
		t.Errorf("message = %q, want %q", resp.Message, pubserver.HealthyMessage)
	}
}
