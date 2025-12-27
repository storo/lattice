package storage

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStore_SetAndGet(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	defer store.Close()

	key := "test-key"
	value := []byte("test-value")

	if err := store.Set(ctx, key, value, 0); err != nil {
		t.Fatalf("failed to set: %v", err)
	}

	got, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("expected %s, got %s", string(value), string(got))
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	defer store.Close()

	_, err := store.Get(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	defer store.Close()

	key := "test-key"
	value := []byte("test-value")

	store.Set(ctx, key, value, 0)

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	_, err := store.Get(ctx, key)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMemoryStore_TTL(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	defer store.Close()

	key := "test-key"
	value := []byte("test-value")
	ttl := 50 * time.Millisecond

	if err := store.Set(ctx, key, value, ttl); err != nil {
		t.Fatalf("failed to set: %v", err)
	}

	// Should exist immediately
	got, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("expected key to exist: %v", err)
	}
	if string(got) != string(value) {
		t.Errorf("expected %s, got %s", string(value), string(got))
	}

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)

	// Should be expired now
	_, err = store.Get(ctx, key)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after TTL, got %v", err)
	}
}

func TestMemoryStore_Overwrite(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	defer store.Close()

	key := "test-key"

	store.Set(ctx, key, []byte("value1"), 0)
	store.Set(ctx, key, []byte("value2"), 0)

	got, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	if string(got) != "value2" {
		t.Errorf("expected value2, got %s", string(got))
	}
}

func TestMemoryStore_Ping(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	defer store.Close()

	if err := store.Ping(ctx); err != nil {
		t.Errorf("expected ping to succeed, got %v", err)
	}
}

func TestMemoryStore_Close(t *testing.T) {
	store := NewMemoryStore()

	if err := store.Close(); err != nil {
		t.Errorf("expected close to succeed, got %v", err)
	}
}

func TestMemoryStore_Exists(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	defer store.Close()

	key := "test-key"
	value := []byte("test-value")

	// Should not exist initially
	if store.Exists(ctx, key) {
		t.Error("expected key to not exist initially")
	}

	store.Set(ctx, key, value, 0)

	// Should exist after set
	if !store.Exists(ctx, key) {
		t.Error("expected key to exist after set")
	}

	store.Delete(ctx, key)

	// Should not exist after delete
	if store.Exists(ctx, key) {
		t.Error("expected key to not exist after delete")
	}
}

func TestMemoryStore_Keys(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	defer store.Close()

	store.Set(ctx, "key1", []byte("value1"), 0)
	store.Set(ctx, "key2", []byte("value2"), 0)
	store.Set(ctx, "other", []byte("value3"), 0)

	keys, err := store.Keys(ctx, "key*")
	if err != nil {
		t.Fatalf("failed to get keys: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("expected 2 keys matching 'key*', got %d", len(keys))
	}

	// All keys
	allKeys, err := store.Keys(ctx, "*")
	if err != nil {
		t.Fatalf("failed to get all keys: %v", err)
	}

	if len(allKeys) != 3 {
		t.Errorf("expected 3 total keys, got %d", len(allKeys))
	}
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	defer store.Close()

	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			store.Set(ctx, "key", []byte("value"), 0)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			store.Get(ctx, "key")
		}
		done <- true
	}()

	<-done
	<-done
}

func TestMemoryStore_Keys_AdvancedPatterns(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	defer store.Close()

	// Set up Redis-like namespaced keys
	store.Set(ctx, "user:123:settings", []byte("v1"), 0)
	store.Set(ctx, "user:456:settings", []byte("v2"), 0)
	store.Set(ctx, "user:123:profile", []byte("v3"), 0)
	store.Set(ctx, "agent:abc:state", []byte("v4"), 0)

	tests := []struct {
		pattern  string
		expected int
	}{
		{"user:*:settings", 2}, // Wildcard in middle
		{"user:123:*", 2},      // Suffix wildcard
		{"*:settings", 2},      // Prefix wildcard
		{"user:*", 3},          // All user keys
		{"agent:*", 1},         // Agent keys
		{"*", 4},               // All keys
		{"nonexistent:*", 0},   // No match
	}

	for _, tt := range tests {
		keys, err := store.Keys(ctx, tt.pattern)
		if err != nil {
			t.Fatalf("pattern %s: failed: %v", tt.pattern, err)
		}
		if len(keys) != tt.expected {
			t.Errorf("pattern %s: expected %d keys, got %d (%v)",
				tt.pattern, tt.expected, len(keys), keys)
		}
	}
}

func TestMemoryStore_Keys_QuestionMark(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	defer store.Close()

	store.Set(ctx, "key1", []byte("v1"), 0)
	store.Set(ctx, "key2", []byte("v2"), 0)
	store.Set(ctx, "key10", []byte("v3"), 0)

	// ? matches single character
	keys, err := store.Keys(ctx, "key?")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys matching 'key?', got %d (%v)", len(keys), keys)
	}
}
