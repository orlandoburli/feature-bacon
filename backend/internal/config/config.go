package config

import "os"

type Config struct {
	Mode        string
	Persistence string
	ConfigFile  string
	HTTPAddr    string
	LogLevel    string
	LogFormat   string
}

func Load() Config {
	return Config{
		Mode:        envOrDefault("BACON_MODE", "sidecar"),
		Persistence: envOrDefault("BACON_PERSISTENCE", "file"),
		ConfigFile:  envOrDefault("BACON_CONFIG_FILE", "/etc/bacon/flags.yaml"),
		HTTPAddr:    envOrDefault("BACON_HTTP_ADDR", ":8080"),
		LogLevel:    envOrDefault("BACON_LOG_LEVEL", "info"),
		LogFormat:   envOrDefault("BACON_LOG_FORMAT", "json"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
