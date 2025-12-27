package patterns

import (
	"context"
	"testing"

	"github.com/storo/lettice/pkg/agent"
	"github.com/storo/lettice/pkg/core"
	"github.com/storo/lettice/pkg/provider"
)

func TestSupervisor_DelegateToWorker(t *testing.T) {
	ctx := context.Background()

	// Create worker agents
	worker1 := agent.New("researcher").
		Model(provider.NewMockWithResponse("Research result")).
		Provides(core.CapResearch).
		Build()

	worker2 := agent.New("writer").
		Model(provider.NewMockWithResponse("Written content")).
		Provides(core.CapWriting).
		Build()

	// Create supervisor that can delegate
	supervisor := NewSupervisor(
		WithWorkers(worker1, worker2),
		WithStrategy(StrategyRoundRobin),
	)

	// Delegate a research task
	result, err := supervisor.Delegate(ctx, core.CapResearch, "Find information about AI")
	if err != nil {
		t.Fatalf("failed to delegate: %v", err)
	}

	if result.Output != "Research result" {
		t.Errorf("expected 'Research result', got '%s'", result.Output)
	}
}

func TestSupervisor_NoWorkerAvailable(t *testing.T) {
	ctx := context.Background()

	worker := agent.New("researcher").
		Model(provider.NewMockWithResponse("Research result")).
		Provides(core.CapResearch).
		Build()

	supervisor := NewSupervisor(WithWorkers(worker))

	// Try to delegate a capability no worker provides
	_, err := supervisor.Delegate(ctx, core.CapCoding, "Write code")
	if err != ErrNoWorkerAvailable {
		t.Errorf("expected ErrNoWorkerAvailable, got %v", err)
	}
}

func TestSupervisor_AllWorkers(t *testing.T) {
	worker1 := agent.New("researcher-1").
		Model(provider.NewMockWithResponse("Result 1")).
		Provides(core.CapResearch).
		Build()

	worker2 := agent.New("researcher-2").
		Model(provider.NewMockWithResponse("Result 2")).
		Provides(core.CapResearch).
		Build()

	supervisor := NewSupervisor(WithWorkers(worker1, worker2))

	// Get all workers for a capability
	workers := supervisor.WorkersFor(core.CapResearch)
	if len(workers) != 2 {
		t.Errorf("expected 2 workers, got %d", len(workers))
	}
}

func TestSupervisor_BroadcastToAll(t *testing.T) {
	ctx := context.Background()

	worker1 := agent.New("researcher-1").
		Model(provider.NewMockWithResponse("Result 1")).
		Provides(core.CapResearch).
		Build()

	worker2 := agent.New("researcher-2").
		Model(provider.NewMockWithResponse("Result 2")).
		Provides(core.CapResearch).
		Build()

	supervisor := NewSupervisor(WithWorkers(worker1, worker2))

	// Broadcast to all workers with a capability
	results, errs := supervisor.Broadcast(ctx, core.CapResearch, "Research task")

	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestSupervisor_RaceToFirst(t *testing.T) {
	ctx := context.Background()

	// Create workers with different response times (mocked by order)
	worker1 := agent.New("fast").
		Model(provider.NewMockWithResponse("Fast result")).
		Provides(core.CapResearch).
		Build()

	worker2 := agent.New("slow").
		Model(provider.NewMockWithResponse("Slow result")).
		Provides(core.CapResearch).
		Build()

	supervisor := NewSupervisor(
		WithWorkers(worker1, worker2),
		WithStrategy(StrategyRaceFirst),
	)

	result, err := supervisor.Delegate(ctx, core.CapResearch, "Race task")
	if err != nil {
		t.Fatalf("failed to race: %v", err)
	}

	// Should get one of the results
	if result.Output != "Fast result" && result.Output != "Slow result" {
		t.Errorf("unexpected result: %s", result.Output)
	}
}

func TestSupervisor_AddRemoveWorker(t *testing.T) {
	worker1 := agent.New("worker-1").
		Model(provider.NewMockWithResponse("Result")).
		Provides(core.CapResearch).
		Build()

	worker2 := agent.New("worker-2").
		Model(provider.NewMockWithResponse("Result")).
		Provides(core.CapResearch).
		Build()

	supervisor := NewSupervisor()

	// Add workers
	supervisor.AddWorker(worker1)
	supervisor.AddWorker(worker2)

	workers := supervisor.WorkersFor(core.CapResearch)
	if len(workers) != 2 {
		t.Errorf("expected 2 workers after add, got %d", len(workers))
	}

	// Remove a worker
	supervisor.RemoveWorker(worker1.ID())

	workers = supervisor.WorkersFor(core.CapResearch)
	if len(workers) != 1 {
		t.Errorf("expected 1 worker after remove, got %d", len(workers))
	}
}

func TestSupervisor_HealthCheck(t *testing.T) {
	worker := agent.New("healthy").
		Model(provider.NewMockWithResponse("OK")).
		Provides(core.CapResearch).
		Build()

	supervisor := NewSupervisor(WithWorkers(worker))

	// All workers should be healthy by default
	health := supervisor.Health()
	if !health[worker.ID()] {
		t.Error("worker should be healthy")
	}
}
