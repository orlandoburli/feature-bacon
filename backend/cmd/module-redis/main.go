package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/moduleutil"
	redisstore "github.com/orlandoburli/feature-bacon/modules/redis/store"
)

func main() {
	addr := moduleutil.EnvOrDefault("MODULE_GRPC_ADDR", ":50051")
	redisAddr := moduleutil.EnvOrDefault("MODULE_REDIS_ADDR", "localhost:6379")
	redisPassword := os.Getenv("MODULE_REDIS_PASSWORD")

	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
	})
	defer client.Close()

	if err := client.Ping(context.Background()).Err(); err != nil {
		slog.Error("failed to ping Redis", "error", err)
		os.Exit(1)
	}

	st := redisstore.New(client)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := moduleutil.ServeModule(ctx, addr, func(s *grpc.Server) {
		pb.RegisterPersistenceServiceServer(s, st)
	}); err != nil {
		slog.Error("module failed", "error", err)
		os.Exit(1)
	}
}
