package middleware

import (
	"context"
	"testing"

	"github.com/storo/lettice/pkg/agent"
	"github.com/storo/lettice/pkg/protocol"
	"github.com/storo/lettice/pkg/provider"
)

func TestTracingMiddleware(t *testing.T) {
	ctx := context.Background()

	tracer := protocol.NewTracer()

	mockProvider := provider.NewMockWithResponse("Hello!")

	a := agent.New("test-agent").
		Model(mockProvider).
		Build()

	wrapped := WrapWithTracing(a, tracer)

	_, err := wrapped.Run(ctx, "Say hello")
	if err != nil {
		t.Fatalf("failed to run: %v", err)
	}

	spans := tracer.Spans()
	if len(spans) != 1 {
		t.Errorf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.Name != "agent.run" {
		t.Errorf("expected span name 'agent.run', got '%s'", span.Name)
	}

	if span.Attributes["agent.id"] != a.ID() {
		t.Error("expected agent.id attribute")
	}

	if span.Attributes["agent.name"] != "test-agent" {
		t.Error("expected agent.name attribute")
	}
}

func TestTracingMiddleware_Error(t *testing.T) {
	ctx := context.Background()

	tracer := protocol.NewTracer()

	// Agent without provider will fail
	a := agent.New("failing-agent").Build()

	wrapped := WrapWithTracing(a, tracer)

	_, err := wrapped.Run(ctx, "Test")
	if err == nil {
		t.Error("expected error")
	}

	spans := tracer.Spans()
	if len(spans) != 1 {
		t.Errorf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.Status != protocol.StatusError {
		t.Error("expected error status")
	}
}

func TestTracingMiddleware_PreservesTraceID(t *testing.T) {
	tracer := protocol.NewTracer()

	// Start with a custom trace ID
	ctx := protocol.WithTraceID(context.Background(), "custom-trace-123")

	mockProvider := provider.NewMockWithResponse("Result")

	a := agent.New("test-agent").
		Model(mockProvider).
		Build()

	wrapped := WrapWithTracing(a, tracer)

	_, err := wrapped.Run(ctx, "Test")
	if err != nil {
		t.Fatalf("failed to run: %v", err)
	}

	spans := tracer.Spans()
	if spans[0].TraceID != "custom-trace-123" {
		t.Errorf("expected trace ID 'custom-trace-123', got '%s'", spans[0].TraceID)
	}
}

func TestTracingMiddleware_NestedCalls(t *testing.T) {
	tracer := protocol.NewTracer()

	ctx := context.Background()

	mockProvider := provider.NewMockWithResponse("Result")

	a1 := agent.New("agent-1").Model(mockProvider).Build()
	a2 := agent.New("agent-2").Model(mockProvider).Build()

	wrapped1 := WrapWithTracing(a1, tracer)
	wrapped2 := WrapWithTracing(a2, tracer)

	// Simulate nested calls by running in sequence
	ctx, span1 := tracer.StartSpan(ctx, "outer")
	_, _ = wrapped1.Run(ctx, "First call")
	_, _ = wrapped2.Run(ctx, "Second call")
	span1.End()

	spans := tracer.Spans()
	// Should have: outer span + 2 agent spans
	if len(spans) != 3 {
		t.Errorf("expected 3 spans, got %d", len(spans))
	}
}
