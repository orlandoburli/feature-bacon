package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orlandoburli/feature-bacon/internal/api/problem"
)

const pathFlags = "/api/v1/flags"

func TestReadOnly_GETPassesThrough(t *testing.T) {
	mw := ReadOnly(true)
	h := mw(okHandler())

	r := httptest.NewRequest(http.MethodGet, pathFlags, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestReadOnly_BlocksWrites(t *testing.T) {
	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	mw := ReadOnly(true)
	h := mw(okHandler())

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			r := httptest.NewRequest(method, pathFlags, nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)

			if w.Code != http.StatusConflict {
				t.Errorf("expected 409, got %d", w.Code)
			}

			var p problem.Problem
			if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
				t.Fatalf("decode problem: %v", err)
			}
			if p.Title != "Read-Only Mode" {
				t.Errorf("title = %q, want %q", p.Title, "Read-Only Mode")
			}
		})
	}
}

func TestReadOnly_Disabled(t *testing.T) {
	mw := ReadOnly(false)
	h := mw(okHandler())

	r := httptest.NewRequest(http.MethodPost, pathFlags, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when read-only disabled, got %d", w.Code)
	}
}
