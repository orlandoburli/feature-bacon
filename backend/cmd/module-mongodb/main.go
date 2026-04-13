package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"google.golang.org/grpc"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/moduleutil"
	mongostore "github.com/orlandoburli/feature-bacon/modules/mongodb/store"
)

func main() {
	addr := moduleutil.EnvOrDefault("MODULE_GRPC_ADDR", ":50051")
	mongoURI := moduleutil.EnvOrDefault("MODULE_MONGO_URI", "mongodb://localhost:27017")
	mongoDBName := moduleutil.EnvOrDefault("MODULE_MONGO_DB", "feature_bacon")

	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		slog.Error("failed to connect to MongoDB", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			slog.Error("failed to disconnect MongoDB", "error", err)
		}
	}()

	if err := client.Ping(context.Background(), nil); err != nil {
		slog.Error("failed to ping MongoDB", "error", err)
		os.Exit(1)
	}

	db := client.Database(mongoDBName)
	if err := mongostore.EnsureIndexes(context.Background(), db); err != nil {
		slog.Error("failed to create indexes", "error", err)
		os.Exit(1)
	}

	st := mongostore.New(db)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := moduleutil.ServeModule(ctx, addr, func(s *grpc.Server) {
		pb.RegisterPersistenceServiceServer(s, st)
	}); err != nil {
		slog.Error("module failed", "error", err)
		os.Exit(1)
	}
}
