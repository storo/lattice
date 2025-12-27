package core

import (
	"context"
	"testing"
)

func TestCallChain_EmptyContext(t *testing.T) {
	ctx := context.Background()
	chain := CallChain(ctx)

	if chain != nil {
		t.Errorf("expected nil chain for empty context, got %v", chain)
	}
}

func TestWithCallChain_AddsAgent(t *testing.T) {
	ctx := context.Background()
	ctx = WithCallChain(ctx, "agent-1")

	chain := CallChain(ctx)
	if len(chain) != 1 {
		t.Fatalf("expected chain length 1, got %d", len(chain))
	}
	if chain[0] != "agent-1" {
		t.Errorf("expected agent-1, got %s", chain[0])
	}
}

func TestWithCallChain_ChainsMultipleAgents(t *testing.T) {
	ctx := context.Background()
	ctx = WithCallChain(ctx, "agent-1")
	ctx = WithCallChain(ctx, "agent-2")
	ctx = WithCallChain(ctx, "agent-3")

	chain := CallChain(ctx)
	if len(chain) != 3 {
		t.Fatalf("expected chain length 3, got %d", len(chain))
	}

	expected := []string{"agent-1", "agent-2", "agent-3"}
	for i, id := range expected {
		if chain[i] != id {
			t.Errorf("expected %s at position %d, got %s", id, i, chain[i])
		}
	}
}

func TestInCallChain_DetectsCycle(t *testing.T) {
	ctx := context.Background()
	ctx = WithCallChain(ctx, "agent-1")
	ctx = WithCallChain(ctx, "agent-2")

	if !InCallChain(ctx, "agent-1") {
		t.Error("expected agent-1 to be in chain")
	}
	if !InCallChain(ctx, "agent-2") {
		t.Error("expected agent-2 to be in chain")
	}
	if InCallChain(ctx, "agent-3") {
		t.Error("expected agent-3 to NOT be in chain")
	}
}

func TestHopCount_StartsAtZero(t *testing.T) {
	ctx := context.Background()
	count := HopCount(ctx)

	if count != 0 {
		t.Errorf("expected hop count 0, got %d", count)
	}
}

func TestWithHopCount_Increments(t *testing.T) {
	ctx := context.Background()
	ctx = WithHopCount(ctx)
	ctx = WithHopCount(ctx)
	ctx = WithHopCount(ctx)

	count := HopCount(ctx)
	if count != 3 {
		t.Errorf("expected hop count 3, got %d", count)
	}
}

func TestTraceID_EmptyByDefault(t *testing.T) {
	ctx := context.Background()
	traceID := TraceID(ctx)

	if traceID != "" {
		t.Errorf("expected empty trace ID, got %s", traceID)
	}
}

func TestWithTraceID_SetsTraceID(t *testing.T) {
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-123")

	traceID := TraceID(ctx)
	if traceID != "trace-123" {
		t.Errorf("expected trace-123, got %s", traceID)
	}
}

func TestRemainingTimeout_EmptyByDefault(t *testing.T) {
	ctx := context.Background()
	timeout := RemainingTimeout(ctx)

	if timeout != 0 {
		t.Errorf("expected 0 timeout, got %v", timeout)
	}
}

func TestWithRemainingTimeout_SetsTimeout(t *testing.T) {
	ctx := context.Background()
	ctx = WithRemainingTimeout(ctx, 30000) // 30 seconds in ms

	timeout := RemainingTimeout(ctx)
	if timeout != 30000 {
		t.Errorf("expected 30000, got %d", timeout)
	}
}

func TestCallChain_DoesNotMutateOriginal(t *testing.T) {
	ctx1 := context.Background()
	ctx1 = WithCallChain(ctx1, "agent-1")

	ctx2 := WithCallChain(ctx1, "agent-2")

	chain1 := CallChain(ctx1)
	chain2 := CallChain(ctx2)

	if len(chain1) != 1 {
		t.Errorf("expected chain1 length 1, got %d", len(chain1))
	}
	if len(chain2) != 2 {
		t.Errorf("expected chain2 length 2, got %d", len(chain2))
	}
}
