package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/grpcclient"
	"github.com/orlandoburli/feature-bacon/internal/moduleutil"
	"github.com/orlandoburli/feature-bacon/internal/pubserver"
	"github.com/orlandoburli/feature-bacon/modules/grpcpub"
)

func main() {
	targetAddr := os.Getenv("MODULE_GRPC_TARGET")
	if targetAddr == "" {
		slog.Error("MODULE_GRPC_TARGET is required")
		os.Exit(1)
	}

	addr := moduleutil.EnvOrDefault("MODULE_GRPC_ADDR", ":50052")

	conn, err := grpcclient.Dial(targetAddr, nil)
	if err != nil {
		slog.Error("failed to connect to target", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	sender := grpcpub.NewSender(conn)
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
