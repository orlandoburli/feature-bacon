package tlsutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	fmtUnexpectedErr = "unexpected error: %v"
)

func generateTestCerts(t *testing.T) (caPath, certPath, keyPath string) {
	t.Helper()
	dir := t.TempDir()

	caKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	caDER, _ := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)

	srvKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	srvTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	srvDER, _ := x509.CreateCertificate(rand.Reader, srvTemplate, caTemplate, &srvKey.PublicKey, caKey)

	caPath = filepath.Join(dir, "ca.pem")
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	writePEM(t, caPath, "CERTIFICATE", caDER)
	writePEM(t, certPath, "CERTIFICATE", srvDER)

	keyDER, _ := x509.MarshalECPrivateKey(srvKey)
	writePEM(t, keyPath, "EC PRIVATE KEY", keyDER)

	return caPath, certPath, keyPath
}

func writePEM(t *testing.T, path, blockType string, data []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	_ = pem.Encode(f, &pem.Block{Type: blockType, Bytes: data})
}

func TestConfig_Enabled(t *testing.T) {
	c := Config{}
	if c.Enabled() {
		t.Error("expected disabled with empty paths")
	}
	c = Config{CAFile: "ca.pem", CertFile: "cert.pem", KeyFile: "key.pem"}
	if !c.Enabled() {
		t.Error("expected enabled with all paths set")
	}
}

func TestClientTLSConfig_Valid(t *testing.T) {
	ca, cert, key := generateTestCerts(t)
	cfg := Config{CAFile: ca, CertFile: cert, KeyFile: key}

	tlsCfg, err := ClientTLSConfig(cfg)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if tlsCfg.RootCAs == nil {
		t.Error("expected RootCAs to be set")
	}
	if len(tlsCfg.Certificates) != 1 {
		t.Error("expected one client certificate")
	}
}

func TestServerTLSConfig_Valid(t *testing.T) {
	ca, cert, key := generateTestCerts(t)
	cfg := Config{CAFile: ca, CertFile: cert, KeyFile: key}

	tlsCfg, err := ServerTLSConfig(cfg)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if tlsCfg.ClientCAs == nil {
		t.Error("expected ClientCAs to be set")
	}
	if tlsCfg.ClientAuth != 4 { // tls.RequireAndVerifyClientCert
		t.Errorf("expected RequireAndVerifyClientCert, got %d", tlsCfg.ClientAuth)
	}
}

func TestClientTLSConfig_BadCA(t *testing.T) {
	_, err := ClientTLSConfig(Config{CAFile: "/nonexistent", CertFile: "c", KeyFile: "k"})
	if err == nil {
		t.Fatal("expected error for missing CA")
	}
}

func TestClientTLSConfig_BadCert(t *testing.T) {
	ca, _, _ := generateTestCerts(t)
	_, err := ClientTLSConfig(Config{CAFile: ca, CertFile: "/nonexistent", KeyFile: "/nonexistent"})
	if err == nil {
		t.Fatal("expected error for missing cert")
	}
}

func TestClientTLSConfig_InvalidPEM(t *testing.T) {
	dir := t.TempDir()
	badCA := filepath.Join(dir, "bad-ca.pem")
	_ = os.WriteFile(badCA, []byte("not a cert"), 0644)

	_, cert, key := generateTestCerts(t)
	_, err := ClientTLSConfig(Config{CAFile: badCA, CertFile: cert, KeyFile: key})
	if err == nil {
		t.Fatal("expected error for invalid CA PEM")
	}
}
