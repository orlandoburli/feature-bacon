package auth

import "sync"

// MemKeyStore is an in-memory KeyStore backed by a map of hash -> APIKey.
// Used for config-file and env-var based key loading.
type MemKeyStore struct {
	mu   sync.RWMutex
	keys map[string]*APIKey // keyed by hash
}

func NewMemKeyStore() *MemKeyStore {
	return &MemKeyStore{keys: make(map[string]*APIKey)}
}

func (s *MemKeyStore) Add(key *APIKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[key.KeyHash] = key
}

func (s *MemKeyStore) LookupByHash(hash string) (*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.keys[hash], nil
}

var _ KeyStore = (*MemKeyStore)(nil)
