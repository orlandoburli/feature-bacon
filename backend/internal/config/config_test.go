package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{
		"BACON_MODE", "BACON_PERSISTENCE", "BACON_CONFIG_FILE",
		"BACON_HTTP_ADDR", "BACON_LOG_LEVEL", "BACON_LOG_FORMAT",
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
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.field, c.got, c.want)
		}
	}
}

func TestLoadFromEnv(t *testing.T) {
	envs := map[string]string{
		"BACON_MODE":        "multi-tenant",
		"BACON_PERSISTENCE": "grpc",
		"BACON_CONFIG_FILE": "/tmp/flags.yaml",
		"BACON_HTTP_ADDR":   ":9090",
		"BACON_LOG_LEVEL":   "debug",
		"BACON_LOG_FORMAT":  "text",
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
		{"Mode", cfg.Mode, "multi-tenant"},
		{"Persistence", cfg.Persistence, "grpc"},
		{"ConfigFile", cfg.ConfigFile, "/tmp/flags.yaml"},
		{"HTTPAddr", cfg.HTTPAddr, ":9090"},
		{"LogLevel", cfg.LogLevel, "debug"},
		{"LogFormat", cfg.LogFormat, "text"},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.field, c.got, c.want)
		}
	}
}

func TestLoadPartialOverride(t *testing.T) {
	for _, key := range []string{
		"BACON_MODE", "BACON_PERSISTENCE", "BACON_CONFIG_FILE",
		"BACON_HTTP_ADDR", "BACON_LOG_LEVEL", "BACON_LOG_FORMAT",
	} {
		os.Unsetenv(key)
	}

	t.Setenv("BACON_MODE", "multi-tenant")
	t.Setenv("BACON_LOG_LEVEL", "warn")

	cfg := Load()

	if cfg.Mode != "multi-tenant" {
		t.Errorf("Mode = %q, want %q", cfg.Mode, "multi-tenant")
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
