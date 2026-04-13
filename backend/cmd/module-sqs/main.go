package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/config"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"google.golang.org/grpc"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/moduleutil"
	"github.com/orlandoburli/feature-bacon/internal/pubserver"
	sqsmod "github.com/orlandoburli/feature-bacon/modules/sqs"
)

func main() {
	queueURL := os.Getenv("MODULE_SQS_QUEUE_URL")
	if queueURL == "" {
		slog.Error("MODULE_SQS_QUEUE_URL is required")
		os.Exit(1)
	}

	addr := moduleutil.EnvOrDefault("MODULE_GRPC_ADDR", ":50052")

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		slog.Error("failed to load AWS config", "error", err)
		os.Exit(1)
	}

	client := awssqs.NewFromConfig(cfg)
	sender := sqsmod.NewSender(client, queueURL)
	srv := pubserver.New(sender)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := moduleutil.ServeModule(ctx, addr, func(s *grpc.Server) {
		pb.RegisterPublisherServiceServer(s, srv)
	}); err != nil {
		slog.Error("module failed", "error", err)
		os.Exit(1)
	}
}
