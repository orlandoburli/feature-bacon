package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/IBM/sarama"
	"google.golang.org/grpc"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/moduleutil"
	"github.com/orlandoburli/feature-bacon/modules/kafka/server"
)

func main() {
	brokersStr := os.Getenv("MODULE_KAFKA_BROKERS")
	if brokersStr == "" {
		slog.Error("MODULE_KAFKA_BROKERS is required")
		os.Exit(1)
	}
	brokers := strings.Split(brokersStr, ",")

	topic := moduleutil.EnvOrDefault("MODULE_KAFKA_TOPIC", "bacon-events")
	addr := moduleutil.EnvOrDefault("MODULE_GRPC_ADDR", ":50052")

	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForAll

	producer, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		slog.Error("failed to create Kafka producer", "error", err)
		os.Exit(1)
	}
	defer producer.Close()

	srv := server.New(producer, topic)

	grpcServer, err := moduleutil.NewGRPCServer(func(s *grpc.Server) {
		pb.RegisterPublisherServiceServer(s, srv)
	})
	if err != nil {
		slog.Error("failed to create gRPC server", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := moduleutil.ListenAndServe(ctx, grpcServer, addr); err != nil {
		slog.Error("failed to listen", "error", err, "addr", addr)
		os.Exit(1)
	}
}
