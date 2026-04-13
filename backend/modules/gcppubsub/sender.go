package gcppubsub

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub/v2"
	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/pubserver"
)

type publisher interface {
	publish(ctx context.Context, data []byte) (string, error)
	stop()
}

type gcpPublisher struct {
	pub *pubsub.Publisher
}

func (p *gcpPublisher) publish(ctx context.Context, data []byte) (string, error) {
	return p.pub.Publish(ctx, &pubsub.Message{Data: data}).Get(ctx)
}

func (p *gcpPublisher) stop() {
	p.pub.Stop()
}

type Sender struct {
	pub publisher
}

func NewSender(pub *pubsub.Publisher) *Sender {
	return &Sender{pub: &gcpPublisher{pub: pub}}
}

func (s *Sender) Send(event *pb.Event) error {
	data, err := pubserver.MarshalEvent(event)
	if err != nil {
		return err
	}
	if _, err = s.pub.publish(context.Background(), data); err != nil {
		return fmt.Errorf("pubsub publish: %w", err)
	}
	return nil
}

func (s *Sender) Healthy(_ context.Context) (bool, string) {
	return true, pubserver.HealthyMessage
}

func (s *Sender) Close() error {
	s.pub.stop()
	return nil
}
