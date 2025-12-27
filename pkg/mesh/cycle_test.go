package mesh

import (
	"context"
	"testing"

	"github.com/storo/lattice/pkg/core"
)

func TestCycleDetector_Check_NoCycle(t *testing.T) {
	ctx := context.Background()
	cd := NewCycleDetector(10)

	// Empty chain - should pass
	if err := cd.Check(ctx, "agent-1"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Add agent-1 to chain
	ctx = core.WithCallChain(ctx, "agent-1")

	// Different agent - should pass
	if err := cd.Check(ctx, "agent-2"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCycleDetector_Check_DetectsCycle(t *testing.T) {
	ctx := context.Background()
	cd := NewCycleDetector(10)

	// Add agent-1 to chain
	ctx = core.WithCallChain(ctx, "agent-1")

	// Same agent - should detect cycle
	err := cd.Check(ctx, "agent-1")
	if err == nil {
		t.Error("expected cycle detection error")
	}
	if err != ErrCycleDetected {
		t.Errorf("expected ErrCycleDetected, got %v", err)
	}
}

func TestCycleDetector_Check_MaxHops(t *testing.T) {
	ctx := context.Background()
	cd := NewCycleDetector(3)

	// Add 3 agents to chain
	ctx = core.WithCallChain(ctx, "agent-1")
	ctx = core.WithHopCount(ctx)
	ctx = core.WithCallChain(ctx, "agent-2")
	ctx = core.WithHopCount(ctx)
	ctx = core.WithCallChain(ctx, "agent-3")
	ctx = core.WithHopCount(ctx)

	// 4th agent should exceed max hops
	err := cd.Check(ctx, "agent-4")
	if err == nil {
		t.Error("expected max hops error")
	}
	if err != ErrMaxHopsExceeded {
		t.Errorf("expected ErrMaxHopsExceeded, got %v", err)
	}
}

func TestCycleDetector_PrepareContext(t *testing.T) {
	ctx := context.Background()
	cd := NewCycleDetector(10)

	// Prepare context
	ctx = cd.PrepareContext(ctx, "agent-1")

	// Check that agent was added to chain
	chain := core.CallChain(ctx)
	if len(chain) != 1 {
		t.Errorf("expected 1 agent in chain, got %d", len(chain))
	}
	if chain[0] != "agent-1" {
		t.Errorf("expected agent-1 in chain, got %s", chain[0])
	}

	// Check hop count was incremented
	if core.HopCount(ctx) != 1 {
		t.Errorf("expected hop count 1, got %d", core.HopCount(ctx))
	}
}

func TestCycleDetector_DefaultMaxHops(t *testing.T) {
	// Zero should use default
	cd := NewCycleDetector(0)
	if cd.maxHops != DefaultMaxHops {
		t.Errorf("expected default max hops %d, got %d", DefaultMaxHops, cd.maxHops)
	}

	// Negative should use default
	cd = NewCycleDetector(-5)
	if cd.maxHops != DefaultMaxHops {
		t.Errorf("expected default max hops %d, got %d", DefaultMaxHops, cd.maxHops)
	}
}

func TestCycleDetector_ComplexChain(t *testing.T) {
	ctx := context.Background()
	cd := NewCycleDetector(10)

	// Simulate: A -> B -> C
	ctx = cd.PrepareContext(ctx, "A")
	ctx = cd.PrepareContext(ctx, "B")
	ctx = cd.PrepareContext(ctx, "C")

	// D is fine
	if err := cd.Check(ctx, "D"); err != nil {
		t.Errorf("D should be allowed: %v", err)
	}

	// B is a cycle
	if err := cd.Check(ctx, "B"); err != ErrCycleDetected {
		t.Errorf("B should be detected as cycle")
	}

	// A is a cycle
	if err := cd.Check(ctx, "A"); err != ErrCycleDetected {
		t.Errorf("A should be detected as cycle")
	}
}
