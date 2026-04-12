package auth

import (
	"testing"
	"time"
)

const (
	rawValidKey   = "ba_eval_valid123"
	rawRevokedKey = "ba_eval_revoked456"
	rawUnknownKey = "ba_eval_unknown789"
)

func seedStore() *MemKeyStore {
	store := NewMemKeyStore()
	store.Add(&APIKey{
		ID:       "k1",
		TenantID: "acme",
		KeyHash:  HashKey(rawValidKey),
		Scope:    ScopeEvaluation,
		Name:     "valid key",
	})

	revoked := time.Now()
	store.Add(&APIKey{
		ID:        "k2",
		TenantID:  "acme",
		KeyHash:   HashKey(rawRevokedKey),
		Scope:     ScopeEvaluation,
		Name:      "revoked key",
		RevokedAt: &revoked,
	})
	return store
}

func TestAuthenticateAPIKey_Valid(t *testing.T) {
	store := seedStore()

	key, err := AuthenticateAPIKey(store, rawValidKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.TenantID != "acme" {
		t.Errorf("expected tenant acme, got %s", key.TenantID)
	}
	if key.Scope != ScopeEvaluation {
		t.Errorf("expected scope evaluation, got %s", key.Scope)
	}
}

func TestAuthenticateAPIKey_Revoked(t *testing.T) {
	store := seedStore()

	_, err := AuthenticateAPIKey(store, rawRevokedKey)
	if err == nil {
		t.Fatal("expected error for revoked key")
	}
}

func TestAuthenticateAPIKey_Unknown(t *testing.T) {
	store := seedStore()

	_, err := AuthenticateAPIKey(store, rawUnknownKey)
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestAuthenticateAPIKey_NilStore(t *testing.T) {
	_, err := AuthenticateAPIKey(nil, rawValidKey)
	if err == nil {
		t.Fatal("expected error for nil store")
	}
}
