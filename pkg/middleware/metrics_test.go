package middleware

import (
	"context"
	"testing"

	"github.com/storo/lattice/pkg/agent"
	"github.com/storo/lattice/pkg/provider"
)

func TestMetricsMiddleware(t *testing.T) {
	ctx := context.Background()

	collector := NewMetricsCollector()

	mockProvider := provider.NewMockWithResponse("Hello!")

	a := agent.New("test-agent").
		Model(mockProvider).
		Build()

	wrapped := WrapWithMetrics(a, collector)

	// Run multiple times
	for i := 0; i < 5; i++ {
		_, err := wrapped.Run(ctx, "Say hello")
		if err != nil {
			t.Fatalf("failed to run: %v", err)
		}
	}

	metrics := collector.GetAgentMetrics(a.ID())

	if metrics.TotalCalls != 5 {
		t.Errorf("expected 5 total calls, got %d", metrics.TotalCalls)
	}

	if metrics.SuccessCount != 5 {
		t.Errorf("expected 5 successes, got %d", metrics.SuccessCount)
	}

	if metrics.ErrorCount != 0 {
		t.Errorf("expected 0 errors, got %d", metrics.ErrorCount)
	}
}

func TestMetricsMiddleware_Errors(t *testing.T) {
	ctx := context.Background()

	collector := NewMetricsCollector()

	// Agent without provider will fail
	a := agent.New("failing-agent").Build()

	wrapped := WrapWithMetrics(a, collector)

	for i := 0; i < 3; i++ {
		_, _ = wrapped.Run(ctx, "Test")
	}

	metrics := collector.GetAgentMetrics(a.ID())

	if metrics.ErrorCount != 3 {
		t.Errorf("expected 3 errors, got %d", metrics.ErrorCount)
	}

	if metrics.SuccessCount != 0 {
		t.Errorf("expected 0 successes, got %d", metrics.SuccessCount)
	}
}

func TestMetricsMiddleware_Duration(t *testing.T) {
	ctx := context.Background()

	collector := NewMetricsCollector()

	mockProvider := provider.NewMockWithResponse("Result")

	a := agent.New("test-agent").
		Model(mockProvider).
		Build()

	wrapped := WrapWithMetrics(a, collector)

	_, err := wrapped.Run(ctx, "Test")
	if err != nil {
		t.Fatalf("failed to run: %v", err)
	}

	metrics := collector.GetAgentMetrics(a.ID())

	if metrics.AvgDuration == 0 {
		t.Error("expected non-zero average duration")
	}

	if metrics.MaxDuration == 0 {
		t.Error("expected non-zero max duration")
	}
}

func TestMetricsMiddleware_Tokens(t *testing.T) {
	ctx := context.Background()

	collector := NewMetricsCollector()

	mockProvider := provider.NewMockWithResponse("Result")

	a := agent.New("test-agent").
		Model(mockProvider).
		Build()

	wrapped := WrapWithMetrics(a, collector)

	_, err := wrapped.Run(ctx, "Test")
	if err != nil {
		t.Fatalf("failed to run: %v", err)
	}

	metrics := collector.GetAgentMetrics(a.ID())

	// Mock provider doesn't return tokens, but structure should work
	if metrics.TotalCalls != 1 {
		t.Error("expected 1 call")
	}
}

func TestMetricsCollector_AllMetrics(t *testing.T) {
	ctx := context.Background()

	collector := NewMetricsCollector()

	a1 := agent.New("agent-1").Model(provider.NewMockWithResponse("R1")).Build()
	a2 := agent.New("agent-2").Model(provider.NewMockWithResponse("R2")).Build()

	wrapped1 := WrapWithMetrics(a1, collector)
	wrapped2 := WrapWithMetrics(a2, collector)

	_, _ = wrapped1.Run(ctx, "Test1")
	_, _ = wrapped2.Run(ctx, "Test2")

	all := collector.AllMetrics()

	if len(all) != 2 {
		t.Errorf("expected 2 agent metrics, got %d", len(all))
	}
}

func TestMetricsCollector_Reset(t *testing.T) {
	ctx := context.Background()

	collector := NewMetricsCollector()

	a := agent.New("test-agent").
		Model(provider.NewMockWithResponse("Result")).
		Build()

	wrapped := WrapWithMetrics(a, collector)

	_, _ = wrapped.Run(ctx, "Test")

	metrics := collector.GetAgentMetrics(a.ID())
	if metrics.TotalCalls != 1 {
		t.Error("expected 1 call before reset")
	}

	collector.Reset()

	metrics = collector.GetAgentMetrics(a.ID())
	if metrics.TotalCalls != 0 {
		t.Error("expected 0 calls after reset")
	}
}

func TestMetricsCollector_Summary(t *testing.T) {
	ctx := context.Background()

	collector := NewMetricsCollector()

	a := agent.New("test-agent").
		Model(provider.NewMockWithResponse("Result")).
		Build()

	wrapped := WrapWithMetrics(a, collector)

	for i := 0; i < 10; i++ {
		_, _ = wrapped.Run(ctx, "Test")
	}

	summary := collector.Summary()

	if summary.TotalCalls != 10 {
		t.Errorf("expected 10 total calls, got %d", summary.TotalCalls)
	}

	if summary.AgentCount != 1 {
		t.Errorf("expected 1 agent, got %d", summary.AgentCount)
	}
}
