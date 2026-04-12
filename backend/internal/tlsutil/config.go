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

// loadCAAndCert loads the CA pool and key pair from the config paths.
func loadCAAndCert(cfg Config) (*x509.CertPool, tls.Certificate, error) {
	caCert, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, tls.Certificate{}, fmt.Errorf("read CA cert: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, tls.Certificate{}, fmt.Errorf("failed to append CA cert")
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, tls.Certificate{}, fmt.Errorf("load cert/key: %w", err)
	}
	return pool, cert, nil
}

// ClientTLSConfig builds a tls.Config suitable for a gRPC client with mTLS.
func ClientTLSConfig(cfg Config) (*tls.Config, error) {
	pool, cert, err := loadCAAndCert(cfg)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// ServerTLSConfig builds a tls.Config suitable for a gRPC server with mTLS.
func ServerTLSConfig(cfg Config) (*tls.Config, error) {
	pool, cert, err := loadCAAndCert(cfg)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
