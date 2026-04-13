package pubserver

import (
	"context"
	"encoding/json"
	"fmt"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

const HealthyMessage = "ok"

type Sender interface {
	Send(event *pb.Event) error
	Healthy(ctx context.Context) (bool, string)
	Close() error
}

type Server struct {
	pb.UnimplementedPublisherServiceServer
	sender Sender
}

func New(sender Sender) *Server {
	return &Server{sender: sender}
}

func (s *Server) Publish(ctx context.Context, req *pb.PublishRequest) (*pb.PublishResponse, error) {
	if req.Event == nil {
		return &pb.PublishResponse{}, nil
	}
	if err := s.sender.Send(req.Event); err != nil {
		return nil, err
	}
	return &pb.PublishResponse{}, nil
}

func (s *Server) PublishBatch(ctx context.Context, req *pb.PublishBatchRequest) (*pb.PublishBatchResponse, error) {
	for _, event := range req.Events {
		if err := s.sender.Send(event); err != nil {
			return nil, err
		}
	}
	return &pb.PublishBatchResponse{}, nil
}

func (s *Server) HealthCheck(ctx context.Context, _ *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	healthy, msg := s.sender.Healthy(ctx)
	return &pb.HealthCheckResponse{Healthy: healthy, Message: msg}, nil
}

func MarshalEvent(event *pb.Event) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal event: %w", err)
	}
	return data, nil
}
