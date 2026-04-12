package auth

import (
	"fmt"
	"strings"
	"time"
)

// LoadKeysFromEnv parses BACON_API_KEYS format: "rawkey:scope,rawkey:scope,..."
// All keys are bound to the given tenantID.
func LoadKeysFromEnv(store *MemKeyStore, raw string, tenantID string) error {
	if raw == "" {
		return nil
	}
	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid API key entry %q: expected key:scope", pair)
		}

		rawKey := strings.TrimSpace(parts[0])
		scope := Scope(strings.TrimSpace(parts[1]))

		if rawKey == "" {
			return fmt.Errorf("empty API key in entry %q", pair)
		}
		if !scope.Valid() {
			return fmt.Errorf("invalid scope %q in entry %q", scope, pair)
		}

		store.Add(&APIKey{
			ID:        fmt.Sprintf("env-%s", Prefix(rawKey)),
			TenantID:  tenantID,
			KeyHash:   HashKey(rawKey),
			KeyPrefix: Prefix(rawKey),
			Scope:     scope,
			Name:      "env-configured",
			CreatedAt: time.Now(),
		})
	}
	return nil
}

// ConfigFileKey represents an API key defined in a config file.
type ConfigFileKey struct {
	Key   string `yaml:"key"`
	Scope string `yaml:"scope"`
	Name  string `yaml:"name"`
}

// LoadKeysFromConfig loads API keys from parsed config file entries.
func LoadKeysFromConfig(store *MemKeyStore, keys []ConfigFileKey, tenantID string) error {
	for _, k := range keys {
		scope := Scope(k.Scope)
		if k.Key == "" {
			return fmt.Errorf("config file API key has empty key for tenant %q", tenantID)
		}
		if !scope.Valid() {
			return fmt.Errorf("invalid scope %q for key %q", k.Scope, Prefix(k.Key))
		}

		store.Add(&APIKey{
			ID:        fmt.Sprintf("cfg-%s", Prefix(k.Key)),
			TenantID:  tenantID,
			KeyHash:   HashKey(k.Key),
			KeyPrefix: Prefix(k.Key),
			Scope:     scope,
			Name:      k.Name,
			CreatedAt: time.Now(),
		})
	}
	return nil
}
