package auth

import "testing"

func TestHashKey(t *testing.T) {
	h1 := HashKey("ba_eval_abc123")
	h2 := HashKey("ba_eval_abc123")
	if h1 != h2 {
		t.Error("same input should produce same hash")
	}

	h3 := HashKey("ba_eval_xyz789")
	if h1 == h3 {
		t.Error("different inputs should produce different hashes")
	}

	if len(h1) != 64 {
		t.Errorf("expected 64-char hex, got %d", len(h1))
	}
}

func TestPrefix(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"ba_eval_abc123456", "ba_eval_"},
		{"short", "short"},
		{"exactly8", "exactly8"},
		{"12345678X", "12345678"},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := Prefix(tt.raw); got != tt.want {
				t.Errorf("Prefix(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestAPIKey_Active(t *testing.T) {
	k := &APIKey{ID: "1", TenantID: "acme", Scope: ScopeEvaluation}
	if !k.Active() {
		t.Error("expected key without RevokedAt to be active")
	}

	now := k.CreatedAt
	k.RevokedAt = &now
	if k.Active() {
		t.Error("expected key with RevokedAt to be inactive")
	}
}
