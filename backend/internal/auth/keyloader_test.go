package auth

import "testing"

const (
	envTenant     = "_default"
	evalKeyRaw    = "ba_eval_key1"
	mgmtKeyRaw    = "ba_mgmt_key2"
	fmtUnexpErr   = "unexpected error: %v"
	fmtExpectedOK = "expected key to be found"
)

func TestLoadKeysFromEnv_Valid(t *testing.T) {
	store := NewMemKeyStore()
	err := LoadKeysFromEnv(store, evalKeyRaw+":evaluation,"+mgmtKeyRaw+":management", envTenant)
	if err != nil {
		t.Fatalf(fmtUnexpErr, err)
	}

	k1, _ := store.Find(HashKey(evalKeyRaw))
	if k1 == nil {
		t.Fatal(fmtExpectedOK)
	}
	if k1.Scope != ScopeEvaluation {
		t.Errorf("expected evaluation scope, got %s", k1.Scope)
	}
	if k1.TenantID != envTenant {
		t.Errorf("expected tenant %s, got %s", envTenant, k1.TenantID)
	}

	k2, _ := store.Find(HashKey(mgmtKeyRaw))
	if k2 == nil {
		t.Fatal(fmtExpectedOK)
	}
	if k2.Scope != ScopeManagement {
		t.Errorf("expected management scope, got %s", k2.Scope)
	}
}

func TestLoadKeysFromEnv_Empty(t *testing.T) {
	store := NewMemKeyStore()
	if err := LoadKeysFromEnv(store, "", envTenant); err != nil {
		t.Fatalf(fmtUnexpErr, err)
	}
}

func TestLoadKeysFromEnv_InvalidFormat(t *testing.T) {
	store := NewMemKeyStore()
	err := LoadKeysFromEnv(store, "nocolon", envTenant)
	if err == nil {
		t.Fatal("expected error for missing colon")
	}
}

func TestLoadKeysFromEnv_InvalidScope(t *testing.T) {
	store := NewMemKeyStore()
	err := LoadKeysFromEnv(store, "somekey:badscope", envTenant)
	if err == nil {
		t.Fatal("expected error for invalid scope")
	}
}

func TestLoadKeysFromEnv_EmptyKey(t *testing.T) {
	store := NewMemKeyStore()
	err := LoadKeysFromEnv(store, ":evaluation", envTenant)
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestLoadKeysFromConfig_Valid(t *testing.T) {
	store := NewMemKeyStore()
	keys := []ConfigFileKey{
		{Key: evalKeyRaw, Scope: "evaluation", Name: "prod eval"},
		{Key: mgmtKeyRaw, Scope: "management", Name: "admin"},
	}
	err := LoadKeysFromConfig(store, keys, "acme")
	if err != nil {
		t.Fatalf(fmtUnexpErr, err)
	}

	k, _ := store.Find(HashKey(evalKeyRaw))
	if k == nil {
		t.Fatal(fmtExpectedOK)
	}
	if k.TenantID != "acme" {
		t.Errorf("expected tenant acme, got %s", k.TenantID)
	}
	if k.Name != "prod eval" {
		t.Errorf("expected name 'prod eval', got %s", k.Name)
	}
}

func TestLoadKeysFromConfig_EmptyKey(t *testing.T) {
	store := NewMemKeyStore()
	keys := []ConfigFileKey{{Key: "", Scope: "evaluation"}}
	if err := LoadKeysFromConfig(store, keys, "acme"); err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestLoadKeysFromConfig_InvalidScope(t *testing.T) {
	store := NewMemKeyStore()
	keys := []ConfigFileKey{{Key: "somekey", Scope: "admin"}}
	if err := LoadKeysFromConfig(store, keys, "acme"); err == nil {
		t.Fatal("expected error for invalid scope")
	}
}
