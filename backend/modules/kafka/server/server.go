package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IBM/sarama"
	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

type Server struct {
	pb.UnimplementedPublisherServiceServer
	producer sarama.SyncProducer
	topic    string
}

func New(producer sarama.SyncProducer, topic string) *Server {
	return &Server{producer: producer, topic: topic}
}

func (s *Server) Publish(ctx context.Context, req *pb.PublishRequest) (*pb.PublishResponse, error) {
	if req.Event == nil {
		return &pb.PublishResponse{}, nil
	}
	if err := s.send(req.Event); err != nil {
		return nil, err
	}
	return &pb.PublishResponse{}, nil
}

func (s *Server) PublishBatch(ctx context.Context, req *pb.PublishBatchRequest) (*pb.PublishBatchResponse, error) {
	for _, event := range req.Events {
		if err := s.send(event); err != nil {
			return nil, err
		}
	}
	return &pb.PublishBatchResponse{}, nil
}

func (s *Server) HealthCheck(ctx context.Context, _ *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{Healthy: true, Message: "ok"}, nil
}

func (s *Server) send(event *pb.Event) error {
	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
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
