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

const (
	baseURLFmt      = "http://%s"
	flagDarkMode    = "dark-mode"
	userDefault     = "user-1"
	flagDisabled    = "disabled-flag"
	flagNewCheckout = "new-checkout"
	decodeErrFmt    = "decode: %v"
	testAddr        = "127.0.0.1:18080"
)

var binaryPath string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "bacon-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkdirtemp: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	binaryPath = filepath.Join(tmp, "bacon-core")

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/bacon-core") //NOSONAR — test-only build step, no user input
	cmd.Dir = filepath.Join("..", "..")
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
      - key: ` + flagDarkMode + `
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
      - key: ` + flagNewCheckout + `
        type: boolean
        semantics: deterministic
        enabled: true
        description: New checkout flow
        rules: []
        defaultResult:
          enabled: true
          variant: v2
      - key: ` + flagDisabled + `
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

	base := fmt.Sprintf(baseURLFmt, testAddr)
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

func assertEvalResponse(t *testing.T, resp *http.Response, got *evalResponse, wantKey string, wantOn bool, wantVar, wantReason string) {
	t.Helper()
	if got.FlagKey != wantKey {
		t.Errorf("flagKey = %q, want %q", got.FlagKey, wantKey)
	}
	if got.Enabled != wantOn {
		t.Errorf("enabled = %v, want %v", got.Enabled, wantOn)
	}
	if got.Variant != wantVar {
		t.Errorf("variant = %q, want %q", got.Variant, wantVar)
	}
	if got.Reason != wantReason {
		t.Errorf("reason = %q, want %q", got.Reason, wantReason)
	}
	if resp.Header.Get("X-Request-Id") == "" {
		t.Error("X-Request-Id header missing")
	}
	if resp.Header.Get("X-Bacon-Version") == "" {
		t.Error("X-Bacon-Version header missing")
	}
}

func TestEvaluateSingleFlag(t *testing.T) {
	flagsFile := writeFlagsFile(t)
	cmd := startServer(t, flagsFile)
	defer stopServer(t, cmd)

	base := fmt.Sprintf(baseURLFmt, testAddr)

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
				"flagKey": flagDarkMode,
				"context": map[string]any{
					"subjectId":   userDefault,
					"environment": "production",
				},
			},
			wantKey: flagDarkMode,
			wantOn:  true,
			wantVar: "on",
			reason:  "rule_match",
		},
		{
			name: "dark-mode default in staging",
			body: map[string]any{
				"flagKey": flagDarkMode,
				"context": map[string]any{
					"subjectId":   userDefault,
					"environment": "staging",
				},
			},
			wantKey: flagDarkMode,
			wantOn:  false,
			wantVar: "off",
			reason:  "default",
		},
		{
			name: "disabled flag returns disabled",
			body: map[string]any{
				"flagKey": flagDisabled,
				"context": map[string]any{
					"subjectId": userDefault,
				},
			},
			wantKey: flagDisabled,
			wantOn:  false,
			wantVar: "",
			reason:  "disabled",
		},
		{
			name: "new-checkout default result",
			body: map[string]any{
				"flagKey": flagNewCheckout,
				"context": map[string]any{
					"subjectId": "user-2",
				},
			},
			wantKey: flagNewCheckout,
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
				t.Fatalf(decodeErrFmt, err)
			}

			assertEvalResponse(t, resp, &result, tt.wantKey, tt.wantOn, tt.wantVar, tt.reason)
		})
	}
}

func TestEvaluateBatch(t *testing.T) {
	flagsFile := writeFlagsFile(t)
	cmd := startServer(t, flagsFile)
	defer stopServer(t, cmd)

	base := fmt.Sprintf(baseURLFmt, testAddr)

	body := map[string]any{
		"flagKeys": []string{flagDarkMode, flagNewCheckout, flagDisabled},
		"context": map[string]any{
			"subjectId":   userDefault,
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
		t.Fatalf(decodeErrFmt, err)
	}

	if len(batch.Results) != 3 {
		t.Fatalf("results count = %d, want 3", len(batch.Results))
	}

	expected := map[string]struct {
		enabled bool
		variant string
	}{
		flagDarkMode:    {true, "on"},
		flagNewCheckout: {true, "v2"},
		flagDisabled:    {false, ""},
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

	if resp.Header.Get("X-Request-Id") == "" {
		t.Error("X-Request-Id header missing")
	}
	if resp.Header.Get("X-Bacon-Version") == "" {
		t.Error("X-Bacon-Version header missing")
	}
}

func TestNotFoundFlag(t *testing.T) {
	flagsFile := writeFlagsFile(t)
	cmd := startServer(t, flagsFile)
	defer stopServer(t, cmd)

	base := fmt.Sprintf(baseURLFmt, testAddr)

	body := map[string]any{
		"flagKey": "nonexistent",
		"context": map[string]any{
			"subjectId": userDefault,
		},
	}

	resp := postJSON(t, base+"/api/v1/evaluate", body)
	defer resp.Body.Close()

	var result evalResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf(decodeErrFmt, err)
	}

	if result.Enabled {
		t.Error("expected enabled = false for nonexistent flag")
	}
	if result.Reason != "not_found" {
		t.Errorf("reason = %q, want %q", result.Reason, "not_found")
	}
}
