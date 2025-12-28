package config

import (
	"fmt"

	"github.com/storo/lattice/pkg/provider"
	"github.com/storo/lattice/pkg/provider/anthropic"
	"github.com/storo/lattice/pkg/provider/ollama"
	"github.com/storo/lattice/pkg/storage"
)

// NewProvider creates a provider from configuration.
func NewProvider(cfg ProviderConfig) (provider.Provider, error) {
	switch cfg.Type {
	case "anthropic":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("anthropic provider requires api_key")
		}
		var opts []anthropic.Option
		if cfg.BaseURL != "" {
			opts = append(opts, anthropic.WithBaseURL(cfg.BaseURL))
		}
		if cfg.Model != "" {
			opts = append(opts, anthropic.WithModel(cfg.Model))
		}
		return anthropic.NewClient(cfg.APIKey, opts...), nil

	case "ollama":
		var opts []ollama.Option
		if cfg.BaseURL != "" {
			opts = append(opts, ollama.WithBaseURL(cfg.BaseURL))
		}
		if cfg.Model != "" {
			opts = append(opts, ollama.WithModel(cfg.Model))
		}
		return ollama.NewClient(opts...), nil

	case "mock", "":
		return provider.NewMock(), nil

	default:
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}
}

// NewStore creates a storage backend from configuration.
func NewStore(cfg StorageConfig) (storage.Store, error) {
	switch cfg.Type {
	case "memory", "":
		return storage.NewMemoryStore(), nil

	case "redis":
		if cfg.Address == "" {
			cfg.Address = "localhost:6379"
		}
		var opts []storage.RedisOption
		if cfg.Password != "" {
			opts = append(opts, storage.WithPassword(cfg.Password))
		}
		if cfg.DB != 0 {
			opts = append(opts, storage.WithDB(cfg.DB))
		}
		return storage.NewRedisStore(cfg.Address, opts...)

	case "sqlite":
		path := cfg.Path
		if path == "" {
			path = "./lattice.db"
		}
		return storage.NewSQLiteStore(path)

	default:
		return nil, fmt.Errorf("unknown storage type: %s", cfg.Type)
	}
}
