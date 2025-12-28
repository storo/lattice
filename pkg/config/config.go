package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the main configuration structure.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Mesh     MeshConfig     `yaml:"mesh"`
	Provider ProviderConfig `yaml:"provider"`
	Agents   []AgentConfig  `yaml:"agents"`
	Auth     AuthConfig     `yaml:"auth"`
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Addr string `yaml:"addr"`
}

// MeshConfig contains mesh settings.
type MeshConfig struct {
	MaxHops  int    `yaml:"max_hops"`
	Balancer string `yaml:"balancer"`
}

// ProviderConfig contains LLM provider settings.
type ProviderConfig struct {
	Type   string `yaml:"type"`
	APIKey string `yaml:"api_key"`
}

// AgentConfig defines an agent.
type AgentConfig struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	System      string   `yaml:"system"`
	Provides    []string `yaml:"provides"`
	Needs       []string `yaml:"needs"`
}

// AuthConfig contains authentication settings.
type AuthConfig struct {
	Keys []KeyConfig `yaml:"keys"`
}

// KeyConfig defines an API key.
type KeyConfig struct {
	ID          string   `yaml:"id"`
	Roles       []string `yaml:"roles"`
	Permissions []string `yaml:"permissions"`
}

// Load reads a configuration file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}
	if cfg.Mesh.MaxHops == 0 {
		cfg.Mesh.MaxHops = 10
	}
	if cfg.Mesh.Balancer == "" {
		cfg.Mesh.Balancer = "round-robin"
	}
	if cfg.Provider.Type == "" {
		cfg.Provider.Type = "mock"
	}

	// Get API key from environment if not in config
	if cfg.Provider.APIKey == "" {
		cfg.Provider.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	return &cfg, nil
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Addr: ":8080",
		},
		Mesh: MeshConfig{
			MaxHops:  10,
			Balancer: "round-robin",
		},
		Provider: ProviderConfig{
			Type: "mock",
		},
		Agents: []AgentConfig{
			{
				Name:     "assistant",
				System:   "You are a helpful assistant.",
				Provides: []string{"general"},
			},
		},
		Auth: AuthConfig{
			Keys: []KeyConfig{
				{
					ID:          "demo-key",
					Roles:       []string{"user"},
					Permissions: []string{"mesh:run", "agents:read"},
				},
			},
		},
	}
}

// Save writes configuration to a file.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
