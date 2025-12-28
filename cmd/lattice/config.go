package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/storo/lattice/pkg/config"
)

var configTemplate string

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  `Initialize and validate configuration files.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new configuration file",
	RunE:  runConfigInit,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate a configuration file",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runConfigValidate,
}

func init() {
	configInitCmd.Flags().StringVar(&configTemplate, "template", "basic", "template type (basic|full)")
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	filename := "lattice.yaml"
	if cfgFile != "" {
		filename = cfgFile
	}

	// Check if file already exists
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("file %s already exists", filename)
	}

	var cfg *config.Config
	if configTemplate == "full" {
		cfg = fullConfig()
	} else {
		cfg = config.DefaultConfig()
	}

	if err := cfg.Save(filename); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Configuration file created: %s\n", filename)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Set ANTHROPIC_API_KEY environment variable")
	fmt.Println("  2. Edit the config file to add your agents")
	fmt.Println("  3. Run: lattice serve")

	return nil
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	filename := "lattice.yaml"
	if len(args) > 0 {
		filename = args[0]
	} else if cfgFile != "" {
		filename = cfgFile
	}

	cfg, err := config.Load(filename)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Validate required fields
	var warnings []string

	if cfg.Server.Addr == "" {
		warnings = append(warnings, "server.addr is empty, will use default :8080")
	}

	if cfg.Provider.Type == "anthropic" && cfg.Provider.APIKey == "" && os.Getenv("ANTHROPIC_API_KEY") == "" {
		warnings = append(warnings, "anthropic provider requires ANTHROPIC_API_KEY")
	}

	if len(cfg.Agents) == 0 {
		warnings = append(warnings, "no agents defined")
	}

	for i, agent := range cfg.Agents {
		if agent.Name == "" {
			warnings = append(warnings, fmt.Sprintf("agent[%d]: name is required", i))
		}
		if len(agent.Provides) == 0 {
			warnings = append(warnings, fmt.Sprintf("agent[%d] (%s): no capabilities provided", i, agent.Name))
		}
	}

	if len(cfg.Auth.Keys) == 0 {
		warnings = append(warnings, "no API keys defined, server will be open")
	}

	fmt.Printf("Configuration file: %s\n", filename)
	fmt.Printf("Server address: %s\n", cfg.Server.Addr)
	fmt.Printf("Provider: %s\n", cfg.Provider.Type)
	fmt.Printf("Agents: %d\n", len(cfg.Agents))
	fmt.Printf("API Keys: %d\n", len(cfg.Auth.Keys))

	if len(warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range warnings {
			fmt.Printf("  - %s\n", w)
		}
	} else {
		fmt.Println("\nConfiguration is valid!")
	}

	return nil
}

func fullConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Addr: ":8080",
		},
		Mesh: config.MeshConfig{
			MaxHops:  10,
			Balancer: "round-robin",
		},
		Provider: config.ProviderConfig{
			Type: "anthropic",
		},
		Agents: []config.AgentConfig{
			{
				Name:        "researcher",
				Description: "Expert at researching topics and finding information",
				System:      "You are a research expert. Find accurate information and cite sources when possible.",
				Provides:    []string{"research"},
			},
			{
				Name:        "writer",
				Description: "Skilled writer that creates clear, engaging content",
				System:      "You are a skilled writer. Create clear, engaging content based on provided information.",
				Provides:    []string{"writing"},
				Needs:       []string{"research"},
			},
			{
				Name:        "coder",
				Description: "Expert programmer in multiple languages",
				System:      "You are an expert programmer. Write clean, efficient code with good documentation.",
				Provides:    []string{"coding"},
			},
			{
				Name:        "reviewer",
				Description: "Code reviewer that ensures quality and best practices",
				System:      "You are a code reviewer. Check for bugs, security issues, and suggest improvements.",
				Provides:    []string{"review"},
				Needs:       []string{"coding"},
			},
		},
		Auth: config.AuthConfig{
			Keys: []config.KeyConfig{
				{
					ID:          "admin-key",
					Roles:       []string{"admin"},
					Permissions: []string{"*"},
				},
				{
					ID:          "user-key",
					Roles:       []string{"user"},
					Permissions: []string{"mesh:run", "agents:read"},
				},
			},
		},
	}
}
