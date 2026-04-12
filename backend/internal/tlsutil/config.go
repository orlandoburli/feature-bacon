package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// Config holds paths for mTLS configuration.
type Config struct {
	CAFile   string // CA certificate for verifying the peer
	CertFile string // Client/server certificate
	KeyFile  string // Client/server private key
}

// Enabled returns true if all TLS paths are configured.
func (c Config) Enabled() bool {
	return c.CAFile != "" && c.CertFile != "" && c.KeyFile != ""
}

// ClientTLSConfig builds a tls.Config suitable for a gRPC client with mTLS.
func ClientTLSConfig(cfg Config) (*tls.Config, error) {
	caCert, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA cert")
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load client cert/key: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// ServerTLSConfig builds a tls.Config suitable for a gRPC server with mTLS.
func ServerTLSConfig(cfg Config) (*tls.Config, error) {
	caCert, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA cert")
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load server cert/key: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
