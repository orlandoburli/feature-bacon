package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib" // Register pgx as database/sql driver
	"github.com/pressly/goose/v3"
	"google.golang.org/grpc"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/moduleutil"
	"github.com/orlandoburli/feature-bacon/modules/postgres/migrations"
	"github.com/orlandoburli/feature-bacon/modules/postgres/store"
)

func main() {
	dsn := os.Getenv("MODULE_POSTGRES_DSN")
	if dsn == "" {
		slog.Error("MODULE_POSTGRES_DSN is required")
		os.Exit(1)
	}

	addr := moduleutil.EnvOrDefault("MODULE_GRPC_ADDR", ":50051")

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("failed to set goose dialect", "error", err)
		os.Exit(1)
	}
	if err := goose.Up(db, "."); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	st := store.New(db)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := moduleutil.ServeModule(ctx, addr, func(s *grpc.Server) {
		pb.RegisterPersistenceServiceServer(s, st)
	}); err != nil {
		slog.Error("module failed", "error", err)
		os.Exit(1)
	}
}
