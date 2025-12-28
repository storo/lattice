package storage

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"strings"
	"time"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// SQLiteStore is a SQLite-based implementation of Store.
// It provides persistent storage without external dependencies (no Docker required).
type SQLiteStore struct {
	db              *sql.DB
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// SQLiteOption configures the SQLite store.
type SQLiteOption func(*sqliteConfig)

type sqliteConfig struct {
	cleanupInterval time.Duration
	pragmas         map[string]string
}

// WithCleanupInterval sets how often expired entries are cleaned up.
func WithCleanupInterval(d time.Duration) SQLiteOption {
	return func(c *sqliteConfig) {
		c.cleanupInterval = d
	}
}

// WithWALMode enables Write-Ahead Logging for better concurrent performance.
func WithWALMode() SQLiteOption {
	return func(c *sqliteConfig) {
		c.pragmas["journal_mode"] = "WAL"
	}
}

// WithSyncMode sets the synchronous pragma (NORMAL, FULL, OFF).
func WithSyncMode(mode string) SQLiteOption {
	return func(c *sqliteConfig) {
		c.pragmas["synchronous"] = mode
	}
}

// NewSQLiteStore creates a new SQLite-based store.
// Path can be a file path or ":memory:" for in-memory database.
func NewSQLiteStore(dbPath string, opts ...SQLiteOption) (*SQLiteStore, error) {
	cfg := &sqliteConfig{
		cleanupInterval: 5 * time.Minute,
		pragmas: map[string]string{
			"journal_mode": "WAL",
			"synchronous":  "NORMAL",
			"busy_timeout": "5000",
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Apply pragmas
	for k, v := range cfg.pragmas {
		if _, err := db.Exec(fmt.Sprintf("PRAGMA %s = %s", k, v)); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma %s: %w", k, err)
		}
	}

	// Create schema
	schema := `
		CREATE TABLE IF NOT EXISTS kv (
			key TEXT PRIMARY KEY,
			value BLOB NOT NULL,
			expires_at INTEGER
		);
		CREATE INDEX IF NOT EXISTS idx_expires_at ON kv(expires_at) WHERE expires_at IS NOT NULL;
	`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	s := &SQLiteStore{
		db:              db,
		cleanupInterval: cfg.cleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	// Start background cleanup goroutine
	go s.cleanupLoop()

	return s, nil
}

// Get retrieves a value by key.
func (s *SQLiteStore) Get(ctx context.Context, key string) ([]byte, error) {
	var value []byte
	var expiresAt sql.NullInt64

	err := s.db.QueryRowContext(ctx,
		"SELECT value, expires_at FROM kv WHERE key = ?", key,
	).Scan(&value, &expiresAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	// Check expiration (using milliseconds for precision)
	if expiresAt.Valid && time.Now().UnixMilli() > expiresAt.Int64 {
		// Delete expired entry
		go s.Delete(context.Background(), key)
		return nil, ErrNotFound
	}

	return value, nil
}

// Set stores a value with an optional TTL.
func (s *SQLiteStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	var expiresAt sql.NullInt64
	if ttl > 0 {
		expiresAt = sql.NullInt64{Int64: time.Now().Add(ttl).UnixMilli(), Valid: true}
	}

	_, err := s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO kv (key, value, expires_at) VALUES (?, ?, ?)",
		key, value, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("failed to set key: %w", err)
	}

	return nil
}

// Delete removes a key from the store.
func (s *SQLiteStore) Delete(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM kv WHERE key = ?", key)
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}
	return nil
}

// Exists checks if a key exists and is not expired.
func (s *SQLiteStore) Exists(ctx context.Context, key string) bool {
	var expiresAt sql.NullInt64

	err := s.db.QueryRowContext(ctx,
		"SELECT expires_at FROM kv WHERE key = ?", key,
	).Scan(&expiresAt)

	if err != nil {
		return false
	}

	// Check expiration (using milliseconds for precision)
	if expiresAt.Valid && time.Now().UnixMilli() > expiresAt.Int64 {
		return false
	}

	return true
}

// Keys returns all keys matching a pattern.
// Supports basic glob patterns with * wildcard.
func (s *SQLiteStore) Keys(ctx context.Context, pattern string) ([]string, error) {
	now := time.Now().UnixMilli()

	// SQLite GLOB uses different syntax than path.Match
	// Convert * to % for SQL LIKE, but we'll use path.Match for accurate matching
	rows, err := s.db.QueryContext(ctx,
		"SELECT key FROM kv WHERE (expires_at IS NULL OR expires_at > ?)", now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query keys: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan key: %w", err)
		}

		// Use path.Match for consistent pattern matching with MemoryStore
		if matchPattern(pattern, key) {
			keys = append(keys, key)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return keys, nil
}

// Ping checks if the store is available.
func (s *SQLiteStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close closes the store and stops the cleanup goroutine.
func (s *SQLiteStore) Close() error {
	close(s.stopCleanup)
	return s.db.Close()
}

// cleanupLoop periodically removes expired entries.
func (s *SQLiteStore) cleanupLoop() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanup()
		case <-s.stopCleanup:
			return
		}
	}
}

// cleanup removes expired entries from the database.
func (s *SQLiteStore) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, _ = s.db.ExecContext(ctx,
		"DELETE FROM kv WHERE expires_at IS NOT NULL AND expires_at < ?",
		time.Now().UnixMilli(),
	)
}

// Stats returns statistics about the store.
func (s *SQLiteStore) Stats(ctx context.Context) (map[string]int64, error) {
	stats := make(map[string]int64)

	// Total keys
	var total int64
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM kv").Scan(&total)
	if err != nil {
		return nil, err
	}
	stats["total_keys"] = total

	// Active (non-expired) keys
	var active int64
	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM kv WHERE expires_at IS NULL OR expires_at > ?",
		time.Now().UnixMilli(),
	).Scan(&active)
	if err != nil {
		return nil, err
	}
	stats["active_keys"] = active

	// Expired keys (pending cleanup)
	stats["expired_keys"] = total - active

	return stats, nil
}

// globToSQLPattern converts a glob pattern to SQL LIKE pattern.
// This is used for optimized queries when possible.
func globToSQLPattern(pattern string) string {
	// Simple conversion: * -> %, ? -> _
	result := strings.ReplaceAll(pattern, "*", "%")
	result = strings.ReplaceAll(result, "?", "_")
	return result
}

// matchPattern checks if a key matches a glob pattern.
// Uses path.Match for consistency with MemoryStore.
func matchPatternLocal(pattern, key string) bool {
	matched, err := path.Match(pattern, key)
	if err != nil {
		return pattern == key
	}
	return matched
}

// Compile-time check that SQLiteStore implements Store.
var _ Store = (*SQLiteStore)(nil)
