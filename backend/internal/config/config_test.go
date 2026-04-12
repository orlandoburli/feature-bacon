package config

import (
	"os"
	"testing"
)

const modeMultiTenant = "multi-tenant"

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{
		"BACON_MODE", "BACON_PERSISTENCE", "BACON_CONFIG_FILE",
		"BACON_HTTP_ADDR", "BACON_LOG_LEVEL", "BACON_LOG_FORMAT",
		"BACON_AUTH_ENABLED", "BACON_API_KEYS",
		"BACON_JWT_ISSUER", "BACON_JWT_AUDIENCE", "BACON_JWT_JWKS_URL",
		"BACON_JWT_TENANT_CLAIM", "BACON_JWT_SCOPE_CLAIM",
	} {
		os.Unsetenv(key)
	}

	cfg := Load()

	checks := []struct {
		field string
		got   string
		want  string
	}{
		{"Mode", cfg.Mode, "sidecar"},
		{"Persistence", cfg.Persistence, "file"},
		{"ConfigFile", cfg.ConfigFile, "/etc/bacon/flags.yaml"},
		{"HTTPAddr", cfg.HTTPAddr, ":8080"},
		{"LogLevel", cfg.LogLevel, "info"},
		{"LogFormat", cfg.LogFormat, "json"},
		{"JWTTenantClaim", cfg.JWTTenantClaim, "tenant"},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.field, c.got, c.want)
		}
	}

	if !cfg.AuthEnabled {
		t.Error("expected AuthEnabled to default to true")
	}
	if cfg.APIKeys != "" {
		t.Errorf("expected APIKeys to default to empty, got %q", cfg.APIKeys)
	}
}

func TestLoadFromEnv(t *testing.T) {
	envs := map[string]string{
		"BACON_MODE":             modeMultiTenant,
		"BACON_PERSISTENCE":      "grpc",
		"BACON_CONFIG_FILE":      "/tmp/flags.yaml",
		"BACON_HTTP_ADDR":        ":9090",
		"BACON_LOG_LEVEL":        "debug",
		"BACON_LOG_FORMAT":       "text",
		"BACON_AUTH_ENABLED":     "false",
		"BACON_API_KEYS":         "ba_eval_x:evaluation",
		"BACON_JWT_ISSUER":       "https://auth.test.com",
		"BACON_JWT_AUDIENCE":     "bacon-api",
		"BACON_JWT_JWKS_URL":     "https://auth.test.com/.well-known/jwks.json",
		"BACON_JWT_TENANT_CLAIM": "org_id",
		"BACON_JWT_SCOPE_CLAIM":  "scope",
	}

	for k, v := range envs {
		t.Setenv(k, v)
	}

	cfg := Load()

	checks := []struct {
		field string
		got   string
		want  string
	}{
		{"Mode", cfg.Mode, modeMultiTenant},
		{"Persistence", cfg.Persistence, "grpc"},
		{"ConfigFile", cfg.ConfigFile, "/tmp/flags.yaml"},
		{"HTTPAddr", cfg.HTTPAddr, ":9090"},
		{"LogLevel", cfg.LogLevel, "debug"},
		{"LogFormat", cfg.LogFormat, "text"},
		{"APIKeys", cfg.APIKeys, "ba_eval_x:evaluation"},
		{"JWTIssuer", cfg.JWTIssuer, "https://auth.test.com"},
		{"JWTAudience", cfg.JWTAudience, "bacon-api"},
		{"JWTJWKSURL", cfg.JWTJWKSURL, "https://auth.test.com/.well-known/jwks.json"},
		{"JWTTenantClaim", cfg.JWTTenantClaim, "org_id"},
		{"JWTScopeClaim", cfg.JWTScopeClaim, "scope"},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.field, c.got, c.want)
		}
	}

	if cfg.AuthEnabled {
		t.Error("expected AuthEnabled to be false")
	}
}

func TestLoadPartialOverride(t *testing.T) {
	for _, key := range []string{
		"BACON_MODE", "BACON_PERSISTENCE", "BACON_CONFIG_FILE",
		"BACON_HTTP_ADDR", "BACON_LOG_LEVEL", "BACON_LOG_FORMAT",
	} {
		os.Unsetenv(key)
	}

	t.Setenv("BACON_MODE", modeMultiTenant)
	t.Setenv("BACON_LOG_LEVEL", "warn")

	cfg := Load()

	if cfg.Mode != modeMultiTenant {
		t.Errorf("Mode = %q, want %q", cfg.Mode, modeMultiTenant)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "warn")
	}
	if cfg.Persistence != "file" {
		t.Errorf("Persistence = %q, want default %q", cfg.Persistence, "file")
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want default %q", cfg.HTTPAddr, ":8080")
	}
}

func TestEnvOrDefault(t *testing.T) {
	const key = "BACON_TEST_ENVORD"
	os.Unsetenv(key)

	if got := envOrDefault(key, "fallback"); got != "fallback" {
		t.Errorf("envOrDefault unset = %q, want %q", got, "fallback")
	}

	t.Setenv(key, "custom")
	if got := envOrDefault(key, "fallback"); got != "custom" {
		t.Errorf("envOrDefault set = %q, want %q", got, "custom")
	}
}
