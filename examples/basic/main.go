// Package main demonstrates basic Lattice usage.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/storo/lettice"
	"github.com/storo/lettice/pkg/provider"
)

func main() {
	ctx := context.Background()

	// For this example, we use mock providers
	// In production, use: provider.NewAnthropic(apiKey)
	researchProvider := provider.NewMockWithResponse(
		"Research findings: AI agents are increasingly being used for complex task automation. " +
			"Key trends include multi-agent collaboration, tool use, and reasoning capabilities.",
	)

	writerProvider := provider.NewMockWithResponse(
		"# AI Agent Trends in 2024\n\n" +
			"AI agents are revolutionizing how we approach complex tasks. " +
			"This article explores the latest trends in agent technology.",
	)

	// Create agents with capabilities
	researcher := lattice.NewAgent("researcher").
		Model(researchProvider).
		System("You are a research expert. Find relevant information on the given topic.").
		Provides(lattice.CapResearch).
		Build()

	writer := lattice.NewAgent("writer").
		Model(writerProvider).
		System("You are a skilled writer. Create engaging content based on research.").
		Provides(lattice.CapWriting).
		Needs(lattice.CapResearch). // Writer can delegate to researcher
		Build()

	// Create mesh and register agents
	mesh := lattice.NewMesh(lattice.WithMaxHops(3))
	if err := mesh.Register(researcher, writer); err != nil {
		log.Fatalf("Failed to register agents: %v", err)
	}

	// Run a task
	fmt.Println("=== Running Research Agent ===")
	result, err := mesh.RunAgent(ctx, researcher.ID(), "Research AI agent trends")
	if err != nil {
		log.Fatalf("Research failed: %v", err)
	}
	fmt.Printf("Result: %s\n\n", result.Output)

	fmt.Println("=== Running Writer Agent ===")
	result, err = mesh.RunAgent(ctx, writer.ID(), "Write an article about AI agents")
	if err != nil {
		log.Fatalf("Writing failed: %v", err)
	}
	fmt.Printf("Result: %s\n\n", result.Output)

	fmt.Println("=== Using Mesh.Run (auto-select agent) ===")
	result, err = mesh.Run(ctx, "Find information about LLMs")
	if err != nil {
		log.Fatalf("Mesh run failed: %v", err)
	}
	fmt.Printf("Result: %s\n", result.Output)
}
