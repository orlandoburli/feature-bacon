package configfile

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/orlandoburli/feature-bacon/internal/engine"
	"gopkg.in/yaml.v3"
)

const DefaultTenant = "_default"

// multiTenantFile is the top-level YAML structure for multi-tenant mode.
type multiTenantFile struct {
	Tenants map[string]tenantBlock `yaml:"tenants"`
}

type tenantBlock struct {
	Flags []engine.FlagDefinition `yaml:"flags"`
}

// singleTenantFile is the top-level YAML structure for sidecar / single-tenant files.
type singleTenantFile struct {
	Flags []engine.FlagDefinition `yaml:"flags"`
}

// Store is a file-backed FlagStore that supports single-file, directory,
// and sidecar modes. It is safe for concurrent reads and supports hot reload.
type Store struct {
	path string
	mu   sync.RWMutex
	data map[string]map[string]*engine.FlagDefinition // tenant -> flagKey -> definition
}

// New creates a Store and performs the initial load from the given path.
// path may be a YAML file or a directory of per-tenant YAML files.
func New(path string) (*Store, error) {
	s := &Store{path: path}
	if err := s.Reload(); err != nil {
		return nil, err
	}
	return s, nil
}

// GetFlag implements engine.FlagStore.
func (s *Store) GetFlag(tenantID, flagKey string) (*engine.FlagDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	flags, ok := s.data[tenantID]
	if !ok {
		return nil, nil
	}
	return flags[flagKey], nil
}

// ListFlagKeys implements engine.FlagStore.
func (s *Store) ListFlagKeys(tenantID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	flags, ok := s.data[tenantID]
	if !ok {
		return nil, nil
	}

	keys := make([]string, 0, len(flags))
	for k := range flags {
		keys = append(keys, k)
	}
	return keys, nil
}

// Reload re-reads the config from disk and atomically swaps the in-memory data.
func (s *Store) Reload() error {
	info, err := os.Stat(s.path)
	if err != nil {
		return fmt.Errorf("configfile: stat %s: %w", s.path, err)
	}

	var data map[string]map[string]*engine.FlagDefinition
	if info.IsDir() {
		data, err = loadDirectory(s.path)
	} else {
		data, err = loadFile(s.path)
	}
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.data = data
	s.mu.Unlock()
	return nil
}

// WatchSignal starts a goroutine that reloads the config on SIGHUP.
// It blocks until the context is cancelled.
func (s *Store) WatchSignal(ctx context.Context) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP)
	defer signal.Stop(ch)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			if err := s.Reload(); err != nil {
				log.Printf("configfile: reload on SIGHUP failed: %v", err)
			} else {
				log.Println("configfile: reloaded on SIGHUP")
			}
		}
	}
}

func loadFile(path string) (map[string]map[string]*engine.FlagDefinition, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("configfile: read %s: %w", path, err)
	}

	var mt multiTenantFile
	if err := yaml.Unmarshal(raw, &mt); err != nil {
		return nil, fmt.Errorf("configfile: parse %s: %w", path, err)
	}

	if len(mt.Tenants) > 0 {
		return buildIndex(mt.Tenants)
	}

	var st singleTenantFile
	if err := yaml.Unmarshal(raw, &st); err != nil {
		return nil, fmt.Errorf("configfile: parse single-tenant %s: %w", path, err)
	}

	return buildIndex(map[string]tenantBlock{
		DefaultTenant: {Flags: st.Flags},
	})
}

func loadDirectory(dir string) (map[string]map[string]*engine.FlagDefinition, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("configfile: read dir %s: %w", dir, err)
	}

	tenants := make(map[string]tenantBlock)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		tenantID := strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("configfile: read %s: %w", name, err)
		}

		var st singleTenantFile
		if err := yaml.Unmarshal(raw, &st); err != nil {
			return nil, fmt.Errorf("configfile: parse %s: %w", name, err)
		}

		tenants[tenantID] = tenantBlock{Flags: st.Flags}
	}

	return buildIndex(tenants)
}

func buildIndex(tenants map[string]tenantBlock) (map[string]map[string]*engine.FlagDefinition, error) {
	data := make(map[string]map[string]*engine.FlagDefinition, len(tenants))
	for tid, tb := range tenants {
		m := make(map[string]*engine.FlagDefinition, len(tb.Flags))
		for i := range tb.Flags {
			f := &tb.Flags[i]
			if f.Key == "" {
				return nil, fmt.Errorf("configfile: tenant %q has a flag with missing key", tid)
			}
			if f.Semantics == engine.SemanticsPersistent {
				log.Printf("configfile: tenant %q flag %q uses persistent semantics — will degrade without persistence backend", tid, f.Key)
			}
			m[f.Key] = f
		}
		data[tid] = m
	}
	return data, nil
}

// compile-time interface check
var _ engine.FlagStore = (*Store)(nil)
