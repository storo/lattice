package mesh

import (
	"context"
	"testing"

	"github.com/storo/lattice/pkg/agent"
	"github.com/storo/lattice/pkg/core"
	"github.com/storo/lattice/pkg/provider"
)

func TestMesh_Register(t *testing.T) {
	m := New()

	a := agent.New("test-agent").Build()

	if err := m.Register(a); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	ctx := context.Background()
	found, err := m.GetAgent(ctx, a.ID())
	if err != nil {
		t.Fatalf("failed to get agent: %v", err)
	}
	if found.ID() != a.ID() {
		t.Errorf("expected ID %s, got %s", a.ID(), found.ID())
	}
}

func TestMesh_RegisterMultiple(t *testing.T) {
	m := New()

	a1 := agent.New("agent-1").Build()
	a2 := agent.New("agent-2").Build()

	if err := m.Register(a1, a2); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	ctx := context.Background()
	agents, err := m.ListAgents(ctx)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
}

func TestMesh_Run_SimpleAgent(t *testing.T) {
	ctx := context.Background()

	mockProvider := provider.NewMockWithResponse("Hello from agent!")

	m := New()

	a := agent.New("greeter").
		Model(mockProvider).
		Build()

	m.Register(a)

	result, err := m.RunAgent(ctx, a.ID(), "Say hello")
	if err != nil {
		t.Fatalf("failed to run: %v", err)
	}

	if result.Output != "Hello from agent!" {
		t.Errorf("expected 'Hello from agent!', got '%s'", result.Output)
	}
}

func TestMesh_Run_WithDelegation(t *testing.T) {
	ctx := context.Background()

	// Create researcher that provides research
	researcher := agent.New("researcher").
		Model(provider.NewMockWithResponse("Research findings: AI is advancing")).
		Provides(core.CapResearch).
		Build()

	// Create writer that needs research and will delegate
	writerCallCount := 0
	writerProvider := &provider.MockProvider{
		ChatFunc: func(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
			writerCallCount++
			if writerCallCount == 1 {
				// First call: use the delegate tool
				return &provider.ChatResponse{
					StopReason: provider.StopReasonToolUse,
					ToolCalls: []core.ToolCall{
						{
							ID:     "call-1",
							Name:   "delegate_to_research",
							Params: []byte(`{"task": "Find info about AI"}`),
						},
					},
				}, nil
			}
			// Second call: return final response
			return &provider.ChatResponse{
				Content:    "Article based on research",
				StopReason: provider.StopReasonEndTurn,
			}, nil
		},
	}

	writer := agent.New("writer").
		Model(writerProvider).
		Needs(core.CapResearch).
		Provides(core.CapWriting).
		Build()

	m := New()
	m.Register(researcher, writer)

	// Prepare writer with injected tools
	if err := m.PrepareAgent(ctx, writer); err != nil {
		t.Fatalf("failed to prepare: %v", err)
	}

	result, err := m.RunAgent(ctx, writer.ID(), "Write about AI")
	if err != nil {
		t.Fatalf("failed to run: %v", err)
	}

	if result.Output != "Article based on research" {
		t.Errorf("expected 'Article based on research', got '%s'", result.Output)
	}
}

func TestMesh_WithMaxHops(t *testing.T) {
	m := New(WithMaxHops(5))

	if m.cycleDetector.MaxHops() != 5 {
		t.Errorf("expected max hops 5, got %d", m.cycleDetector.MaxHops())
	}
}

func TestMesh_FindProviders(t *testing.T) {
	ctx := context.Background()
	m := New()

	researcher := agent.New("researcher").
		Provides(core.CapResearch).
		Build()

	writer := agent.New("writer").
		Provides(core.CapWriting).
		Build()

	m.Register(researcher, writer)

	providers, err := m.FindProviders(ctx, core.CapResearch)
	if err != nil {
		t.Fatalf("failed to find: %v", err)
	}

	if len(providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(providers))
	}
}

func TestMesh_TraceID(t *testing.T) {
	ctx := context.Background()

	m := New()

	a := agent.New("test").
		Model(provider.NewMockWithResponse("test")).
		Build()

	m.Register(a)

	result, err := m.RunAgent(ctx, a.ID(), "test")
	if err != nil {
		t.Fatalf("failed to run: %v", err)
	}

	// Should have a trace ID
	if result.TraceID == "" {
		t.Error("expected trace ID to be set")
	}
}
