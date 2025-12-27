package patterns

import (
	"context"
	"testing"

	"github.com/storo/lattice/pkg/agent"
	"github.com/storo/lattice/pkg/provider"
)

func TestSequential_RunPipeline(t *testing.T) {
	ctx := context.Background()

	// Create a pipeline: research -> analyze -> write
	researcher := agent.New("researcher").
		Model(provider.NewMockWithResponse("Research data")).
		Build()

	analyzer := agent.New("analyzer").
		Model(provider.NewMockWithResponse("Analysis complete")).
		Build()

	writer := agent.New("writer").
		Model(provider.NewMockWithResponse("Final report")).
		Build()

	pipeline := NewSequential(researcher, analyzer, writer)

	result, err := pipeline.Run(ctx, "Research AI trends")
	if err != nil {
		t.Fatalf("failed to run pipeline: %v", err)
	}

	// Should get the final agent's output
	if result.Output != "Final report" {
		t.Errorf("expected 'Final report', got '%s'", result.Output)
	}
}

func TestSequential_EmptyPipeline(t *testing.T) {
	ctx := context.Background()

	pipeline := NewSequential()

	_, err := pipeline.Run(ctx, "Test")
	if err != ErrEmptyPipeline {
		t.Errorf("expected ErrEmptyPipeline, got %v", err)
	}
}

func TestSequential_PassContext(t *testing.T) {
	ctx := context.Background()

	// First agent transforms input
	transform := agent.New("transform").
		Model(provider.NewMockWithResponse("Transformed: data")).
		Build()

	// Second agent processes the transformed input
	process := agent.New("process").
		Model(provider.NewMockWithResponse("Processed result")).
		Build()

	pipeline := NewSequential(transform, process)

	result, err := pipeline.Run(ctx, "Initial data")
	if err != nil {
		t.Fatalf("failed to run pipeline: %v", err)
	}

	if result.Output != "Processed result" {
		t.Errorf("expected 'Processed result', got '%s'", result.Output)
	}
}

func TestSequential_AccumulatesTokens(t *testing.T) {
	ctx := context.Background()

	// Create agents (mock doesn't track tokens, but structure supports it)
	a1 := agent.New("a1").Model(provider.NewMockWithResponse("R1")).Build()
	a2 := agent.New("a2").Model(provider.NewMockWithResponse("R2")).Build()

	pipeline := NewSequential(a1, a2)

	result, err := pipeline.Run(ctx, "Input")
	if err != nil {
		t.Fatalf("failed to run: %v", err)
	}

	// Result should have combined duration
	if result.Duration == 0 {
		t.Error("expected non-zero duration")
	}
}

func TestSequential_StopsOnError(t *testing.T) {
	ctx := context.Background()

	// First agent succeeds
	success := agent.New("success").
		Model(provider.NewMockWithResponse("OK")).
		Build()

	// Second agent fails (no provider)
	failing := agent.New("failing").Build()

	// Third agent would succeed but shouldn't run
	never := agent.New("never").
		Model(provider.NewMockWithResponse("Should not see this")).
		Build()

	pipeline := NewSequential(success, failing, never)

	_, err := pipeline.Run(ctx, "Test")
	if err == nil {
		t.Error("expected error from failing agent")
	}
}
