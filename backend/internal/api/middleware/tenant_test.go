package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orlandoburli/feature-bacon/internal/api/handlers"
)

func TestTenantResolver_AuthDisabled(t *testing.T) {
	mw := TenantResolver(true)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tid, _ := r.Context().Value(handlers.TenantIDKey).(string)
		if tid != defaultTenantID {
			t.Errorf("expected %s, got %q", defaultTenantID, tid)
		}
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, pathEvaluate, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestTenantResolver_FromAuthContext(t *testing.T) {
	mw := TenantResolver(false)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tid, _ := r.Context().Value(handlers.TenantIDKey).(string)
		if tid != "acme" {
			t.Errorf("expected acme, got %q", tid)
		}
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, pathEvaluate, nil)
	ctx := context.WithValue(r.Context(), tenantIDKey, "acme")
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestTenantResolver_MissingTenant(t *testing.T) {
	mw := TenantResolver(false)
	h := mw(okHandler())

	r := httptest.NewRequest(http.MethodGet, pathEvaluate, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
