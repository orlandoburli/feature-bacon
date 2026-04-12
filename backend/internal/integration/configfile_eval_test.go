//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

var binaryPath string

const testAddr = "127.0.0.1:18080"

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "bacon-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkdirtemp: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	binaryPath = filepath.Join(tmp, "bacon-core")

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/bacon-core") //NOSONAR — test-only build step, no user input
	cmd.Dir = filepath.Join("..", "..", "..")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// Uses _default tenant because no auth middleware sets tenant context yet.
// Second tenant (acme) included to satisfy multi-tenant YAML structure.
const flagsYAML = `tenants:
  _default:
    flags:
      - key: dark-mode
        type: boolean
        semantics: deterministic
        enabled: true
        description: Dark mode toggle
        rules:
          - conditions:
              - attribute: environment
                operator: equals
                value: production
            rolloutPercentage: 100
            variant: "on"
        defaultResult:
          enabled: false
          variant: "off"
      - key: new-checkout
        type: boolean
        semantics: deterministic
        enabled: true
        description: New checkout flow
        rules: []
        defaultResult:
          enabled: true
          variant: v2
      - key: disabled-flag
        type: boolean
        semantics: deterministic
        enabled: false
        description: A disabled flag
        rules: []
        defaultResult:
          enabled: false
          variant: ""
  acme:
    flags:
      - key: beta-feature
        type: boolean
        semantics: deterministic
        enabled: true
        description: Beta feature for acme
        rules:
          - conditions:
              - attribute: environment
                operator: equals
                value: staging
            rolloutPercentage: 100
            variant: beta
        defaultResult:
          enabled: false
          variant: stable
`

type evalResponse struct {
	TenantID string `json:"tenantId"`
	FlagKey  string `json:"flagKey"`
	Enabled  bool   `json:"enabled"`
	Variant  string `json:"variant"`
	Reason   string `json:"reason"`
}

type batchResponse struct {
	Results []evalResponse `json:"results"`
}

func startServer(t *testing.T, flagsFile string) *exec.Cmd {
	t.Helper()

	cmd := exec.Command(binaryPath)
	cmd.Env = append(os.Environ(),
		"BACON_PERSISTENCE=file",
		"BACON_CONFIG_FILE="+flagsFile,
		"BACON_HTTP_ADDR="+testAddr,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}

	base := fmt.Sprintf("http://%s", testAddr)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return cmd
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = cmd.Process.Kill()
	t.Fatal("server did not become ready within 10s")
	return nil
}

func stopServer(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

func writeFlagsFile(t *testing.T) string {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), "flags.yaml")
	if err := os.WriteFile(tmp, []byte(flagsYAML), 0644); err != nil {
		t.Fatalf("write flags file: %v", err)
	}
	return tmp
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func TestEvaluateSingleFlag(t *testing.T) {
	flagsFile := writeFlagsFile(t)
	cmd := startServer(t, flagsFile)
	defer stopServer(t, cmd)

	base := fmt.Sprintf("http://%s", testAddr)

	tests := []struct {
		name    string
		body    map[string]any
		wantKey string
		wantOn  bool
		wantVar string
		reason  string
	}{
		{
			name: "dark-mode enabled in production",
			body: map[string]any{
				"flagKey": "dark-mode",
				"context": map[string]any{
					"subjectId":   "user-1",
					"environment": "production",
				},
			},
			wantKey: "dark-mode",
			wantOn:  true,
			wantVar: "on",
			reason:  "rule_match",
		},
		{
			name: "dark-mode default in staging",
			body: map[string]any{
				"flagKey": "dark-mode",
				"context": map[string]any{
					"subjectId":   "user-1",
					"environment": "staging",
				},
			},
			wantKey: "dark-mode",
			wantOn:  false,
			wantVar: "off",
			reason:  "default",
		},
		{
			name: "disabled flag returns disabled",
			body: map[string]any{
				"flagKey": "disabled-flag",
				"context": map[string]any{
					"subjectId": "user-1",
				},
			},
			wantKey: "disabled-flag",
			wantOn:  false,
			wantVar: "",
			reason:  "disabled",
		},
		{
			name: "new-checkout default result",
			body: map[string]any{
				"flagKey": "new-checkout",
				"context": map[string]any{
					"subjectId": "user-2",
				},
			},
			wantKey: "new-checkout",
			wantOn:  true,
			wantVar: "v2",
			reason:  "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := postJSON(t, base+"/api/v1/evaluate", tt.body)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}

			var result evalResponse
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("decode: %v", err)
			}

			if result.FlagKey != tt.wantKey {
				t.Errorf("flagKey = %q, want %q", result.FlagKey, tt.wantKey)
			}
			if result.Enabled != tt.wantOn {
				t.Errorf("enabled = %v, want %v", result.Enabled, tt.wantOn)
			}
			if result.Variant != tt.wantVar {
				t.Errorf("variant = %q, want %q", result.Variant, tt.wantVar)
			}
			if result.Reason != tt.reason {
				t.Errorf("reason = %q, want %q", result.Reason, tt.reason)
			}

			if rid := resp.Header.Get("X-Request-Id"); rid == "" {
				t.Error("X-Request-Id header missing")
			}
			if ver := resp.Header.Get("X-Bacon-Version"); ver == "" {
				t.Error("X-Bacon-Version header missing")
			}
		})
	}
}

func TestEvaluateBatch(t *testing.T) {
	flagsFile := writeFlagsFile(t)
	cmd := startServer(t, flagsFile)
	defer stopServer(t, cmd)

	base := fmt.Sprintf("http://%s", testAddr)

	body := map[string]any{
		"flagKeys": []string{"dark-mode", "new-checkout", "disabled-flag"},
		"context": map[string]any{
			"subjectId":   "user-1",
			"environment": "production",
		},
	}

	resp := postJSON(t, base+"/api/v1/evaluate/batch", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var batch batchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(batch.Results) != 3 {
		t.Fatalf("results count = %d, want 3", len(batch.Results))
	}

	expected := map[string]struct {
		enabled bool
		variant string
	}{
		"dark-mode":     {true, "on"},
		"new-checkout":  {true, "v2"},
		"disabled-flag": {false, ""},
	}

	for _, r := range batch.Results {
		want, ok := expected[r.FlagKey]
		if !ok {
			t.Errorf("unexpected flag key: %s", r.FlagKey)
			continue
		}
		if r.Enabled != want.enabled {
			t.Errorf("%s: enabled = %v, want %v", r.FlagKey, r.Enabled, want.enabled)
		}
		if r.Variant != want.variant {
			t.Errorf("%s: variant = %q, want %q", r.FlagKey, r.Variant, want.variant)
		}
	}

	if rid := resp.Header.Get("X-Request-Id"); rid == "" {
		t.Error("X-Request-Id header missing")
	}
	if ver := resp.Header.Get("X-Bacon-Version"); ver == "" {
		t.Error("X-Bacon-Version header missing")
	}
}

func TestNotFoundFlag(t *testing.T) {
	flagsFile := writeFlagsFile(t)
	cmd := startServer(t, flagsFile)
	defer stopServer(t, cmd)

	base := fmt.Sprintf("http://%s", testAddr)

	body := map[string]any{
		"flagKey": "nonexistent",
		"context": map[string]any{
			"subjectId": "user-1",
		},
	}

	resp := postJSON(t, base+"/api/v1/evaluate", body)
	defer resp.Body.Close()

	var result evalResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if result.Enabled {
		t.Error("expected enabled = false for nonexistent flag")
	}
	if result.Reason != "not_found" {
		t.Errorf("reason = %q, want %q", result.Reason, "not_found")
	}
}
