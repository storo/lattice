package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/storo/lattice/pkg/agent"
	"github.com/storo/lattice/pkg/config"
	"github.com/storo/lattice/pkg/core"
	"github.com/storo/lattice/pkg/mesh"
	"github.com/storo/lattice/pkg/protocol/http"
	"github.com/storo/lattice/pkg/provider"
	"github.com/storo/lattice/pkg/security"
)

var serveAddr string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Lattice server",
	Long:  `Start an HTTP server exposing the agent mesh.`,
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", "", "listen address (overrides config)")
}

func runServe(cmd *cobra.Command, args []string) error {
	// Load config
	configPath := cfgFile
	if configPath == "" {
		configPath = "lattice.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		// Use default config if file doesn't exist
		if os.IsNotExist(err) {
			log.Printf("Config file not found, using defaults")
			cfg = config.DefaultConfig()
		} else {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Override addr if specified
	if serveAddr != "" {
		cfg.Server.Addr = serveAddr
	}

	// Create provider using factory
	llmProvider, err := config.NewProvider(cfg.Provider)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	log.Printf("Using %s provider", llmProvider.Name())

	// Create mesh
	var meshOpts []mesh.Option
	meshOpts = append(meshOpts, mesh.WithMaxHops(cfg.Mesh.MaxHops))

	switch cfg.Mesh.Balancer {
	case "random":
		meshOpts = append(meshOpts, mesh.WithBalancer(mesh.NewRandomBalancer(nil)))
	case "first":
		meshOpts = append(meshOpts, mesh.WithBalancer(mesh.NewFirstBalancer()))
	default:
		meshOpts = append(meshOpts, mesh.WithBalancer(mesh.NewRoundRobinBalancer()))
	}

	m := mesh.New(meshOpts...)

	// Create and register agents
	for _, agentCfg := range cfg.Agents {
		a := createAgentFromConfig(agentCfg, llmProvider)
		if err := m.Register(a); err != nil {
			return fmt.Errorf("failed to register agent %s: %w", agentCfg.Name, err)
		}
		log.Printf("Registered agent: %s", agentCfg.Name)
	}

	// Setup authentication
	apiKeyAuth := security.NewAPIKeyAuth()
	for _, key := range cfg.Auth.Keys {
		apiKeyAuth.RegisterKey(key.ID, &security.KeyEntry{
			AgentID:     key.ID,
			Roles:       key.Roles,
			Permissions: key.Permissions,
		})
	}
	auth := security.NewAuth(security.WithAPIKeyAuth(apiKeyAuth))

	// Create HTTP server
	server := http.NewServer(m, http.WithAuth(auth))

	// Start server in goroutine
	go func() {
		log.Printf("Lattice server starting on %s", cfg.Server.Addr)
		log.Println("Endpoints:")
		log.Println("  GET  /health        - Health check (no auth)")
		log.Println("  GET  /agents        - List agents")
		log.Println("  GET  /agents/{id}   - Get agent info")
		log.Println("  POST /agents/{id}/run - Run specific agent")
		log.Println("  POST /mesh/run      - Run on mesh (auto-select)")

		if err := server.ListenAndServe(cfg.Server.Addr); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	log.Println("Server stopped")
	return nil
}

func createAgentFromConfig(cfg config.AgentConfig, prov provider.Provider) core.Agent {
	builder := agent.New(cfg.Name).
		Model(prov).
		System(cfg.System)

	if cfg.Description != "" {
		builder.Description(cfg.Description)
	}

	for _, cap := range cfg.Provides {
		builder.Provides(core.Capability(cap))
	}

	for _, cap := range cfg.Needs {
		builder.Needs(core.Capability(cap))
	}

	return builder.Build()
}
