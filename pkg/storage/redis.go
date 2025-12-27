package storage

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements Store using Redis as the backend.
type RedisStore struct {
	client    *redis.Client
	keyPrefix string
}

// RedisOption configures the Redis store.
type RedisOption func(*redisConfig)

type redisConfig struct {
	password  string
	db        int
	keyPrefix string
}

// WithPassword sets the Redis password.
func WithPassword(password string) RedisOption {
	return func(c *redisConfig) {
		c.password = password
	}
}

// WithDB sets the Redis database number.
func WithDB(db int) RedisOption {
	return func(c *redisConfig) {
		c.db = db
	}
}

// WithKeyPrefix sets a prefix for all keys.
func WithKeyPrefix(prefix string) RedisOption {
	return func(c *redisConfig) {
		c.keyPrefix = prefix
	}
}

// NewRedisStore creates a new Redis-backed store.
func NewRedisStore(addr string, opts ...RedisOption) (*RedisStore, error) {
	cfg := &redisConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.password,
		DB:       cfg.db,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisStore{
		client:    client,
		keyPrefix: cfg.keyPrefix,
	}, nil
}

// prefixKey adds the configured prefix to a key.
func (s *RedisStore) prefixKey(key string) string {
	return s.keyPrefix + key
}

// Get retrieves a value by key.
func (s *RedisStore) Get(ctx context.Context, key string) ([]byte, error) {
	result, err := s.client.Get(ctx, s.prefixKey(key)).Bytes()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Set stores a value with an optional TTL.
func (s *RedisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return s.client.Set(ctx, s.prefixKey(key), value, ttl).Err()
}

// Delete removes a key from the store.
func (s *RedisStore) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, s.prefixKey(key)).Err()
}

// Exists checks if a key exists.
func (s *RedisStore) Exists(ctx context.Context, key string) bool {
	result, err := s.client.Exists(ctx, s.prefixKey(key)).Result()
	return err == nil && result > 0
}

// Keys returns all keys matching a pattern.
func (s *RedisStore) Keys(ctx context.Context, pattern string) ([]string, error) {
	// Use SCAN for production to avoid blocking
	var keys []string
	iter := s.client.Scan(ctx, 0, s.prefixKey(pattern), 0).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		// Remove prefix from returned keys
		if len(s.keyPrefix) > 0 && len(key) > len(s.keyPrefix) {
			key = key[len(s.keyPrefix):]
		}
		keys = append(keys, key)
	}

	if err := iter.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}

// Ping checks if the store is available.
func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// Close closes the Redis connection.
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// Verify RedisStore implements Store
var _ Store = (*RedisStore)(nil)
