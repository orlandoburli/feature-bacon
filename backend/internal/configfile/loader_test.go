package configfile

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/orlandoburli/feature-bacon/internal/engine"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const multiTenantYAML = `
tenants:
  acme:
    flags:
      - key: dark_mode
        type: boolean
        semantics: deterministic
        enabled: true
        rules:
          - conditions:
              - attribute: attributes.plan
                operator: equals
                value: "premium"
            rolloutPercentage: 100
            variant: ""
        defaultResult:
          enabled: false
          variant: ""
      - key: new_checkout
        type: boolean
        semantics: random
        enabled: false
        rules: []
        defaultResult:
          enabled: false
          variant: ""
  globex:
    flags:
      - key: beta_api
        type: boolean
        semantics: deterministic
        enabled: true
        rules: []
        defaultResult:
          enabled: true
          variant: ""
`

func TestMultiTenantFile(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "flags.yaml", multiTenantYAML)

	s, err := New(path)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	flag, err := s.GetFlag("acme", "dark_mode")
	if err != nil {
		t.Fatal(err)
	}
	if flag == nil {
		t.Fatal("expected dark_mode flag, got nil")
	}
	if flag.Type != engine.FlagTypeBoolean {
		t.Errorf("type = %q, want %q", flag.Type, engine.FlagTypeBoolean)
	}
	if !flag.Enabled {
		t.Error("expected flag to be enabled")
	}
	if len(flag.Rules) != 1 {
		t.Fatalf("rules len = %d, want 1", len(flag.Rules))
	}
	if flag.Rules[0].RolloutPercentage != 100 {
		t.Errorf("rollout = %d, want 100", flag.Rules[0].RolloutPercentage)
	}

	keys, err := s.ListFlagKeys("acme")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(keys)
	if len(keys) != 2 || keys[0] != "dark_mode" || keys[1] != "new_checkout" {
		t.Errorf("acme keys = %v, want [dark_mode new_checkout]", keys)
	}

	keys, err = s.ListFlagKeys("globex")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0] != "beta_api" {
		t.Errorf("globex keys = %v, want [beta_api]", keys)
	}
}

func TestDirectoryMode(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "acme.yaml", `
flags:
  - key: feature_x
    type: boolean
    semantics: deterministic
    enabled: true
    rules: []
    defaultResult:
      enabled: false
      variant: ""
`)
	writeFile(t, dir, "globex.yaml", `
flags:
  - key: feature_y
    type: variant
    semantics: random
    enabled: true
    rules: []
    defaultResult:
      enabled: true
      variant: control
`)
	writeFile(t, dir, "readme.txt", "not a yaml file")

	s, err := New(dir)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	flag, err := s.GetFlag("acme", "feature_x")
	if err != nil {
		t.Fatal(err)
	}
	if flag == nil || !flag.Enabled {
		t.Error("expected acme/feature_x to be present and enabled")
	}

	flag, err = s.GetFlag("globex", "feature_y")
	if err != nil {
		t.Fatal(err)
	}
	if flag == nil || flag.Type != engine.FlagTypeVariant {
		t.Error("expected globex/feature_y to be present as variant type")
	}
	if flag.DefaultResult.Variant != "control" {
		t.Errorf("default variant = %q, want %q", flag.DefaultResult.Variant, "control")
	}
}

func TestSidecarSingleTenant(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "flags.yaml", `
flags:
  - key: my_flag
    type: boolean
    semantics: deterministic
    enabled: true
    rules: []
    defaultResult:
      enabled: false
      variant: ""
`)

	s, err := New(path)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	flag, err := s.GetFlag(DefaultTenant, "my_flag")
	if err != nil {
		t.Fatal(err)
	}
	if flag == nil {
		t.Fatal("expected my_flag under default tenant")
	}
	if !flag.Enabled {
		t.Error("expected flag to be enabled")
	}
}

func TestValidationMissingKey(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "flags.yaml", `
tenants:
  acme:
    flags:
      - type: boolean
        semantics: deterministic
        enabled: true
`)

	_, err := New(path)
	if err == nil {
		t.Fatal("expected error for flag with missing key")
	}
	want := `configfile: tenant "acme" has a flag with missing key`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestMissingFile(t *testing.T) {
	_, err := New("/nonexistent/path/flags.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestMissingDirectory(t *testing.T) {
	_, err := New("/nonexistent/dir/")
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestReload(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "flags.yaml", `
tenants:
  acme:
    flags:
      - key: feat_a
        type: boolean
        semantics: deterministic
        enabled: true
        rules: []
        defaultResult:
          enabled: false
          variant: ""
`)

	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}

	keys, _ := s.ListFlagKeys("acme")
	if len(keys) != 1 || keys[0] != "feat_a" {
		t.Fatalf("before reload: keys = %v", keys)
	}

	writeFile(t, dir, "flags.yaml", `
tenants:
  acme:
    flags:
      - key: feat_a
        type: boolean
        semantics: deterministic
        enabled: true
        rules: []
        defaultResult:
          enabled: false
          variant: ""
      - key: feat_b
        type: boolean
        semantics: deterministic
        enabled: false
        rules: []
        defaultResult:
          enabled: false
          variant: ""
`)

	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}

	keys, _ = s.ListFlagKeys("acme")
	sort.Strings(keys)
	if len(keys) != 2 || keys[0] != "feat_a" || keys[1] != "feat_b" {
		t.Errorf("after reload: keys = %v, want [feat_a feat_b]", keys)
	}
}

func TestGetFlagNonExistentTenant(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "flags.yaml", multiTenantYAML)

	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}

	flag, err := s.GetFlag("nope", "dark_mode")
	if err != nil {
		t.Fatal(err)
	}
	if flag != nil {
		t.Error("expected nil for non-existent tenant")
	}
}

func TestGetFlagNonExistentKey(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "flags.yaml", multiTenantYAML)

	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}

	flag, err := s.GetFlag("acme", "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if flag != nil {
		t.Error("expected nil for non-existent flag key")
	}
}

func TestListFlagKeysNonExistentTenant(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "flags.yaml", multiTenantYAML)

	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}

	keys, err := s.ListFlagKeys("nope")
	if err != nil {
		t.Fatal(err)
	}
	if keys != nil {
		t.Errorf("expected nil for non-existent tenant, got %v", keys)
	}
}

func TestConcurrentReads(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "flags.yaml", multiTenantYAML)

	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = s.GetFlag("acme", "dark_mode")
		}()
		go func() {
			defer wg.Done()
			_, _ = s.ListFlagKeys("acme")
		}()
	}
	wg.Wait()
}

func TestConcurrentReadsDuringReload(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "flags.yaml", multiTenantYAML)

	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			_ = s.Reload()
		}()
		go func() {
			defer wg.Done()
			_, _ = s.GetFlag("acme", "dark_mode")
		}()
		go func() {
			defer wg.Done()
			_, _ = s.ListFlagKeys("globex")
		}()
	}
	wg.Wait()
}

func TestDirectoryYMLExtension(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "tenant1.yml", `
flags:
  - key: yml_flag
    type: boolean
    semantics: deterministic
    enabled: true
    rules: []
    defaultResult:
      enabled: false
      variant: ""
`)

	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	flag, err := s.GetFlag("tenant1", "yml_flag")
	if err != nil {
		t.Fatal(err)
	}
	if flag == nil {
		t.Fatal("expected yml_flag under tenant1")
	}
}

func TestPersistentSemanticsWarning(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "flags.yaml", `
tenants:
  acme:
    flags:
      - key: sticky_flag
        type: boolean
        semantics: persistent
        enabled: true
        rules: []
        defaultResult:
          enabled: false
          variant: ""
`)

	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}

	flag, err := s.GetFlag("acme", "sticky_flag")
	if err != nil {
		t.Fatal(err)
	}
	if flag == nil {
		t.Fatal("expected sticky_flag")
	}
	if flag.Semantics != engine.SemanticsPersistent {
		t.Errorf("semantics = %q, want %q", flag.Semantics, engine.SemanticsPersistent)
	}
}

func TestSampleFlagsFile(t *testing.T) {
	s, err := New("../../testdata/flags.yaml")
	if err != nil {
		t.Fatalf("failed to load testdata/flags.yaml: %v", err)
	}

	acmeKeys, _ := s.ListFlagKeys("acme")
	if len(acmeKeys) < 2 {
		t.Errorf("acme should have ≥2 flags, got %d", len(acmeKeys))
	}

	globexKeys, _ := s.ListFlagKeys("globex")
	if len(globexKeys) < 1 {
		t.Errorf("globex should have ≥1 flag, got %d", len(globexKeys))
	}
}

func TestEmptyDirectoryLoads(t *testing.T) {
	dir := t.TempDir()

	s, err := New(dir)
	if err != nil {
		t.Fatalf("New() error on empty dir: %v", err)
	}

	keys, err := s.ListFlagKeys("anything")
	if err != nil {
		t.Fatal(err)
	}
	if keys != nil {
		t.Errorf("expected nil keys from empty dir, got %v", keys)
	}
}

func TestInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "flags.yaml", `{{{invalid yaml`)

	_, err := New(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestDirectoryWithSubdirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "acme.yaml", `
flags:
  - key: feat
    type: boolean
    semantics: deterministic
    enabled: true
    rules: []
    defaultResult:
      enabled: false
      variant: ""
`)
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	flag, _ := s.GetFlag("acme", "feat")
	if flag == nil {
		t.Fatal("expected feat flag despite subdirectory presence")
	}
}

func TestDirectoryWithInvalidYAMLFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bad.yaml", `{{{not valid yaml`)

	_, err := New(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML in directory mode")
	}
}

func TestWatchSignalReloads(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "flags.yaml", `
tenants:
  acme:
    flags:
      - key: v1
        type: boolean
        semantics: deterministic
        enabled: true
        rules: []
        defaultResult:
          enabled: false
          variant: ""
`)

	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		s.WatchSignal(ctx)
		close(done)
	}()

	writeFile(t, dir, "flags.yaml", `
tenants:
  acme:
    flags:
      - key: v1
        type: boolean
        semantics: deterministic
        enabled: true
        rules: []
        defaultResult:
          enabled: false
          variant: ""
      - key: v2
        type: boolean
        semantics: deterministic
        enabled: true
        rules: []
        defaultResult:
          enabled: false
          variant: ""
`)

	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Signal(syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}

	// Give the goroutine time to process the signal and reload.
	deadline := time.After(2 * time.Second)
	for {
		keys, _ := s.ListFlagKeys("acme")
		if len(keys) == 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for SIGHUP reload")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	cancel()
	<-done
}

func TestReloadWithInvalidFileKeepsOldData(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "flags.yaml", `
tenants:
  acme:
    flags:
      - key: feat_a
        type: boolean
        semantics: deterministic
        enabled: true
        rules: []
        defaultResult:
          enabled: false
          variant: ""
`)

	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}

	flag, _ := s.GetFlag("acme", "feat_a")
	if flag == nil {
		t.Fatal("expected feat_a before bad reload")
	}

	writeFile(t, dir, "flags.yaml", `
tenants:
  acme:
    flags:
      - type: boolean
        semantics: deterministic
        enabled: true
`)

	err = s.Reload()
	if err == nil {
		t.Fatal("expected reload to fail on validation")
	}

	flag, _ = s.GetFlag("acme", "feat_a")
	if flag == nil {
		t.Fatal("expected old data to be preserved after failed reload")
	}
}
