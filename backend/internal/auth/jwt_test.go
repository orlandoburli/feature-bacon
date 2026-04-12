package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testKid        = "test-kid-1"
	testIssuer     = "https://auth.test.com"
	testAudience   = "bacon-api"
	claimTenant    = "org_id"
	tenantAcme     = "acme"
	fmtUnexpectErr = "unexpected error: %v"
)

type testJWKS struct {
	key    *rsa.PrivateKey
	server *httptest.Server
}

func newTestJWKS(t *testing.T) *testJWKS {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tj := &testJWKS{key: key}
	tj.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jwks := map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"use": "sig",
					"kid": testKid,
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	t.Cleanup(tj.server.Close)
	return tj
}

func (tj *testJWKS) signToken(claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testKid
	s, err := token.SignedString(tj.key)
	if err != nil {
		panic(err)
	}
	return s
}

func TestJWTValidator_ValidToken(t *testing.T) {
	tj := newTestJWKS(t)
	v := NewJWTValidator(JWTConfig{
		Issuer:      testIssuer,
		Audience:    testAudience,
		JWKSURL:     tj.server.URL,
		TenantClaim: claimTenant,
	})

	tokenStr := tj.signToken(jwt.MapClaims{
		"iss":       testIssuer,
		"aud":       testAudience,
		claimTenant: tenantAcme,
		"exp":       time.Now().Add(time.Hour).Unix(),
	})

	result, err := v.Validate(tokenStr)
	if err != nil {
		t.Fatalf(fmtUnexpectErr, err)
	}
	if result.TenantID != tenantAcme {
		t.Errorf("expected tenant %s, got %s", tenantAcme, result.TenantID)
	}
}

func TestJWTValidator_ExpiredToken(t *testing.T) {
	tj := newTestJWKS(t)
	v := NewJWTValidator(JWTConfig{
		Issuer:      testIssuer,
		Audience:    testAudience,
		JWKSURL:     tj.server.URL,
		TenantClaim: claimTenant,
	})

	tokenStr := tj.signToken(jwt.MapClaims{
		"iss":       testIssuer,
		"aud":       testAudience,
		claimTenant: tenantAcme,
		"exp":       time.Now().Add(-time.Hour).Unix(),
	})

	_, err := v.Validate(tokenStr)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestJWTValidator_MissingTenantClaim(t *testing.T) {
	tj := newTestJWKS(t)
	v := NewJWTValidator(JWTConfig{
		Issuer:      testIssuer,
		Audience:    testAudience,
		JWKSURL:     tj.server.URL,
		TenantClaim: claimTenant,
	})

	tokenStr := tj.signToken(jwt.MapClaims{
		"iss": testIssuer,
		"aud": testAudience,
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	_, err := v.Validate(tokenStr)
	if err == nil {
		t.Fatal("expected error for missing tenant claim")
	}
}

func TestJWTValidator_WrongIssuer(t *testing.T) {
	tj := newTestJWKS(t)
	v := NewJWTValidator(JWTConfig{
		Issuer:      testIssuer,
		Audience:    testAudience,
		JWKSURL:     tj.server.URL,
		TenantClaim: claimTenant,
	})

	tokenStr := tj.signToken(jwt.MapClaims{
		"iss":       "https://wrong-issuer.com",
		"aud":       testAudience,
		claimTenant: tenantAcme,
		"exp":       time.Now().Add(time.Hour).Unix(),
	})

	_, err := v.Validate(tokenStr)
	if err == nil {
		t.Fatal("expected error for wrong issuer")
	}
}

func TestJWTValidator_WrongAudience(t *testing.T) {
	tj := newTestJWKS(t)
	v := NewJWTValidator(JWTConfig{
		Issuer:      testIssuer,
		Audience:    testAudience,
		JWKSURL:     tj.server.URL,
		TenantClaim: claimTenant,
	})

	tokenStr := tj.signToken(jwt.MapClaims{
		"iss":       testIssuer,
		"aud":       "wrong-audience",
		claimTenant: tenantAcme,
		"exp":       time.Now().Add(time.Hour).Unix(),
	})

	_, err := v.Validate(tokenStr)
	if err == nil {
		t.Fatal("expected error for wrong audience")
	}
}

func TestJWTValidator_WithScopeClaim(t *testing.T) {
	tj := newTestJWKS(t)
	v := NewJWTValidator(JWTConfig{
		Issuer:      testIssuer,
		Audience:    testAudience,
		JWKSURL:     tj.server.URL,
		TenantClaim: claimTenant,
		ScopeClaim:  "scope",
	})

	tokenStr := tj.signToken(jwt.MapClaims{
		"iss":       testIssuer,
		"aud":       testAudience,
		claimTenant: tenantAcme,
		"scope":     "management",
		"exp":       time.Now().Add(time.Hour).Unix(),
	})

	result, err := v.Validate(tokenStr)
	if err != nil {
		t.Fatalf(fmtUnexpectErr, err)
	}
	if result.Scope != ScopeManagement {
		t.Errorf("expected scope management, got %s", result.Scope)
	}
}

func TestJWTValidator_NestedTenantClaim(t *testing.T) {
	tj := newTestJWKS(t)
	v := NewJWTValidator(JWTConfig{
		Issuer:      testIssuer,
		Audience:    testAudience,
		JWKSURL:     tj.server.URL,
		TenantClaim: "custom.tenant_id",
	})

	tokenStr := tj.signToken(jwt.MapClaims{
		"iss":    testIssuer,
		"aud":    testAudience,
		"custom": map[string]any{"tenant_id": tenantAcme},
		"exp":    time.Now().Add(time.Hour).Unix(),
	})

	result, err := v.Validate(tokenStr)
	if err != nil {
		t.Fatalf(fmtUnexpectErr, err)
	}
	if result.TenantID != tenantAcme {
		t.Errorf("expected tenant %s, got %s", tenantAcme, result.TenantID)
	}
}

func TestJWTValidator_UnknownKid(t *testing.T) {
	tj := newTestJWKS(t)
	v := NewJWTValidator(JWTConfig{
		Issuer:      testIssuer,
		Audience:    testAudience,
		JWKSURL:     tj.server.URL,
		TenantClaim: claimTenant,
	})

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":       testIssuer,
		"aud":       testAudience,
		claimTenant: tenantAcme,
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
	token.Header["kid"] = "unknown-kid"
	tokenStr, _ := token.SignedString(tj.key)

	_, err := v.Validate(tokenStr)
	if err == nil {
		t.Fatal("expected error for unknown kid")
	}
}

func TestJWTValidator_JWKSFetchFailure(t *testing.T) {
	v := NewJWTValidator(JWTConfig{
		JWKSURL:     "http://127.0.0.1:1/nonexistent",
		TenantClaim: claimTenant,
	})

	_, err := v.Validate("some.jwt.token")
	if err == nil {
		t.Fatal("expected error for unreachable JWKS")
	}
}

func TestJWTConfig_Enabled(t *testing.T) {
	c := JWTConfig{}
	if c.Enabled() {
		t.Error("expected disabled without JWKSURL")
	}
	c.JWKSURL = "https://example.com/.well-known/jwks.json"
	if !c.Enabled() {
		t.Error("expected enabled with JWKSURL")
	}
}
