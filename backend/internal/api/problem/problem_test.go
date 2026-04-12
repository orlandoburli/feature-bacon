package problem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFactoryFunctions(t *testing.T) {
	tests := []struct {
		name      string
		problem   *Problem
		wantType  string
		wantCode  int
		wantTitle string
	}{
		{"Unauthorized", Unauthorized("bad token", "/api/v1/evaluate"), "https://bacon.dev/problems/unauthorized", 401, "Unauthorized"},
		{"Forbidden", Forbidden("no access", "/api/v1/flags"), "https://bacon.dev/problems/forbidden", 403, "Forbidden"},
		{"NotFound", NotFound("flag missing", "/api/v1/flags/x"), "https://bacon.dev/problems/not-found", 404, "Not Found"},
		{"Conflict", Conflict("already exists", "/api/v1/flags"), "https://bacon.dev/problems/conflict", 409, "Conflict"},
		{"ValidationError", ValidationError("flagKey required", "/api/v1/evaluate"), "https://bacon.dev/problems/validation-error", 422, "Validation Error"},
		{"ReadOnlyMode", ReadOnlyMode("/api/v1/flags"), "/problems/read-only-mode", 409, "Read-Only Mode"},
		{"InternalError", InternalError("unexpected", "/api/v1/evaluate"), "https://bacon.dev/problems/internal-error", 500, "Internal Server Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.problem.Type != tt.wantType {
				t.Errorf("type = %q, want %q", tt.problem.Type, tt.wantType)
			}
			if tt.problem.Status != tt.wantCode {
				t.Errorf("status = %d, want %d", tt.problem.Status, tt.wantCode)
			}
			if tt.problem.Title != tt.wantTitle {
				t.Errorf("title = %q, want %q", tt.problem.Title, tt.wantTitle)
			}
		})
	}
}

func TestProblemError(t *testing.T) {
	p := NotFound("flag not found", "/test")
	if p.Error() != "flag not found" {
		t.Errorf("Error() = %q, want %q", p.Error(), "flag not found")
	}

	p2 := ReadOnlyMode("/test")
	if p2.Error() != "Read-Only Mode" {
		t.Errorf("Error() = %q, want %q", p2.Error(), "Read-Only Mode")
	}
}

func TestWrite(t *testing.T) {
	w := httptest.NewRecorder()
	p := ValidationError("flagKey is required", "/api/v1/evaluate")

	Write(w, p)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/problem+json")
	}

	var got Problem
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Type != p.Type {
		t.Errorf("body type = %q, want %q", got.Type, p.Type)
	}
	if got.Detail != "flagKey is required" {
		t.Errorf("body detail = %q, want %q", got.Detail, "flagKey is required")
	}
}

func TestWriteOmitsEmptyFields(t *testing.T) {
	w := httptest.NewRecorder()
	p := &Problem{
		Type:   "https://bacon.dev/problems/test",
		Title:  "Test",
		Status: 400,
	}

	Write(w, p)

	var raw map[string]any
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if _, ok := raw["detail"]; ok {
		t.Error("expected detail to be omitted when empty")
	}
	if _, ok := raw["instance"]; ok {
		t.Error("expected instance to be omitted when empty")
	}
}
