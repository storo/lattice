package security

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

// Errors
var (
	ErrInvalidAPIKey = errors.New("invalid API key")
	ErrExpiredAPIKey = errors.New("API key expired")
)

// APIKeyAuth provides API key authentication with timing-safe comparison.
type APIKeyAuth struct {
	mu   sync.RWMutex
	keys map[string]*KeyEntry // hash(key) -> entry
}

// KeyEntry stores information about a registered API key.
type KeyEntry struct {
	// AgentID is the identifier of the agent this key authenticates.
	AgentID string

	// Roles are the roles granted to this key.
	Roles []string

	// Permissions are specific permissions for this key.
	Permissions []string

	// ExpiresAt is the Unix timestamp when this key expires.
	// 0 means the key never expires.
	ExpiresAt int64
}

// Claims are the authenticated claims from a valid API key.
type Claims struct {
	// AgentID is the authenticated agent's identifier.
	AgentID string

	// Roles are the roles granted to this agent.
	Roles []string

	// Permissions are specific permissions.
	Permissions []string

	// ExpiresAt is when the key expires (0 = never).
	ExpiresAt int64
}

// NewAPIKeyAuth creates a new API key authenticator.
func NewAPIKeyAuth() *APIKeyAuth {
	return &APIKeyAuth{
		keys: make(map[string]*KeyEntry),
	}
}

// RegisterKey registers an API key with the given entry.
// The key is hashed before storage for security.
func (a *APIKeyAuth) RegisterKey(key string, entry *KeyEntry) {
	a.mu.Lock()
	defer a.mu.Unlock()

	hash := hashKey(key)
	a.keys[hash] = entry
}

// RevokeKey removes an API key from the registry.
func (a *APIKeyAuth) RevokeKey(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	hash := hashKey(key)
	delete(a.keys, hash)
}

// Authenticate validates an API key and returns claims if valid.
// Uses constant-time comparison to prevent timing attacks.
func (a *APIKeyAuth) Authenticate(ctx context.Context, key string) (*Claims, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if key == "" {
		// Still do the work to prevent timing attacks on empty keys
		dummyCompare()
		return nil, ErrInvalidAPIKey
	}

	keyHash := hashKey(key)

	// CRITICAL: Iterate over ALL keys for constant-time behavior.
	// A direct map lookup would reveal if a hash exists via timing.
	var matchedEntry *KeyEntry

	for storedHash, entry := range a.keys {
		// Use constant-time comparison
		if constantTimeEqual(keyHash, storedHash) {
			matchedEntry = entry
			// DO NOT break - continue iterating for constant time
		}
	}

	if matchedEntry == nil {
		// Simulate work to maintain constant timing
		dummyCompare()
		return nil, ErrInvalidAPIKey
	}

	// Check expiration
	if matchedEntry.ExpiresAt > 0 {
		now := time.Now().Unix()
		if now > matchedEntry.ExpiresAt {
			return nil, ErrExpiredAPIKey
		}
	}

	return &Claims{
		AgentID:     matchedEntry.AgentID,
		Roles:       matchedEntry.Roles,
		Permissions: matchedEntry.Permissions,
		ExpiresAt:   matchedEntry.ExpiresAt,
	}, nil
}

// hashKey generates a SHA-256 hash of the key.
func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// constantTimeEqual compares two strings in constant time.
func constantTimeEqual(a, b string) bool {
	// Ensure same length for subtle.ConstantTimeCompare
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// dummyCompare simulates comparison work to maintain constant timing.
func dummyCompare() {
	dummy := "0000000000000000000000000000000000000000000000000000000000000000"
	constantTimeEqual(dummy, dummy)
}
