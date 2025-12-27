// Package main demonstrates a complete Lattice server with authentication.
//
// Usage:
//
//	# Set your Anthropic API key
//	export ANTHROPIC_API_KEY=sk-...
//
//	# Run the server
//	go run examples/server/main.go
//
//	# Test endpoints
//	curl http://localhost:8080/health
//	curl -H "X-API-Key: demo-key" http://localhost:8080/agents
//	curl -X POST -H "X-API-Key: demo-key" -H "Content-Type: application/json" \
//	  -d '{"input":"What is 2+2?"}' http://localhost:8080/mesh/run
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/storo/lattice/pkg/agent"
	"github.com/storo/lattice/pkg/core"
	"github.com/storo/lattice/pkg/mesh"
	"github.com/storo/lattice/pkg/middleware"
	"github.com/storo/lattice/pkg/protocol/http"
	"github.com/storo/lattice/pkg/provider"
	"github.com/storo/lattice/pkg/provider/anthropic"
	"github.com/storo/lattice/pkg/security"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")

	// Create provider (real or mock)
	var llmProvider provider.Provider
	if apiKey != "" {
		log.Println("Using Anthropic API")
		llmProvider = anthropic.NewClient(apiKey)
	} else {
		log.Println("ANTHROPIC_API_KEY not set, using mock provider")
		llmProvider = provider.NewMockWithResponse("This is a mock response. Set ANTHROPIC_API_KEY for real responses.")
	}

	// Create a metrics collector for observability
	metrics := middleware.NewMetricsCollector()

	// Create agents
	researcher := createAgent("researcher",
		"You are a research expert. Find and summarize information on any topic.",
		llmProvider,
		metrics,
		core.CapResearch,
	)

	writer := createAgent("writer",
		"You are a skilled writer. Create engaging content based on the given topic.",
		llmProvider,
		metrics,
		core.CapWriting,
	)

	coder := createAgent("coder",
		"You are a programming expert. Write clean, well-documented code.",
		llmProvider,
		metrics,
		core.CapCoding,
	)

	// Create mesh and register agents
	m := mesh.New(mesh.WithMaxHops(5))
	if err := m.Register(researcher, writer, coder); err != nil {
		log.Fatalf("Failed to register agents: %v", err)
	}

	log.Printf("Registered %d agents", 3)

	// Setup authentication
	apiKeyAuth := security.NewAPIKeyAuth()
	apiKeyAuth.RegisterKey("demo-key", &security.KeyEntry{
		AgentID:     "demo-user",
		Roles:       []string{"user"},
		Permissions: []string{"mesh:run", "agents:read"},
	})
	apiKeyAuth.RegisterKey("admin-key", &security.KeyEntry{
		AgentID:     "admin",
		Roles:       []string{"admin"},
		Permissions: []string{"*"},
	})

	auth := security.NewAuth(
		security.WithAPIKeyAuth(apiKeyAuth),
	)

	// Create HTTP server
	server := http.NewServer(m, http.WithAuth(auth))

	// Start server in goroutine
	addr := getEnv("ADDR", ":8080")
	go func() {
		log.Printf("Server starting on %s", addr)
		log.Println("Endpoints:")
		log.Println("  GET  /health           - Health check (no auth)")
		log.Println("  GET  /agents           - List agents")
		log.Println("  GET  /agents/{id}      - Get agent info")
		log.Println("  POST /agents/{id}/run  - Run specific agent")
		log.Println("  POST /mesh/run         - Run on mesh (auto-select)")
		log.Println("")
		log.Println("Auth: X-API-Key header with 'demo-key' or 'admin-key'")

		if err := server.ListenAndServe(addr); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Start metrics reporter
	go reportMetrics(metrics)

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

	// Print final metrics
	printMetricsSummary(metrics)

	log.Println("Server stopped")
}

// createAgent creates an agent with middleware.
func createAgent(
	name, system string,
	prov provider.Provider,
	metrics *middleware.MetricsCollector,
	caps ...core.Capability,
) core.Agent {
	a := agent.New(name).
		Model(prov).
		System(system).
		Provides(caps...).
		Build()

	// Wrap with metrics
	return middleware.WrapWithMetrics(a, metrics)
}

// reportMetrics periodically logs metrics.
func reportMetrics(collector *middleware.MetricsCollector) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		summary := collector.Summary()
		if summary.TotalCalls > 0 {
			log.Printf("Metrics: calls=%d success_rate=%.2f%% avg_duration=%s",
				summary.TotalCalls,
				summary.SuccessRate*100,
				summary.AvgDuration,
			)
		}
	}
}

// printMetricsSummary prints the final metrics.
func printMetricsSummary(collector *middleware.MetricsCollector) {
	fmt.Println("\n=== Final Metrics ===")

	summary := collector.Summary()
	fmt.Printf("Total Calls: %d\n", summary.TotalCalls)
	fmt.Printf("Success Rate: %.2f%%\n", summary.SuccessRate*100)
	fmt.Printf("Average Duration: %s\n", summary.AvgDuration)

	fmt.Println("\nPer-Agent Metrics:")
	for _, m := range collector.AllMetrics() {
		data, _ := json.MarshalIndent(map[string]any{
			"agent":         m.AgentName,
			"total_calls":   m.TotalCalls,
			"success_count": m.SuccessCount,
			"error_count":   m.ErrorCount,
			"avg_duration":  m.AvgDuration.String(),
		}, "", "  ")
		fmt.Println(string(data))
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
