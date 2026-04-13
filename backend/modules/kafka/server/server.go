package server

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/pubserver"
)

type kafkaSender struct {
	producer sarama.SyncProducer
	topic    string
}

func New(producer sarama.SyncProducer, topic string) *pubserver.Server {
	return pubserver.New(&kafkaSender{producer: producer, topic: topic})
}

func (s *kafkaSender) Send(event *pb.Event) error {
	value, err := pubserver.MarshalEvent(event)
	if err != nil {
		return err
	}
	msg := &sarama.ProducerMessage{
		Topic: s.topic,
		Key:   sarama.StringEncoder(event.EventId),
		Value: sarama.ByteEncoder(value),
	}
	_, _, err = s.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("kafka send: %w", err)
	}
	return nil
}

func (s *kafkaSender) Healthy(_ context.Context) (bool, string) {
	return true, pubserver.HealthyMessage
}

func (s *kafkaSender) Close() error {
	return s.producer.Close()
}
