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

// APIKeyEntry represents an API key defined in the config file.
type APIKeyEntry struct {
	Key   string `yaml:"key"`
	Scope string `yaml:"scope"`
	Name  string `yaml:"name"`
}

type tenantBlock struct {
	Flags   []engine.FlagDefinition `yaml:"flags"`
	APIKeys []APIKeyEntry           `yaml:"api_keys"`
}

// singleTenantFile is the top-level YAML structure for sidecar / single-tenant files.
type singleTenantFile struct {
	Flags   []engine.FlagDefinition `yaml:"flags"`
	APIKeys []APIKeyEntry           `yaml:"api_keys"`
}

// Store is a file-backed FlagStore that supports single-file, directory,
// and sidecar modes. It is safe for concurrent reads and supports hot reload.
type Store struct {
	path    string
	mu      sync.RWMutex
	data    map[string]map[string]*engine.FlagDefinition // tenant -> flagKey -> definition
	apiKeys map[string][]APIKeyEntry                     // tenant -> keys
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

// APIKeys returns the API keys defined for each tenant in the config file.
func (s *Store) APIKeys() map[string][]APIKeyEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string][]APIKeyEntry, len(s.apiKeys))
	for tid, keys := range s.apiKeys {
		copied := make([]APIKeyEntry, len(keys))
		copy(copied, keys)
		result[tid] = copied
	}
	return result
}

type loadResult struct {
	flags   map[string]map[string]*engine.FlagDefinition
	apiKeys map[string][]APIKeyEntry
}

// Reload re-reads the config from disk and atomically swaps the in-memory data.
func (s *Store) Reload() error {
	info, err := os.Stat(s.path)
	if err != nil {
		return fmt.Errorf("configfile: stat %s: %w", s.path, err)
	}

	var result *loadResult
	if info.IsDir() {
		result, err = loadDirectory(s.path)
	} else {
		result, err = loadFile(s.path)
	}
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.data = result.flags
	s.apiKeys = result.apiKeys
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

func loadFile(path string) (*loadResult, error) {
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
		DefaultTenant: {Flags: st.Flags, APIKeys: st.APIKeys},
	})
}

func loadDirectory(dir string) (*loadResult, error) {
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

		tenants[tenantID] = tenantBlock(st)
	}

	return buildIndex(tenants)
}

func buildIndex(tenants map[string]tenantBlock) (*loadResult, error) {
	data := make(map[string]map[string]*engine.FlagDefinition, len(tenants))
	keys := make(map[string][]APIKeyEntry, len(tenants))

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

		if len(tb.APIKeys) > 0 {
			keys[tid] = tb.APIKeys
		}
	}
	return &loadResult{flags: data, apiKeys: keys}, nil
}

// compile-time interface check
var _ engine.FlagStore = (*Store)(nil)
