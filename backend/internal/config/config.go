package config

import "os"

type Config struct {
	Mode        string
	Persistence string
	ConfigFile  string
	HTTPAddr    string
	LogLevel    string
	LogFormat   string

	AuthEnabled    bool
	APIKeys        string // comma-separated key:scope pairs
	JWTIssuer      string
	JWTAudience    string
	JWTJWKSURL     string
	JWTTenantClaim string
	JWTScopeClaim  string

	PersistenceAddr string // gRPC address for persistence module
	TLSCA           string
	TLSCert         string
	TLSKey          string
}

func Load() Config {
	return Config{
		Mode:        envOrDefault("BACON_MODE", "sidecar"),
		Persistence: envOrDefault("BACON_PERSISTENCE", "file"),
		ConfigFile:  envOrDefault("BACON_CONFIG_FILE", "/etc/bacon/flags.yaml"),
		HTTPAddr:    envOrDefault("BACON_HTTP_ADDR", ":8080"),
		LogLevel:    envOrDefault("BACON_LOG_LEVEL", "info"),
		LogFormat:   envOrDefault("BACON_LOG_FORMAT", "json"),

		AuthEnabled:    envOrDefault("BACON_AUTH_ENABLED", "true") == "true",
		APIKeys:        os.Getenv("BACON_API_KEYS"),
		JWTIssuer:      os.Getenv("BACON_JWT_ISSUER"),
		JWTAudience:    os.Getenv("BACON_JWT_AUDIENCE"),
		JWTJWKSURL:     os.Getenv("BACON_JWT_JWKS_URL"),
		JWTTenantClaim: envOrDefault("BACON_JWT_TENANT_CLAIM", "tenant"),
		JWTScopeClaim:  os.Getenv("BACON_JWT_SCOPE_CLAIM"),

		PersistenceAddr: os.Getenv("BACON_PERSISTENCE_ADDR"),
		TLSCA:           os.Getenv("BACON_TLS_CA"),
		TLSCert:         os.Getenv("BACON_TLS_CERT"),
		TLSKey:          os.Getenv("BACON_TLS_KEY"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
