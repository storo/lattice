package storage

import (
	"context"
	"path"
	"sync"
	"time"
)

// MemoryStore is an in-memory implementation of Store.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]entry
}

type entry struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]entry),
	}
}

// Get retrieves a value by key.
func (s *MemoryStore) Get(ctx context.Context, key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.data[key]
	if !ok {
		return nil, ErrNotFound
	}

	// Check expiration
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		return nil, ErrNotFound
	}

	// Return a copy to prevent mutation
	result := make([]byte, len(e.value))
	copy(result, e.value)
	return result, nil
}

// Set stores a value with an optional TTL.
func (s *MemoryStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	// Store a copy to prevent mutation
	storedValue := make([]byte, len(value))
	copy(storedValue, value)

	s.data[key] = entry{
		value:     storedValue,
		expiresAt: expiresAt,
	}

	return nil
}

// Delete removes a key from the store.
func (s *MemoryStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	return nil
}

// Exists checks if a key exists and is not expired.
func (s *MemoryStore) Exists(ctx context.Context, key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.data[key]
	if !ok {
		return false
	}

	// Check expiration
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		return false
	}

	return true
}

// Keys returns all keys matching a pattern.
func (s *MemoryStore) Keys(ctx context.Context, pattern string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []string
	now := time.Now()

	for key, e := range s.data {
		// Skip expired entries
		if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
			continue
		}

		if matchPattern(pattern, key) {
			result = append(result, key)
		}
	}

	return result, nil
}

// Ping checks if the store is available.
func (s *MemoryStore) Ping(ctx context.Context) error {
	return nil
}

// Close closes the store.
func (s *MemoryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string]entry)
	return nil
}

// matchPattern checks if a key matches a glob pattern.
// Uses path.Match which supports:
// - * matches any sequence of non-separator characters
// - ? matches any single character
// - [abc] matches one character from the set
// - [a-z] matches one character from the range
func matchPattern(pattern, key string) bool {
	matched, err := path.Match(pattern, key)
	if err != nil {
		// Invalid pattern, fall back to exact match
		return pattern == key
	}
	return matched
}
