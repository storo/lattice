//go:build integration

package storage

import (
	"context"
	"os"
	"testing"
	"time"
)

func getRedisAddr() string {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		return "localhost:6379"
	}
	return addr
}

func TestRedisStore_SetGet(t *testing.T) {
	ctx := context.Background()

	store, err := NewRedisStore(getRedisAddr())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Set a value
	err = store.Set(ctx, "test-key", []byte("test-value"), 0)
	if err != nil {
		t.Fatalf("failed to set: %v", err)
	}

	// Get the value
	value, err := store.Get(ctx, "test-key")
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	if string(value) != "test-value" {
		t.Errorf("expected 'test-value', got '%s'", string(value))
	}

	// Cleanup
	store.Delete(ctx, "test-key")
}

func TestRedisStore_GetNotFound(t *testing.T) {
	ctx := context.Background()

	store, err := NewRedisStore(getRedisAddr())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	_, err = store.Get(ctx, "nonexistent-key")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRedisStore_TTL(t *testing.T) {
	ctx := context.Background()

	store, err := NewRedisStore(getRedisAddr())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Set with short TTL
	err = store.Set(ctx, "ttl-key", []byte("expires-soon"), 100*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to set: %v", err)
	}

	// Should exist immediately
	_, err = store.Get(ctx, "ttl-key")
	if err != nil {
		t.Fatalf("key should exist: %v", err)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be gone
	_, err = store.Get(ctx, "ttl-key")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after TTL, got %v", err)
	}
}

func TestRedisStore_Delete(t *testing.T) {
	ctx := context.Background()

	store, err := NewRedisStore(getRedisAddr())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Set a value
	store.Set(ctx, "delete-key", []byte("value"), 0)

	// Delete it
	err = store.Delete(ctx, "delete-key")
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// Should not exist
	_, err = store.Get(ctx, "delete-key")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestRedisStore_Exists(t *testing.T) {
	ctx := context.Background()

	store, err := NewRedisStore(getRedisAddr())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Should not exist
	if store.Exists(ctx, "exists-key") {
		t.Error("key should not exist")
	}

	// Set it
	store.Set(ctx, "exists-key", []byte("value"), 0)

	// Should exist
	if !store.Exists(ctx, "exists-key") {
		t.Error("key should exist")
	}

	// Cleanup
	store.Delete(ctx, "exists-key")
}

func TestRedisStore_Keys(t *testing.T) {
	ctx := context.Background()

	store, err := NewRedisStore(getRedisAddr(), WithKeyPrefix("test:"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Set multiple keys
	store.Set(ctx, "user:1", []byte("alice"), 0)
	store.Set(ctx, "user:2", []byte("bob"), 0)
	store.Set(ctx, "session:1", []byte("data"), 0)

	// Find user keys
	keys, err := store.Keys(ctx, "user:*")
	if err != nil {
		t.Fatalf("failed to list keys: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}

	// Cleanup
	store.Delete(ctx, "user:1")
	store.Delete(ctx, "user:2")
	store.Delete(ctx, "session:1")
}

func TestRedisStore_Ping(t *testing.T) {
	ctx := context.Background()

	store, err := NewRedisStore(getRedisAddr())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	err = store.Ping(ctx)
	if err != nil {
		t.Errorf("ping failed: %v", err)
	}
}

func TestRedisStore_WithPassword(t *testing.T) {
	// This test requires a Redis with password
	// Skip if no password env var
	password := os.Getenv("REDIS_PASSWORD")
	if password == "" {
		t.Skip("REDIS_PASSWORD not set")
	}

	ctx := context.Background()

	store, err := NewRedisStore(getRedisAddr(), WithPassword(password))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	err = store.Ping(ctx)
	if err != nil {
		t.Errorf("ping failed: %v", err)
	}
}

func TestRedisStore_WithDB(t *testing.T) {
	ctx := context.Background()

	store, err := NewRedisStore(getRedisAddr(), WithDB(1))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Set in DB 1
	store.Set(ctx, "db-test", []byte("value"), 0)

	// Should exist in DB 1
	_, err = store.Get(ctx, "db-test")
	if err != nil {
		t.Errorf("key should exist in DB 1: %v", err)
	}

	// Cleanup
	store.Delete(ctx, "db-test")
}
