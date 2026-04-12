package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	headerRequestID = "X-Request-Id"
)

func TestCorrelation_GeneratesID(t *testing.T) {
	handler := Correlation(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := r.Context().Value(RequestIDKey).(string)
		if !ok || id == "" {
			t.Error("expected request ID in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	got := w.Header().Get(headerRequestID)
	if got == "" {
		t.Fatal("expected " + headerRequestID + " header to be set")
	}
	if len(strings.Split(got, "-")) != 5 {
		t.Errorf("expected UUID format, got %q", got)
	}
}

func TestCorrelation_EchoesExisting(t *testing.T) {
	const existing = "my-custom-id-123"
	handler := Correlation(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Context().Value(RequestIDKey).(string)
		if id != existing {
			t.Errorf("context ID = %q, want %q", id, existing)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(headerRequestID, existing)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get(headerRequestID) != existing {
		t.Errorf("response header = %q, want %q", w.Header().Get(headerRequestID), existing)
	}
}

func TestCorrelation_UniqueIDs(t *testing.T) {
	ids := make(map[string]struct{})
	handler := Correlation(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		id := w.Header().Get(headerRequestID)
		if _, dup := ids[id]; dup {
			t.Fatalf("duplicate ID generated: %s", id)
		}
		ids[id] = struct{}{}
	}
}

func TestVersionHeader(t *testing.T) {
	old := Version
	Version = "1.2.3"
	defer func() { Version = old }()

	handler := VersionHeader(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	got := w.Header().Get("X-Bacon-Version")
	if got != "1.2.3" {
		t.Errorf("X-Bacon-Version = %q, want %q", got, "1.2.3")
	}
}

func TestVersionHeader_Default(t *testing.T) {
	old := Version
	Version = "dev"
	defer func() { Version = old }()

	handler := VersionHeader(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("X-Bacon-Version") != "dev" {
		t.Errorf("expected default version 'dev'")
	}
}

func TestNewUUID_Format(t *testing.T) {
	id := newUUID()
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Fatalf("expected 5 parts, got %d: %q", len(parts), id)
	}
	if len(parts[0]) != 8 || len(parts[1]) != 4 || len(parts[2]) != 4 || len(parts[3]) != 4 || len(parts[4]) != 12 {
		t.Errorf("unexpected UUID segment lengths: %q", id)
	}
	if parts[2][0] != '4' {
		t.Errorf("expected version 4, got %c", parts[2][0])
	}
}
