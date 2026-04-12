package auth

import "testing"

func TestScope_Valid(t *testing.T) {
	tests := []struct {
		scope Scope
		want  bool
	}{
		{ScopeEvaluation, true},
		{ScopeManagement, true},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.scope), func(t *testing.T) {
			if got := tt.scope.Valid(); got != tt.want {
				t.Errorf("Scope(%q).Valid() = %v, want %v", tt.scope, got, tt.want)
			}
		})
	}
}
