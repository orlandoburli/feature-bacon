package auth

import "testing"

func TestMemKeyStore_AddAndLookup(t *testing.T) {
	store := NewMemKeyStore()

	raw := "ba_eval_testkey123"
	key := &APIKey{
		ID:       "k1",
		TenantID: "acme",
		KeyHash:  HashKey(raw),
		Scope:    ScopeEvaluation,
	}
	store.Add(key)

	found, err := store.Find(HashKey(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected key to be found")
	}
	if found.ID != "k1" {
		t.Errorf("expected ID k1, got %s", found.ID)
	}
}

func TestMemKeyStore_LookupNotFound(t *testing.T) {
	store := NewMemKeyStore()

	found, err := store.Find(HashKey("nonexistent"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Error("expected nil for missing key")
	}
}
