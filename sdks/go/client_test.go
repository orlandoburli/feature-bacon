package bacon_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	bacon "github.com/orlandoburli/feature-bacon/sdks/go"
)

const (
	contentTypeJSON    = "application/json"
	headerContentType  = "Content-Type"
	errUnexpectedPath  = "unexpected path: %s"
	testAPIKey         = "test-key"
	errUnexpectedError = "unexpected error: %v"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set(headerContentType, contentTypeJSON)
	json.NewEncoder(w).Encode(v)
}

var testCtx = bacon.EvaluationContext{
	SubjectID:   "user_123",
	Environment: "production",
	Attributes:  map[string]any{"plan": "pro", "country": "BR"},
}

// ---------------------------------------------------------------------------
// Evaluate
// ---------------------------------------------------------------------------

func evaluateHandler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/evaluate" {
			t.Errorf(errUnexpectedPath, r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != testAPIKey {
			t.Errorf("missing or wrong API key header")
		}
		if ct := r.Header.Get(headerContentType); ct != contentTypeJSON {
			t.Errorf("expected application/json content-type, got %s", ct)
		}

		var req struct {
			FlagKey string                  `json:"flagKey"`
			Context bacon.EvaluationContext `json:"context"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.FlagKey != "my_flag" {
			t.Errorf("expected flagKey my_flag, got %s", req.FlagKey)
		}
		if req.Context.SubjectID != "user_123" {
			t.Errorf("expected subjectId user_123, got %s", req.Context.SubjectID)
		}

		writeJSON(w, bacon.EvaluationResult{
			TenantID: "default",
			FlagKey:  "my_flag",
			Enabled:  true,
			Variant:  "control",
			Reason:   "rule_match",
		})
	}
}

func TestEvaluate(t *testing.T) {
	srv := newTestServer(evaluateHandler(t))
	defer srv.Close()

	client := bacon.NewClient(srv.URL, bacon.WithAPIKey(testAPIKey))
	result, err := client.Evaluate(context.Background(), "my_flag", testCtx)
	if err != nil {
		t.Fatalf(errUnexpectedError, err)
	}
	if result.TenantID != "default" {
		t.Errorf("tenantId = %q, want default", result.TenantID)
	}
	if !result.Enabled {
		t.Error("expected enabled = true")
	}
	if result.Variant != "control" {
		t.Errorf("variant = %q, want control", result.Variant)
	}
	if result.Reason != "rule_match" {
		t.Errorf("reason = %q, want rule_match", result.Reason)
	}
}

// ---------------------------------------------------------------------------
// EvaluateBatch
// ---------------------------------------------------------------------------

func TestEvaluateBatch(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/evaluate/batch" {
			t.Errorf(errUnexpectedPath, r.URL.Path)
		}

		var req struct {
			FlagKeys []string                `json:"flagKeys"`
			Context  bacon.EvaluationContext `json:"context"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(req.FlagKeys) != 2 {
			t.Fatalf("expected 2 flag keys, got %d", len(req.FlagKeys))
		}

		writeJSON(w, map[string]any{
			"results": []bacon.EvaluationResult{
				{TenantID: "default", FlagKey: "flag_a", Enabled: true, Reason: "rule_match"},
				{TenantID: "default", FlagKey: "flag_b", Enabled: false, Reason: "default"},
			},
		})
	})
	defer srv.Close()

	client := bacon.NewClient(srv.URL, bacon.WithAPIKey(testAPIKey))
	results, err := client.EvaluateBatch(context.Background(), []string{"flag_a", "flag_b"}, testCtx)
	if err != nil {
		t.Fatalf(errUnexpectedError, err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !results[0].Enabled {
		t.Error("flag_a should be enabled")
	}
	if results[1].Enabled {
		t.Error("flag_b should be disabled")
	}
}

// ---------------------------------------------------------------------------
// IsEnabled / GetVariant convenience methods
// ---------------------------------------------------------------------------

func TestIsEnabled(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, bacon.EvaluationResult{Enabled: true, FlagKey: "feat"})
	})
	defer srv.Close()

	client := bacon.NewClient(srv.URL)
	if !client.IsEnabled(context.Background(), "feat", bacon.EvaluationContext{SubjectID: "u1"}) {
		t.Error("expected IsEnabled = true")
	}
}

func TestIsEnabled_ErrorReturnsFalse(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	client := bacon.NewClient(srv.URL)
	if client.IsEnabled(context.Background(), "feat", bacon.EvaluationContext{SubjectID: "u1"}) {
		t.Error("expected IsEnabled = false on server error")
	}
}

func TestGetVariant(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, bacon.EvaluationResult{Variant: "beta", FlagKey: "feat"})
	})
	defer srv.Close()

	client := bacon.NewClient(srv.URL)
	if v := client.GetVariant(context.Background(), "feat", bacon.EvaluationContext{SubjectID: "u1"}); v != "beta" {
		t.Errorf("variant = %q, want beta", v)
	}
}

func TestGetVariant_ErrorReturnsEmpty(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	client := bacon.NewClient(srv.URL)
	if v := client.GetVariant(context.Background(), "feat", bacon.EvaluationContext{SubjectID: "u1"}); v != "" {
		t.Errorf("expected empty variant on error, got %q", v)
	}
}

// ---------------------------------------------------------------------------
// Healthy / Ready
// ---------------------------------------------------------------------------

func TestHealthy(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Errorf(errUnexpectedPath, r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	client := bacon.NewClient(srv.URL)
	if !client.Healthy(context.Background()) {
		t.Error("expected healthy = true")
	}
}

func TestHealthy_Unhealthy(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	defer srv.Close()

	client := bacon.NewClient(srv.URL)
	if client.Healthy(context.Background()) {
		t.Error("expected healthy = false when server returns 503")
	}
}

func TestReady(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readyz" {
			t.Errorf(errUnexpectedPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	client := bacon.NewClient(srv.URL)
	if !client.Ready(context.Background()) {
		t.Error("expected ready = true")
	}
}

// ---------------------------------------------------------------------------
// Error handling
// ---------------------------------------------------------------------------

func TestEvaluate_APIError(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"type":   "https://featurebacon.dev/errors/unauthorized",
			"title":  "Unauthorized",
			"detail": "invalid api key",
		})
	})
	defer srv.Close()

	client := bacon.NewClient(srv.URL)
	_, err := client.Evaluate(context.Background(), "flag", bacon.EvaluationContext{SubjectID: "u"})
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *bacon.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *bacon.Error, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", apiErr.StatusCode)
	}
	if apiErr.Title != "Unauthorized" {
		t.Errorf("title = %q, want Unauthorized", apiErr.Title)
	}
	if apiErr.Detail != "invalid api key" {
		t.Errorf("detail = %q", apiErr.Detail)
	}

	msg := apiErr.Error()
	if msg == "" {
		t.Error("error message should not be empty")
	}
}

func TestEvaluate_NonJSONError(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		io.WriteString(w, "bad gateway")
	})
	defer srv.Close()

	client := bacon.NewClient(srv.URL)
	_, err := client.Evaluate(context.Background(), "flag", bacon.EvaluationContext{SubjectID: "u"})
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *bacon.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *bacon.Error, got %T", err)
	}
	if apiErr.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", apiErr.StatusCode)
	}
	if apiErr.Detail != "bad gateway" {
		t.Errorf("detail = %q, want 'bad gateway'", apiErr.Detail)
	}
}

func TestEvaluate_InvalidJSON(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)
		io.WriteString(w, `{"broken`)
	})
	defer srv.Close()

	client := bacon.NewClient(srv.URL)
	_, err := client.Evaluate(context.Background(), "flag", bacon.EvaluationContext{SubjectID: "u"})
	if err == nil {
		t.Fatal("expected error on invalid JSON response")
	}
}

func TestEvaluate_NetworkError(t *testing.T) {
	client := bacon.NewClient("http://127.0.0.1:1")
	_, err := client.Evaluate(context.Background(), "flag", bacon.EvaluationContext{SubjectID: "u"})
	if err == nil {
		t.Fatal("expected error on unreachable server")
	}
}

func TestEvaluate_ContextCancelled(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	client := bacon.NewClient(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Evaluate(ctx, "flag", bacon.EvaluationContext{SubjectID: "u"})
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

// ---------------------------------------------------------------------------
// Options
// ---------------------------------------------------------------------------

func TestWithTimeout(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		writeJSON(w, bacon.EvaluationResult{})
	})
	defer srv.Close()

	client := bacon.NewClient(srv.URL, bacon.WithTimeout(50*time.Millisecond))
	_, err := client.Evaluate(context.Background(), "flag", bacon.EvaluationContext{SubjectID: "u"})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestWithHTTPClient(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, bacon.EvaluationResult{FlagKey: "f", Enabled: true})
	})
	defer srv.Close()

	custom := &http.Client{Timeout: 10 * time.Second}
	client := bacon.NewClient(srv.URL, bacon.WithHTTPClient(custom))
	result, err := client.Evaluate(context.Background(), "f", bacon.EvaluationContext{SubjectID: "u"})
	if err != nil {
		t.Fatalf(errUnexpectedError, err)
	}
	if !result.Enabled {
		t.Error("expected enabled")
	}
}

func TestBaseURLTrailingSlash(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Errorf(errUnexpectedPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	client := bacon.NewClient(srv.URL + "/")
	if !client.Healthy(context.Background()) {
		t.Error("expected healthy = true with trailing slash in baseURL")
	}
}

// ---------------------------------------------------------------------------
// Error.Error() with empty detail
// ---------------------------------------------------------------------------

func TestError_ErrorNoDetail(t *testing.T) {
	e := &bacon.Error{StatusCode: 403, Title: "Forbidden"}
	want := "bacon: Forbidden (403)"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
