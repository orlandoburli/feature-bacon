package grpcclient

import (
	"context"
	"fmt"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"google.golang.org/grpc"
)

type PublisherClient struct {
	client pb.PublisherServiceClient
	conn   *grpc.ClientConn
}

func NewPublisherClient(conn *grpc.ClientConn) *PublisherClient {
	return &PublisherClient{
		client: pb.NewPublisherServiceClient(conn),
		conn:   conn,
	}
}

func (c *PublisherClient) Publish(ctx context.Context, event *pb.Event) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	_, err := c.client.Publish(ctx, &pb.PublishRequest{Event: event})
	if err != nil {
		return fmt.Errorf("grpc Publish: %w", err)
	}
	return nil
}

func (c *PublisherClient) Close() error {
	return c.conn.Close()
}
