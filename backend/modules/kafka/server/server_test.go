package server

import (
	"context"
	"testing"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

const (
	topicTest     = "test-topic"
	fmtUnexpected = "unexpected error: %v"
	emptyJSON     = "{}"
)

func newTestServer(t *testing.T) (*mocks.SyncProducer, *Server) {
	t.Helper()
	mp := mocks.NewSyncProducer(t, nil)
	return mp, New(mp, topicTest)
}

func TestPublish_Success(t *testing.T) {
	mp, srv := newTestServer(t)
	mp.ExpectSendMessageAndSucceed()

	_, err := srv.Publish(context.Background(), &pb.PublishRequest{
		Event: &pb.Event{EventId: "e1", EventType: "flag.created", PayloadJson: emptyJSON},
	})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
}

func TestPublish_NilEvent(t *testing.T) {
	_, srv := newTestServer(t)
	_, err := srv.Publish(context.Background(), &pb.PublishRequest{})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
}

func TestPublishBatch_Success(t *testing.T) {
	mp, srv := newTestServer(t)
	mp.ExpectSendMessageAndSucceed()
	mp.ExpectSendMessageAndSucceed()

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

func TestPublishBatch_Empty(t *testing.T) {
	_, srv := newTestServer(t)
	_, err := srv.PublishBatch(context.Background(), &pb.PublishBatchRequest{})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
}

func TestPublish_Error(t *testing.T) {
	mp, srv := newTestServer(t)
	mp.ExpectSendMessageAndFail(sarama.ErrOutOfBrokers)

	_, err := srv.Publish(context.Background(), &pb.PublishRequest{
		Event: &pb.Event{EventId: "e1", PayloadJson: emptyJSON},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPublishBatch_PartialError(t *testing.T) {
	mp, srv := newTestServer(t)
	mp.ExpectSendMessageAndSucceed()
	mp.ExpectSendMessageAndFail(sarama.ErrOutOfBrokers)

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

func TestHealthCheck(t *testing.T) {
	_, srv := newTestServer(t)
	resp, err := srv.HealthCheck(context.Background(), &pb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf(fmtUnexpected, err)
	}
	if !resp.Healthy {
		t.Error("expected healthy = true")
	}
	if resp.Message != "ok" {
		t.Errorf("message = %q, want %q", resp.Message, "ok")
	}
}
