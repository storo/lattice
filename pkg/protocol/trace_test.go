package protocol

import (
	"context"
	"testing"
	"time"
)

func TestSpan_Basic(t *testing.T) {
	tracer := NewTracer()

	ctx := context.Background()
	ctx, span := tracer.StartSpan(ctx, "test-operation")

	if span.TraceID == "" {
		t.Error("expected non-empty trace ID")
	}

	if span.SpanID == "" {
		t.Error("expected non-empty span ID")
	}

	if span.Name != "test-operation" {
		t.Errorf("expected name 'test-operation', got '%s'", span.Name)
	}

	span.End()

	if span.Duration == 0 {
		t.Error("expected non-zero duration")
	}
}

func TestSpan_ParentChild(t *testing.T) {
	tracer := NewTracer()

	ctx := context.Background()
	ctx, parent := tracer.StartSpan(ctx, "parent")

	_, child := tracer.StartSpan(ctx, "child")

	if child.ParentID != parent.SpanID {
		t.Errorf("expected child parent ID '%s', got '%s'", parent.SpanID, child.ParentID)
	}

	if child.TraceID != parent.TraceID {
		t.Error("child and parent should share trace ID")
	}

	child.End()
	parent.End()
}

func TestSpan_Attributes(t *testing.T) {
	tracer := NewTracer()

	ctx := context.Background()
	_, span := tracer.StartSpan(ctx, "test")

	span.SetAttribute("agent.id", "agent-1")
	span.SetAttribute("tokens.in", 100)
	span.SetAttribute("success", true)

	if span.Attributes["agent.id"] != "agent-1" {
		t.Error("expected agent.id attribute")
	}

	if span.Attributes["tokens.in"] != 100 {
		t.Error("expected tokens.in attribute")
	}

	span.End()
}

func TestSpan_Events(t *testing.T) {
	tracer := NewTracer()

	ctx := context.Background()
	_, span := tracer.StartSpan(ctx, "test")

	span.AddEvent("tool_call", map[string]any{
		"tool": "calculator",
	})

	span.AddEvent("tool_result", map[string]any{
		"result": "42",
	})

	if len(span.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(span.Events))
	}

	if span.Events[0].Name != "tool_call" {
		t.Errorf("expected first event 'tool_call', got '%s'", span.Events[0].Name)
	}

	span.End()
}

func TestSpan_Status(t *testing.T) {
	tracer := NewTracer()

	ctx := context.Background()
	_, span := tracer.StartSpan(ctx, "test")

	span.SetStatus(StatusOK, "")
	if span.Status != StatusOK {
		t.Error("expected OK status")
	}

	span.SetStatus(StatusError, "something went wrong")
	if span.Status != StatusError {
		t.Error("expected Error status")
	}
	if span.StatusMessage != "something went wrong" {
		t.Error("expected status message")
	}

	span.End()
}

func TestTracer_Export(t *testing.T) {
	tracer := NewTracer()

	ctx := context.Background()
	ctx, parent := tracer.StartSpan(ctx, "parent")
	_, child := tracer.StartSpan(ctx, "child")

	child.End()
	parent.End()

	spans := tracer.Spans()
	if len(spans) != 2 {
		t.Errorf("expected 2 spans, got %d", len(spans))
	}
}

func TestSpanFromContext(t *testing.T) {
	tracer := NewTracer()

	ctx := context.Background()
	ctx, span := tracer.StartSpan(ctx, "test")

	retrieved := SpanFromContext(ctx)
	if retrieved == nil {
		t.Fatal("expected to retrieve span from context")
	}

	if retrieved.SpanID != span.SpanID {
		t.Error("retrieved span should match original")
	}

	span.End()
}

func TestSpan_WithTraceID(t *testing.T) {
	tracer := NewTracer()

	// Start with a predefined trace ID
	ctx := context.Background()
	ctx = WithTraceID(ctx, "custom-trace-id")

	ctx, span := tracer.StartSpan(ctx, "test")

	if span.TraceID != "custom-trace-id" {
		t.Errorf("expected trace ID 'custom-trace-id', got '%s'", span.TraceID)
	}

	span.End()
}

func TestEvent_Timestamp(t *testing.T) {
	tracer := NewTracer()

	ctx := context.Background()
	_, span := tracer.StartSpan(ctx, "test")

	before := time.Now()
	span.AddEvent("test", nil)
	after := time.Now()

	event := span.Events[0]
	if event.Timestamp.Before(before) || event.Timestamp.After(after) {
		t.Error("event timestamp should be between before and after")
	}

	span.End()
}
