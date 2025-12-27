package patterns

import (
	"context"
	"testing"

	"github.com/storo/lattice/pkg/agent"
	"github.com/storo/lattice/pkg/provider"
)

func TestParallel_RunAll(t *testing.T) {
	ctx := context.Background()

	a1 := agent.New("agent-1").
		Model(provider.NewMockWithResponse("Result 1")).
		Build()

	a2 := agent.New("agent-2").
		Model(provider.NewMockWithResponse("Result 2")).
		Build()

	a3 := agent.New("agent-3").
		Model(provider.NewMockWithResponse("Result 3")).
		Build()

	parallel := NewParallel(a1, a2, a3)

	results, errs := parallel.Run(ctx, "Shared input")

	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestParallel_EmptyAgents(t *testing.T) {
	ctx := context.Background()

	parallel := NewParallel()

	results, errs := parallel.Run(ctx, "Test")

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}

	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}
}

func TestParallel_PartialFailure(t *testing.T) {
	ctx := context.Background()

	success := agent.New("success").
		Model(provider.NewMockWithResponse("OK")).
		Build()

	failing := agent.New("failing").Build() // No provider

	parallel := NewParallel(success, failing)

	results, errs := parallel.Run(ctx, "Test")

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
}

func TestParallel_RaceFirst(t *testing.T) {
	ctx := context.Background()

	a1 := agent.New("agent-1").
		Model(provider.NewMockWithResponse("First")).
		Build()

	a2 := agent.New("agent-2").
		Model(provider.NewMockWithResponse("Second")).
		Build()

	parallel := NewParallel(a1, a2)

	result, err := parallel.Race(ctx, "Race input")
	if err != nil {
		t.Fatalf("failed to race: %v", err)
	}

	if result.Output != "First" && result.Output != "Second" {
		t.Errorf("unexpected result: %s", result.Output)
	}
}

func TestParallel_RaceAllFail(t *testing.T) {
	ctx := context.Background()

	failing1 := agent.New("failing-1").Build()
	failing2 := agent.New("failing-2").Build()

	parallel := NewParallel(failing1, failing2)

	_, err := parallel.Race(ctx, "Test")
	if err == nil {
		t.Error("expected error when all agents fail")
	}
}

func TestParallel_Aggregate(t *testing.T) {
	ctx := context.Background()

	a1 := agent.New("agent-1").
		Model(provider.NewMockWithResponse("Apple")).
		Build()

	a2 := agent.New("agent-2").
		Model(provider.NewMockWithResponse("Banana")).
		Build()

	parallel := NewParallel(a1, a2)

	result := parallel.Aggregate(ctx, "List fruits", func(results []*AgentResult) string {
		var combined string
		for i, r := range results {
			if i > 0 {
				combined += ", "
			}
			combined += r.Output
		}
		return combined
	})

	// Order is not guaranteed, but both should be present
	if result != "Apple, Banana" && result != "Banana, Apple" {
		t.Errorf("unexpected aggregated result: %s", result)
	}
}
