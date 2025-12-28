package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteStore_SetAndGet(t *testing.T) {
	ctx := context.Background()
	store := newTestSQLiteStore(t)
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

func TestSQLiteStore_GetNotFound(t *testing.T) {
	ctx := context.Background()
	store := newTestSQLiteStore(t)
	defer store.Close()

	_, err := store.Get(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSQLiteStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := newTestSQLiteStore(t)
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

func TestSQLiteStore_TTL(t *testing.T) {
	ctx := context.Background()
	store := newTestSQLiteStore(t)
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
	time.Sleep(100 * time.Millisecond)

	// Should be expired now
	_, err = store.Get(ctx, key)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after TTL, got %v", err)
	}
}

func TestSQLiteStore_Overwrite(t *testing.T) {
	ctx := context.Background()
	store := newTestSQLiteStore(t)
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

func TestSQLiteStore_Ping(t *testing.T) {
	ctx := context.Background()
	store := newTestSQLiteStore(t)
	defer store.Close()

	if err := store.Ping(ctx); err != nil {
		t.Errorf("expected ping to succeed, got %v", err)
	}
}

func TestSQLiteStore_Close(t *testing.T) {
	store := newTestSQLiteStore(t)

	if err := store.Close(); err != nil {
		t.Errorf("expected close to succeed, got %v", err)
	}
}

func TestSQLiteStore_Exists(t *testing.T) {
	ctx := context.Background()
	store := newTestSQLiteStore(t)
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

func TestSQLiteStore_Keys(t *testing.T) {
	ctx := context.Background()
	store := newTestSQLiteStore(t)
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

func TestSQLiteStore_Keys_ExcludesExpired(t *testing.T) {
	ctx := context.Background()
	store := newTestSQLiteStore(t)
	defer store.Close()

	// Set one permanent and one expiring
	store.Set(ctx, "permanent", []byte("a"), 0)
	store.Set(ctx, "expiring", []byte("b"), 50*time.Millisecond)

	// Both should be visible initially
	keys, err := store.Keys(ctx, "*")
	if err != nil {
		t.Fatalf("failed to get keys: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys initially, got %d", len(keys))
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Only permanent should be visible
	keys, err = store.Keys(ctx, "*")
	if err != nil {
		t.Fatalf("failed to get keys: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("expected 1 key after expiration, got %d", len(keys))
	}
}

func TestSQLiteStore_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	store := newTestSQLiteStore(t)
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

func TestSQLiteStore_Persistence(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create store and add data
	store1, err := NewSQLiteStore(dbPath, WithCleanupInterval(time.Hour))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	if err := store1.Set(ctx, "persistent", []byte("data"), 0); err != nil {
		t.Fatalf("failed to set: %v", err)
	}
	store1.Close()

	// Open again and verify data persists
	store2, err := NewSQLiteStore(dbPath, WithCleanupInterval(time.Hour))
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	defer store2.Close()

	value, err := store2.Get(ctx, "persistent")
	if err != nil {
		t.Fatalf("failed to get after reopen: %v", err)
	}
	if string(value) != "data" {
		t.Errorf("expected 'data', got '%s'", string(value))
	}
}

func TestSQLiteStore_BinaryData(t *testing.T) {
	ctx := context.Background()
	store := newTestSQLiteStore(t)
	defer store.Close()

	// Binary data with null bytes
	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0x00, 0x10}
	if err := store.Set(ctx, "binary", binaryData, 0); err != nil {
		t.Fatalf("failed to set: %v", err)
	}

	value, err := store.Get(ctx, "binary")
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if len(value) != len(binaryData) {
		t.Errorf("expected %d bytes, got %d", len(binaryData), len(value))
	}
	for i := range binaryData {
		if value[i] != binaryData[i] {
			t.Errorf("byte %d: expected %x, got %x", i, binaryData[i], value[i])
		}
	}
}

func TestSQLiteStore_Stats(t *testing.T) {
	ctx := context.Background()
	store := newTestSQLiteStore(t)
	defer store.Close()

	// Set some keys
	store.Set(ctx, "key1", []byte("a"), 0)
	store.Set(ctx, "key2", []byte("b"), 0)
	store.Set(ctx, "expiring", []byte("c"), 50*time.Millisecond)

	// Check stats
	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if stats["total_keys"] != 3 {
		t.Errorf("expected 3 total keys, got %d", stats["total_keys"])
	}
	if stats["active_keys"] != 3 {
		t.Errorf("expected 3 active keys, got %d", stats["active_keys"])
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	stats, err = store.Stats(ctx)
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if stats["active_keys"] != 2 {
		t.Errorf("expected 2 active keys after expiration, got %d", stats["active_keys"])
	}
	if stats["expired_keys"] != 1 {
		t.Errorf("expected 1 expired key, got %d", stats["expired_keys"])
	}
}

func TestSQLiteStore_WALMode(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "wal_test.db")

	store, err := NewSQLiteStore(dbPath, WithWALMode(), WithCleanupInterval(time.Hour))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Verify WAL mode
	var journalMode string
	if err := store.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("failed to query journal mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("expected 'wal' journal mode, got '%s'", journalMode)
	}
}

func TestSQLiteStore_Keys_AdvancedPatterns(t *testing.T) {
	ctx := context.Background()
	store := newTestSQLiteStore(t)
	defer store.Close()

	// Set up namespaced keys
	store.Set(ctx, "user:123:settings", []byte("v1"), 0)
	store.Set(ctx, "user:456:settings", []byte("v2"), 0)
	store.Set(ctx, "user:123:profile", []byte("v3"), 0)
	store.Set(ctx, "agent:abc:state", []byte("v4"), 0)

	tests := []struct {
		pattern  string
		expected int
	}{
		{"user:*:settings", 2},
		{"user:123:*", 2},
		{"user:*", 3},
		{"agent:*", 1},
		{"*", 4},
		{"nonexistent:*", 0},
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

func TestSQLiteStore_FileCreation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "new.db")

	// File should not exist yet
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatal("expected file to not exist")
	}

	// Create store (should create file)
	store, err := NewSQLiteStore(dbPath, WithCleanupInterval(time.Hour))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// File should exist now
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected file to exist after creating store")
	}
}

// newTestSQLiteStore creates an in-memory SQLite store for testing.
func newTestSQLiteStore(t *testing.T) *SQLiteStore {
	store, err := NewSQLiteStore(":memory:", WithCleanupInterval(time.Hour))
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	return store
}
