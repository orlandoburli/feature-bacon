package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orlandoburli/feature-bacon/internal/auth"
)

func TestRequireScope_AuthDisabled(t *testing.T) {
	mw := RequireScope(auth.ScopeManagement, true)
	h := mw(okHandler())

	r := httptest.NewRequest(http.MethodGet, pathEvaluate, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when auth disabled, got %d", w.Code)
	}
}

func TestRequireScope_MatchingScope(t *testing.T) {
	mw := RequireScope(auth.ScopeEvaluation, false)
	h := mw(okHandler())

	r := httptest.NewRequest(http.MethodGet, pathEvaluate, nil)
	ctx := context.WithValue(r.Context(), scopeKey, auth.ScopeEvaluation)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequireScope_ManagementAccessAll(t *testing.T) {
	mw := RequireScope(auth.ScopeEvaluation, false)
	h := mw(okHandler())

	r := httptest.NewRequest(http.MethodGet, pathEvaluate, nil)
	ctx := context.WithValue(r.Context(), scopeKey, auth.ScopeManagement)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (management can access eval endpoints), got %d", w.Code)
	}
}

func TestRequireScope_InsufficientScope(t *testing.T) {
	mw := RequireScope(auth.ScopeManagement, false)
	h := mw(okHandler())

	r := httptest.NewRequest(http.MethodGet, pathEvaluate, nil)
	ctx := context.WithValue(r.Context(), scopeKey, auth.ScopeEvaluation)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRequireScope_NoScope(t *testing.T) {
	mw := RequireScope(auth.ScopeEvaluation, false)
	h := mw(okHandler())

	r := httptest.NewRequest(http.MethodGet, pathEvaluate, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for missing scope, got %d", w.Code)
	}
}
