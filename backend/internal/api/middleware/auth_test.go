package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orlandoburli/feature-bacon/internal/auth"
)

const (
	rawEvalKey   = "ba_eval_testkey123"
	authHeader   = "Authorization"
	contentJSON  = "application/problem+json"
	pathEvaluate = "/api/v1/evaluate"
	fmtExpect401 = "expected 401, got %d"
)

func newKeyStore() *auth.MemKeyStore {
	store := auth.NewMemKeyStore()
	store.Add(&auth.APIKey{
		ID:       "k1",
		TenantID: "acme",
		KeyHash:  auth.HashKey(rawEvalKey),
		Scope:    auth.ScopeEvaluation,
	})
	return store
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuth_Disabled(t *testing.T) {
	mw := Auth(AuthDeps{}, true)
	h := mw(okHandler())

	r := httptest.NewRequest(http.MethodGet, pathEvaluate, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when auth disabled, got %d", w.Code)
	}
}

func TestAuth_MissingHeader(t *testing.T) {
	mw := Auth(AuthDeps{KeyStore: newKeyStore()}, false)
	h := mw(okHandler())

	r := httptest.NewRequest(http.MethodGet, pathEvaluate, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf(fmtExpect401, w.Code)
	}
}

func TestAuth_MalformedHeader(t *testing.T) {
	mw := Auth(AuthDeps{KeyStore: newKeyStore()}, false)
	h := mw(okHandler())

	r := httptest.NewRequest(http.MethodGet, pathEvaluate, nil)
	r.Header.Set(authHeader, "InvalidFormat")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf(fmtExpect401, w.Code)
	}
}

func TestAuth_UnsupportedScheme(t *testing.T) {
	mw := Auth(AuthDeps{KeyStore: newKeyStore()}, false)
	h := mw(okHandler())

	r := httptest.NewRequest(http.MethodGet, pathEvaluate, nil)
	r.Header.Set(authHeader, "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf(fmtExpect401, w.Code)
	}
}

func TestAuth_ValidAPIKey(t *testing.T) {
	mw := Auth(AuthDeps{KeyStore: newKeyStore()}, false)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tid := TenantIDFromRequest(r)
		if tid != "acme" {
			t.Errorf("expected tenant acme, got %q", tid)
		}
		scope := ScopeFromRequest(r)
		if scope != auth.ScopeEvaluation {
			t.Errorf("expected scope evaluation, got %q", scope)
		}
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, pathEvaluate, nil)
	r.Header.Set(authHeader, "ApiKey "+rawEvalKey)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuth_InvalidAPIKey(t *testing.T) {
	mw := Auth(AuthDeps{KeyStore: newKeyStore()}, false)
	h := mw(okHandler())

	r := httptest.NewRequest(http.MethodGet, pathEvaluate, nil)
	r.Header.Set(authHeader, "ApiKey ba_eval_wrong")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf(fmtExpect401, w.Code)
	}
}

func TestAuth_BearerWithoutJWTConfig(t *testing.T) {
	mw := Auth(AuthDeps{KeyStore: newKeyStore(), JWTEnabled: false}, false)
	h := mw(okHandler())

	r := httptest.NewRequest(http.MethodGet, pathEvaluate, nil)
	r.Header.Set(authHeader, "Bearer some.jwt.token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf(fmtExpect401, w.Code)
	}
}

func TestAuth_EmptyToken(t *testing.T) {
	mw := Auth(AuthDeps{KeyStore: newKeyStore()}, false)
	h := mw(okHandler())

	r := httptest.NewRequest(http.MethodGet, pathEvaluate, nil)
	r.Header.Set(authHeader, "ApiKey ")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for empty token, got %d", w.Code)
	}
}
