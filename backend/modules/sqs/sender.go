package sqs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/pubserver"
)

type sqsAPI interface {
	SendMessage(ctx context.Context, params *awssqs.SendMessageInput, optFns ...func(*awssqs.Options)) (*awssqs.SendMessageOutput, error)
	GetQueueAttributes(ctx context.Context, params *awssqs.GetQueueAttributesInput, optFns ...func(*awssqs.Options)) (*awssqs.GetQueueAttributesOutput, error)
}

type Sender struct {
	client   sqsAPI
	queueURL string
}

func NewSender(client *awssqs.Client, queueURL string) *Sender {
	return &Sender{client: client, queueURL: queueURL}
}

func (s *Sender) Send(event *pb.Event) error {
	data, err := pubserver.MarshalEvent(event)
	if err != nil {
		return err
	}
	_, err = s.client.SendMessage(context.Background(), &awssqs.SendMessageInput{
		QueueUrl:       &s.queueURL,
		MessageBody:    aws.String(string(data)),
		MessageGroupId: aws.String(event.EventType),
	})
	if err != nil {
		return fmt.Errorf("sqs send: %w", err)
	}
	return nil
}

func (s *Sender) Healthy(ctx context.Context) (bool, string) {
	_, err := s.client.GetQueueAttributes(ctx, &awssqs.GetQueueAttributesInput{
		QueueUrl: &s.queueURL,
	})
	if err != nil {
		return false, err.Error()
	}
	return true, pubserver.HealthyMessage
}

func (s *Sender) Close() error { return nil }
