package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"cloud.google.com/go/pubsub/v2"
	"google.golang.org/grpc"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/moduleutil"
	ps "github.com/orlandoburli/feature-bacon/internal/pubserver"
	"github.com/orlandoburli/feature-bacon/modules/gcppubsub"
)

func main() {
	projectID := os.Getenv("MODULE_GCP_PROJECT")
	if projectID == "" {
		slog.Error("MODULE_GCP_PROJECT is required")
		os.Exit(1)
	}

	topicID := moduleutil.EnvOrDefault("MODULE_GCP_TOPIC", "bacon-events")
	addr := moduleutil.EnvOrDefault("MODULE_GRPC_ADDR", ":50052")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		slog.Error("failed to create Pub/Sub client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	publisher := client.Publisher(topicID)
	defer publisher.Stop()

	sender := gcppubsub.NewSender(publisher)
	srv := ps.New(sender)

	if err := moduleutil.ServeModule(ctx, addr, func(s *grpc.Server) {
		pb.RegisterPublisherServiceServer(s, srv)
	}); err != nil {
		slog.Error("module failed", "error", err)
		os.Exit(1)
	}
}
