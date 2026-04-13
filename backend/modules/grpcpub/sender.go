package grpcpub

import (
	"context"
	"fmt"
	"io"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"google.golang.org/grpc"
)

type Sender struct {
	client pb.PublisherServiceClient
	conn   io.Closer
}

func NewSender(conn *grpc.ClientConn) *Sender {
	return &Sender{
		client: pb.NewPublisherServiceClient(conn),
		conn:   conn,
	}
}

func (s *Sender) Send(event *pb.Event) error {
	_, err := s.client.Publish(context.Background(), &pb.PublishRequest{Event: event})
	if err != nil {
		return fmt.Errorf("grpc publish: %w", err)
	}
	return nil
}

func (s *Sender) Healthy(ctx context.Context) (bool, string) {
	resp, err := s.client.HealthCheck(ctx, &pb.HealthCheckRequest{})
	if err != nil {
		return false, err.Error()
	}
	return resp.Healthy, resp.Message
}

func (s *Sender) Close() error {
	return s.conn.Close()
}
