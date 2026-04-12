package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

type APIKey struct {
	ID        string
	TenantID  string
	KeyHash   string
	KeyPrefix string
	Scope     Scope
	Name      string
	CreatedAt time.Time
	RevokedAt *time.Time
}

func (k *APIKey) Active() bool {
	return k.RevokedAt == nil
}

// HashKey returns the hex-encoded SHA-256 hash of a raw API key.
func HashKey(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// Prefix returns the first 8 characters of a raw key for identification.
func Prefix(raw string) string {
	if len(raw) <= 8 {
		return raw
	}
	return raw[:8]
}

// KeyFinder provides read access to API keys by hash.
type KeyFinder interface {
	Find(hash string) (*APIKey, error)
}
