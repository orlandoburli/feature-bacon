package moduleutil

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/orlandoburli/feature-bacon/internal/tlsutil"
)

const fmtUnexpectedErr = "unexpected error: %v"

func TestEnvOrDefault_Fallback(t *testing.T) {
	v := EnvOrDefault("BACON_TEST_NONEXISTENT_KEY_1234", "fallback")
	if v != "fallback" {
		t.Errorf("got %q, want %q", v, "fallback")
	}
}

func TestEnvOrDefault_Set(t *testing.T) {
	t.Setenv("BACON_TEST_MODULEUTIL_KEY", "custom")
	v := EnvOrDefault("BACON_TEST_MODULEUTIL_KEY", "fallback")
	if v != "custom" {
		t.Errorf("got %q, want %q", v, "custom")
	}
}

func TestTLSConfigFromEnv_Empty(t *testing.T) {
	cfg := TLSConfigFromEnv()
	if cfg.Enabled() {
		t.Error("expected TLS disabled when env vars are empty")
	}
}

func TestTLSConfigFromEnv_Set(t *testing.T) {
	t.Setenv("BACON_TLS_CA", "/ca.pem")
	t.Setenv("BACON_TLS_CERT", "/cert.pem")
	t.Setenv("BACON_TLS_KEY", "/key.pem")

	cfg := TLSConfigFromEnv()
	if !cfg.Enabled() {
		t.Error("expected TLS enabled when env vars are set")
	}
}

func TestServerOptions_Disabled(t *testing.T) {
	opts, err := ServerOptions(tlsutil.Config{})
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if opts != nil {
		t.Error("expected nil options when TLS is disabled")
	}
}

func TestServerOptions_InvalidCert(t *testing.T) {
	cfg := tlsutil.Config{
		CAFile:   "/nonexistent",
		CertFile: "/nonexistent",
		KeyFile:  "/nonexistent",
	}
	_, err := ServerOptions(cfg)
	if err == nil {
		t.Error("expected error with invalid cert paths")
	}
}

func TestNewGRPCServer(t *testing.T) {
	registered := false
	srv, err := NewGRPCServer(func(s *grpc.Server) {
		registered = true
	})
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if !registered {
		t.Error("register func was not called")
	}
	srv.Stop()
}

func TestListenAndServe(t *testing.T) {
	srv := grpc.NewServer()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := ListenAndServe(ctx, srv, "127.0.0.1:0")
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
}

func TestListenAndServe_InvalidAddr(t *testing.T) {
	srv := grpc.NewServer()
	defer srv.Stop()

	ctx := context.Background()
	err := ListenAndServe(ctx, srv, "invalid-addr:99999999")
	if err == nil {
		t.Error("expected error with invalid address")
	}
}
