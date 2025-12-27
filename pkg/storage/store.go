package storage

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a key is not found in the store.
var ErrNotFound = errors.New("key not found")

// Store is the interface for all storage backends.
type Store interface {
	// Get retrieves a value by key.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value with an optional TTL.
	// TTL of 0 means no expiration.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a key from the store.
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists.
	Exists(ctx context.Context, key string) bool

	// Keys returns all keys matching a pattern.
	// Supports basic glob patterns with * wildcard.
	Keys(ctx context.Context, pattern string) ([]string, error)

	// Ping checks if the store is available.
	Ping(ctx context.Context) error

	// Close closes the store connection.
	Close() error
}
