package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/internal/tlsutil"
	"github.com/orlandoburli/feature-bacon/modules/postgres/migrations"
	"github.com/orlandoburli/feature-bacon/modules/postgres/server"
	"github.com/orlandoburli/feature-bacon/modules/postgres/store"
)

func main() {
	dsn := os.Getenv("MODULE_POSTGRES_DSN")
	if dsn == "" {
		slog.Error("MODULE_POSTGRES_DSN is required")
		os.Exit(1)
	}

	addr := os.Getenv("MODULE_GRPC_ADDR")
	if addr == "" {
		addr = ":50051"
	}

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
	srv := server.New(st)

	var opts []grpc.ServerOption
	tlsCfg := tlsutil.Config{
		CAFile:   os.Getenv("BACON_TLS_CA"),
		CertFile: os.Getenv("BACON_TLS_CERT"),
		KeyFile:  os.Getenv("BACON_TLS_KEY"),
	}
	if tlsCfg.Enabled() {
		tc, err := tlsutil.ServerTLSConfig(tlsCfg)
		if err != nil {
			slog.Error("failed to load TLS config", "error", err)
			os.Exit(1)
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tc)))
		slog.Info("TLS enabled")
	}

	grpcServer := grpc.NewServer(opts...)
	pb.RegisterPersistenceServiceServer(grpcServer, srv)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("failed to listen", "error", err, "addr", addr)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("gRPC server starting", "addr", addr)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")
	grpcServer.GracefulStop()
}
