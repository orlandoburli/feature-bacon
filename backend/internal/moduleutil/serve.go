package moduleutil

import (
	"context"
	"log/slog"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/orlandoburli/feature-bacon/internal/tlsutil"
)

func TLSConfigFromEnv() tlsutil.Config {
	return tlsutil.Config{
		CAFile:   os.Getenv("BACON_TLS_CA"),
		CertFile: os.Getenv("BACON_TLS_CERT"),
		KeyFile:  os.Getenv("BACON_TLS_KEY"),
	}
}

func ServerOptions(tlsCfg tlsutil.Config) ([]grpc.ServerOption, error) {
	if !tlsCfg.Enabled() {
		return nil, nil
	}
	tc, err := tlsutil.ServerTLSConfig(tlsCfg)
	if err != nil {
		return nil, err
	}
	slog.Info("TLS enabled")
	return []grpc.ServerOption{grpc.Creds(credentials.NewTLS(tc))}, nil
}

type RegisterFunc func(s *grpc.Server)

func NewGRPCServer(register RegisterFunc) (*grpc.Server, error) {
	tlsCfg := TLSConfigFromEnv()
	opts, err := ServerOptions(tlsCfg)
	if err != nil {
		return nil, err
	}
	srv := grpc.NewServer(opts...)
	register(srv)
	return srv, nil
}

func ListenAndServe(ctx context.Context, srv *grpc.Server, addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	go func() {
		slog.Info("gRPC server starting", "addr", addr)
		if err := srv.Serve(lis); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")
	srv.GracefulStop()
	return nil
}

func EnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
