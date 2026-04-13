package main

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	bacon "github.com/orlandoburli/feature-bacon/sdks/go"
)

func mockBaconAPI(t *testing.T, healthOK bool, flagResults map[string]bacon.EvaluationResult) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			if healthOK {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
		case "/api/v1/evaluate":
			var req struct {
				FlagKey string `json:"flagKey"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			res, ok := flagResults[req.FlagKey]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(res)
		case "/api/v1/evaluate/batch":
			var req struct {
				FlagKeys []string `json:"flagKeys"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			var results []bacon.EvaluationResult
			for _, key := range req.FlagKeys {
				if res, ok := flagResults[key]; ok {
					results = append(results, res)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"results": results})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestPrice(t *testing.T) {
	if got := price(100.0, false); got != 100.0 {
		t.Errorf("price(100, false) = %f, want 100.0", got)
	}
	if got := price(100.0, true); math.Abs(got-90.0) > 0.001 {
		t.Errorf("price(100, true) = %f, want 90.0", got)
	}
	if got := price(29.99, true); math.Abs(got-26.991) > 0.001 {
		t.Errorf("price(29.99, true) = %f, want 26.991", got)
	}
}

func TestEnvOr(t *testing.T) {
	const key = "TEST_ENVOR_KEY_12345"
	os.Unsetenv(key)

	if got := envOr(key, "default"); got != "default" {
		t.Errorf("envOr unset = %q, want %q", got, "default")
	}

	t.Setenv(key, "custom")
	if got := envOr(key, "default"); got != "custom" {
		t.Errorf("envOr set = %q, want %q", got, "custom")
	}
}

func TestHandleHealth_Healthy(t *testing.T) {
	srv := mockBaconAPI(t, true, nil)
	defer srv.Close()
	client := bacon.NewClient(srv.URL)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	handleHealth(client)(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %v, want %q", body["status"], "ok")
	}
	if body["baconHealthy"] != true {
		t.Errorf("baconHealthy = %v, want true", body["baconHealthy"])
	}
}

func TestHandleHealth_Unhealthy(t *testing.T) {
	srv := mockBaconAPI(t, false, nil)
	defer srv.Close()
	client := bacon.NewClient(srv.URL)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	handleHealth(client)(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "degraded" {
		t.Errorf("status = %v, want %q", body["status"], "degraded")
	}
	if body["baconHealthy"] != false {
		t.Errorf("baconHealthy = %v, want false", body["baconHealthy"])
	}
}

func TestHandleHome(t *testing.T) {
	flags := map[string]bacon.EvaluationResult{
		"dark_mode":         {FlagKey: "dark_mode", Enabled: true, Variant: "on", Reason: "rule"},
		"new_pricing":       {FlagKey: "new_pricing", Enabled: false, Variant: "off", Reason: "default"},
		"beta_features":     {FlagKey: "beta_features", Enabled: true, Variant: "group_a", Reason: "rule"},
		"checkout_redesign": {FlagKey: "checkout_redesign", Enabled: true, Variant: "variant_b", Reason: "experiment"},
	}
	srv := mockBaconAPI(t, true, flags)
	defer srv.Close()
	client := bacon.NewClient(srv.URL)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?user=alice", nil)
	handleHome(client)(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["service"] != "product-catalog" {
		t.Errorf("service = %v, want %q", body["service"], "product-catalog")
	}
	if body["user"] != "alice" {
		t.Errorf("user = %v, want %q", body["user"], "alice")
	}
	features, ok := body["features"].(map[string]any)
	if !ok {
		t.Fatal("features is not a map")
	}
	for _, key := range []string{"dark_mode", "new_pricing", "beta_features", "checkout_redesign"} {
		if _, exists := features[key]; !exists {
			t.Errorf("missing feature %q", key)
		}
	}
	dm := features["dark_mode"].(map[string]any)
	if dm["enabled"] != true {
		t.Errorf("dark_mode.enabled = %v, want true", dm["enabled"])
	}
	if dm["variant"] != "on" {
		t.Errorf("dark_mode.variant = %v, want %q", dm["variant"], "on")
	}
}

func TestHandleHome_DefaultUser(t *testing.T) {
	flags := map[string]bacon.EvaluationResult{
		"dark_mode":         {FlagKey: "dark_mode"},
		"new_pricing":       {FlagKey: "new_pricing"},
		"beta_features":     {FlagKey: "beta_features"},
		"checkout_redesign": {FlagKey: "checkout_redesign"},
	}
	srv := mockBaconAPI(t, true, flags)
	defer srv.Close()
	client := bacon.NewClient(srv.URL)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handleHome(client)(rec, req)

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["user"] != "anonymous" {
		t.Errorf("user = %v, want %q", body["user"], "anonymous")
	}
}

func TestHandleProducts_NewPricingEnabled(t *testing.T) {
	flags := map[string]bacon.EvaluationResult{
		"new_pricing":       {FlagKey: "new_pricing", Enabled: true, Variant: "on"},
		"checkout_redesign": {FlagKey: "checkout_redesign", Enabled: true, Variant: "variant_b"},
	}
	srv := mockBaconAPI(t, true, flags)
	defer srv.Close()
	client := bacon.NewClient(srv.URL)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/products?user=bob", nil)
	handleProducts(client)(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["newPricingActive"] != true {
		t.Errorf("newPricingActive = %v, want true", body["newPricingActive"])
	}
	if body["checkoutVariant"] != "variant_b" {
		t.Errorf("checkoutVariant = %v, want %q", body["checkoutVariant"], "variant_b")
	}

	products := body["products"].([]any)
	if len(products) != 3 {
		t.Fatalf("got %d products, want 3", len(products))
	}
	first := products[0].(map[string]any)
	wantPrice := 29.99 * 0.9
	if math.Abs(first["price"].(float64)-wantPrice) > 0.01 {
		t.Errorf("first product price = %v, want %.2f", first["price"], wantPrice)
	}
}

func TestHandleProducts_NewPricingDisabled(t *testing.T) {
	flags := map[string]bacon.EvaluationResult{
		"new_pricing":       {FlagKey: "new_pricing", Enabled: false, Variant: "off"},
		"checkout_redesign": {FlagKey: "checkout_redesign", Enabled: false, Variant: "control"},
	}
	srv := mockBaconAPI(t, true, flags)
	defer srv.Close()
	client := bacon.NewClient(srv.URL)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/products?user=carol", nil)
	handleProducts(client)(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["newPricingActive"] != false {
		t.Errorf("newPricingActive = %v, want false", body["newPricingActive"])
	}
	if body["checkoutVariant"] != "control" {
		t.Errorf("checkoutVariant = %v, want %q", body["checkoutVariant"], "control")
	}

	products := body["products"].([]any)
	first := products[0].(map[string]any)
	if math.Abs(first["price"].(float64)-29.99) > 0.01 {
		t.Errorf("first product price = %v, want 29.99", first["price"])
	}
}

func TestHandleProducts_DefaultUser(t *testing.T) {
	flags := map[string]bacon.EvaluationResult{
		"new_pricing":       {FlagKey: "new_pricing", Enabled: false},
		"checkout_redesign": {FlagKey: "checkout_redesign", Enabled: false, Variant: "control"},
	}
	srv := mockBaconAPI(t, true, flags)
	defer srv.Close()
	client := bacon.NewClient(srv.URL)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	handleProducts(client)(rec, req)

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["newPricingActive"] != false {
		t.Errorf("newPricingActive = %v, want false", body["newPricingActive"])
	}
}

func TestHandleHome_BaconError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	client := bacon.NewClient(srv.URL)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?user=eve", nil)
	handleHome(client)(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (graceful degradation)", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	features, ok := body["features"].(map[string]any)
	if !ok {
		t.Fatal("features is not a map")
	}
	if len(features) != 0 {
		t.Errorf("expected empty features on error, got %d entries", len(features))
	}
	if body["user"] != "eve" {
		t.Errorf("user = %v, want %q", body["user"], "eve")
	}
}
