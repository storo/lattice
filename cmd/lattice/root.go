package main

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	cfgFile   string
	serverURL string
	apiKey    string
	output    string
)

var rootCmd = &cobra.Command{
	Use:   "lattice",
	Short: "Lattice - Agent Mesh Framework CLI",
	Long: `Lattice is a CLI for managing AI agent meshes.

Start a server, run agents, and interact with your mesh from the command line.`,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: lattice.yaml)")
	rootCmd.PersistentFlags().StringVar(&serverURL, "url", getEnv("LATTICE_URL", "http://localhost:8080"), "server URL")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", os.Getenv("LATTICE_API_KEY"), "API key for authentication")
	rootCmd.PersistentFlags().StringVar(&output, "output", "text", "output format (text, json)")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(meshCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(interactiveCmd)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
