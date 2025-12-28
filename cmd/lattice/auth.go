package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/storo/lattice/pkg/config"
)

var (
	authRole    string
	authExpires string
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage API keys",
	Long:  `Generate, list, and revoke API keys for authentication.`,
}

var authGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new API key",
	RunE:  runAuthGenerate,
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all API keys",
	RunE:  runAuthList,
}

var authRevokeCmd = &cobra.Command{
	Use:   "revoke <key-id>",
	Short: "Revoke an API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthRevoke,
}

func init() {
	authGenerateCmd.Flags().StringVar(&authRole, "role", "user", "role for the key (user|admin)")
	authGenerateCmd.Flags().StringVar(&authExpires, "expires", "", "expiration (e.g., 7d, 30d) - not yet implemented")

	authCmd.AddCommand(authGenerateCmd)
	authCmd.AddCommand(authListCmd)
	authCmd.AddCommand(authRevokeCmd)
}

func runAuthGenerate(cmd *cobra.Command, args []string) error {
	// Load config
	configPath := cfgFile
	if configPath == "" {
		configPath = "lattice.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %s (run 'lattice config init' first)", configPath)
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Generate random key
	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}
	keyID := "lk-" + hex.EncodeToString(keyBytes)

	// Set permissions based on role
	var permissions []string
	switch authRole {
	case "admin":
		permissions = []string{"*"}
	case "user":
		permissions = []string{"mesh:run", "agents:read"}
	default:
		return fmt.Errorf("invalid role: %s (must be 'user' or 'admin')", authRole)
	}

	// Add to config
	cfg.Auth.Keys = append(cfg.Auth.Keys, config.KeyConfig{
		ID:          keyID,
		Roles:       []string{authRole},
		Permissions: permissions,
	})

	// Save config
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Generated new API key:\n")
	fmt.Printf("  Key ID: %s\n", keyID)
	fmt.Printf("  Role:   %s\n", authRole)
	fmt.Printf("  Permissions: %s\n", strings.Join(permissions, ", "))
	fmt.Printf("\nKey added to %s\n", configPath)
	fmt.Printf("\nUsage:\n")
	fmt.Printf("  lattice mesh run \"task\" --api-key %s\n", keyID)
	fmt.Printf("  curl -H \"X-API-Key: %s\" http://localhost:8080/agents\n", keyID)

	return nil
}

func runAuthList(cmd *cobra.Command, args []string) error {
	// Load config
	configPath := cfgFile
	if configPath == "" {
		configPath = "lattice.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %s", configPath)
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Auth.Keys) == 0 {
		fmt.Println("No API keys configured")
		return nil
	}

	if output == "json" {
		// Simple JSON output
		fmt.Println("[")
		for i, key := range cfg.Auth.Keys {
			comma := ","
			if i == len(cfg.Auth.Keys)-1 {
				comma = ""
			}
			fmt.Printf(`  {"id": "%s", "roles": ["%s"], "permissions": ["%s"]}%s`+"\n",
				maskKey(key.ID),
				strings.Join(key.Roles, "\", \""),
				strings.Join(key.Permissions, "\", \""),
				comma)
		}
		fmt.Println("]")
		return nil
	}

	fmt.Printf("%-40s  %-10s  %s\n", "KEY ID", "ROLE", "PERMISSIONS")
	fmt.Println("----------------------------------------  ----------  --------------------")
	for _, key := range cfg.Auth.Keys {
		roles := strings.Join(key.Roles, ", ")
		perms := strings.Join(key.Permissions, ", ")
		if len(perms) > 30 {
			perms = perms[:27] + "..."
		}
		fmt.Printf("%-40s  %-10s  %s\n", maskKey(key.ID), roles, perms)
	}

	return nil
}

func runAuthRevoke(cmd *cobra.Command, args []string) error {
	keyID := args[0]

	// Load config
	configPath := cfgFile
	if configPath == "" {
		configPath = "lattice.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %s", configPath)
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Find and remove key
	found := false
	newKeys := make([]config.KeyConfig, 0, len(cfg.Auth.Keys))
	for _, key := range cfg.Auth.Keys {
		if key.ID == keyID {
			found = true
			continue
		}
		newKeys = append(newKeys, key)
	}

	if !found {
		return fmt.Errorf("key not found: %s", keyID)
	}

	cfg.Auth.Keys = newKeys

	// Save config
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Revoked API key: %s\n", keyID)
	fmt.Printf("Updated config: %s\n", configPath)

	return nil
}

// maskKey masks the middle portion of a key for display
func maskKey(key string) string {
	if len(key) <= 10 {
		return key
	}
	return key[:6] + "..." + key[len(key)-4:]
}
