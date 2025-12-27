package mesh

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/storo/lattice/pkg/agent"
	"github.com/storo/lattice/pkg/core"
	"github.com/storo/lattice/pkg/provider"
	"github.com/storo/lattice/pkg/registry"
)

func TestInjector_InjectTools(t *testing.T) {
	ctx := context.Background()
	reg := registry.NewLocal()

	// Create a researcher agent
	researcher := agent.New("researcher").
		Model(provider.NewMockWithResponse("research result")).
		Provides(core.CapResearch).
		Build()

	reg.Register(ctx, researcher)

	// Create an agent that needs research
	writer := agent.New("writer").
		Needs(core.CapResearch).
		Build()

	// Create injector
	injector := NewInjector(reg, NewRoundRobinBalancer(), NewCycleDetector(10))

	// Inject tools
	tools, err := injector.InjectTools(ctx, writer)
	if err != nil {
		t.Fatalf("failed to inject tools: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	tool := tools[0]
	if tool.Name() != "delegate_to_research" {
		t.Errorf("expected tool name 'delegate_to_research', got '%s'", tool.Name())
	}
}

func TestInjector_InjectTools_NoProviders(t *testing.T) {
	ctx := context.Background()
	reg := registry.NewLocal()

	// Create an agent that needs a capability no one provides
	writer := agent.New("writer").
		Needs(core.CapResearch).
		Build()

	injector := NewInjector(reg, NewRoundRobinBalancer(), NewCycleDetector(10))

	// Inject tools - should return empty list, not error
	tools, err := injector.InjectTools(ctx, writer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tools) != 0 {
		t.Errorf("expected 0 tools when no providers, got %d", len(tools))
	}
}

func TestAgentTool_Execute(t *testing.T) {
	ctx := context.Background()

	// Create a researcher agent
	researcher := agent.New("researcher").
		Model(provider.NewMockWithResponse("research result")).
		Provides(core.CapResearch).
		Build()

	// Create the agent tool
	tool := &AgentTool{
		capability:    core.CapResearch,
		providers:     []core.Agent{researcher},
		balancer:      NewFirstBalancer(),
		cycleDetector: NewCycleDetector(10),
	}

	// Execute the tool
	params := json.RawMessage(`{"task": "research AI"}`)
	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}

	if result != "research result" {
		t.Errorf("expected 'research result', got '%s'", result)
	}
}

func TestAgentTool_Execute_DetectsCycle(t *testing.T) {
	ctx := context.Background()

	// Create an agent
	researcher := agent.New("researcher").
		Model(provider.NewMockWithResponse("research result")).
		Build()

	// Add researcher to call chain (simulating it's already running)
	ctx = core.WithCallChain(ctx, researcher.ID())

	// Create the agent tool
	tool := &AgentTool{
		capability:    core.CapResearch,
		providers:     []core.Agent{researcher},
		balancer:      NewFirstBalancer(),
		cycleDetector: NewCycleDetector(10),
	}

	// Execute should fail with cycle detected
	params := json.RawMessage(`{"task": "research AI"}`)
	_, err := tool.Execute(ctx, params)
	if err == nil {
		t.Error("expected cycle detection error")
	}
}

func TestAgentTool_Schema(t *testing.T) {
	tool := &AgentTool{
		capability: core.CapResearch,
	}

	schema := tool.Schema()

	var parsed map[string]any
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	if parsed["type"] != "object" {
		t.Errorf("expected type object")
	}

	props, ok := parsed["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties")
	}

	if _, ok := props["task"]; !ok {
		t.Error("expected task property")
	}
}
