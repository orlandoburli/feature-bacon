package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig holds the configuration for JWT validation.
type JWTConfig struct {
	Issuer      string
	Audience    string
	JWKSURL     string
	TenantClaim string
	ScopeClaim  string
}

// Enabled returns true if JWT authentication is configured.
func (c *JWTConfig) Enabled() bool {
	return c.JWKSURL != ""
}

// JWTValidator validates JWT tokens against a JWKS endpoint.
type JWTValidator struct {
	config     JWTConfig
	mu         sync.RWMutex
	keys       map[string]*rsa.PublicKey
	lastFetch  time.Time
	refreshTTL time.Duration
	httpClient *http.Client
}

// NewJWTValidator creates a validator. Keys are fetched lazily on first use.
func NewJWTValidator(cfg JWTConfig) *JWTValidator {
	return &JWTValidator{
		config:     cfg,
		keys:       make(map[string]*rsa.PublicKey),
		refreshTTL: 5 * time.Minute,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// JWTResult holds the extracted claims after successful validation.
type JWTResult struct {
	TenantID string
	Scope    Scope
}

// Validate parses and validates a JWT token string.
func (v *JWTValidator) Validate(tokenStr string) (*JWTResult, error) {
	if err := v.ensureKeys(); err != nil {
		return nil, fmt.Errorf("jwks fetch failed: %w", err)
	}

	token, err := jwt.Parse(tokenStr, v.keyFunc, v.parserOptions()...)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	tenantID, err := extractClaim(claims, v.config.TenantClaim)
	if err != nil {
		return nil, fmt.Errorf("missing tenant claim %q", v.config.TenantClaim)
	}

	var scope Scope
	if v.config.ScopeClaim != "" {
		if s, err := extractClaim(claims, v.config.ScopeClaim); err == nil {
			scope = Scope(s)
		}
	}

	return &JWTResult{TenantID: tenantID, Scope: scope}, nil
}

func (v *JWTValidator) parserOptions() []jwt.ParserOption {
	opts := []jwt.ParserOption{jwt.WithValidMethods([]string{"RS256"})}
	if v.config.Issuer != "" {
		opts = append(opts, jwt.WithIssuer(v.config.Issuer))
	}
	if v.config.Audience != "" {
		opts = append(opts, jwt.WithAudience(v.config.Audience))
	}
	return opts
}

func (v *JWTValidator) keyFunc(token *jwt.Token) (any, error) {
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("missing kid in token header")
	}

	v.mu.RLock()
	key, found := v.keys[kid]
	v.mu.RUnlock()

	if found {
		return key, nil
	}

	if err := v.fetchKeys(); err != nil {
		return nil, err
	}

	v.mu.RLock()
	key, found = v.keys[kid]
	v.mu.RUnlock()

	if !found {
		return nil, fmt.Errorf("unknown kid %q", kid)
	}
	return key, nil
}

func (v *JWTValidator) ensureKeys() error {
	v.mu.RLock()
	fresh := len(v.keys) > 0 && time.Since(v.lastFetch) < v.refreshTTL
	v.mu.RUnlock()

	if fresh {
		return nil
	}
	return v.fetchKeys()
}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (v *JWTValidator) fetchKeys() error {
	resp, err := v.httpClient.Get(v.config.JWKSURL)
	if err != nil {
		return fmt.Errorf("jwks request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks endpoint returned %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("jwks decode failed: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" || k.Use != "sig" {
			continue
		}
		pub, err := parseRSAPublicKey(k)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}

	v.mu.Lock()
	v.keys = keys
	v.lastFetch = time.Now()
	v.mu.Unlock()
	return nil
}

func parseRSAPublicKey(k jwkKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}

func extractClaim(claims jwt.MapClaims, path string) (string, error) {
	parts := strings.Split(path, ".")
	var current any = map[string]any(claims)

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return "", fmt.Errorf("claim %q is not an object", part)
		}
		current = m[part]
	}

	if s, ok := current.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("claim %q not found or not a string", path)
}
