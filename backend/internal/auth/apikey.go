package auth

import "fmt"

// AuthenticateAPIKey hashes the raw key, looks it up, and validates it is active.
// Returns the matched APIKey or an error describing the failure.
func AuthenticateAPIKey(store KeyStore, rawKey string) (*APIKey, error) {
	if store == nil {
		return nil, fmt.Errorf("no API key store configured")
	}

	hash := HashKey(rawKey)
	key, err := store.LookupByHash(hash)
	if err != nil {
		return nil, fmt.Errorf("key lookup failed: %w", err)
	}
	if key == nil {
		return nil, fmt.Errorf("invalid API key")
	}
	if !key.Active() {
		return nil, fmt.Errorf("API key has been revoked")
	}
	return key, nil
}
