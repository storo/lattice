package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/storo/lattice/pkg/core"
	"github.com/storo/lattice/pkg/provider"
)

func TestAgent_Run_Basic(t *testing.T) {
	ctx := context.Background()

	mockProvider := &provider.MockProvider{
		ChatFunc: func(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
			return &provider.ChatResponse{
				Content:    "Hello, I'm a test agent!",
				StopReason: provider.StopReasonEndTurn,
				Usage: provider.Usage{
					InputTokens:  10,
					OutputTokens: 8,
				},
			}, nil
		},
	}

	agent := New("test-agent").
		Model(mockProvider).
		System("You are a helpful assistant").
		Build()

	result, err := agent.Run(ctx, "Hello!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Output != "Hello, I'm a test agent!" {
		t.Errorf("expected output 'Hello, I'm a test agent!', got '%s'", result.Output)
	}
	if result.TokensIn != 10 {
		t.Errorf("expected tokens in 10, got %d", result.TokensIn)
	}
	if result.TokensOut != 8 {
		t.Errorf("expected tokens out 8, got %d", result.TokensOut)
	}
}

func TestAgent_Run_WithToolCall(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	mockProvider := &provider.MockProvider{
		ChatFunc: func(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
			callCount++
			if callCount == 1 {
				// First call: request tool use
				return &provider.ChatResponse{
					StopReason: provider.StopReasonToolUse,
					ToolCalls: []core.ToolCall{
						{
							ID:     "call-123",
							Name:   "test_tool",
							Params: json.RawMessage(`{"input": "test"}`),
						},
					},
				}, nil
			}
			// Second call: return final response with tool result
			return &provider.ChatResponse{
				Content:    "Tool returned: tool result",
				StopReason: provider.StopReasonEndTurn,
			}, nil
		},
	}

	// Create a test tool
	testTool := &testToolImpl{
		name:        "test_tool",
		description: "A test tool",
		executeFunc: func(ctx context.Context, params json.RawMessage) (string, error) {
			return "tool result", nil
		},
	}

	agent := New("test-agent").
		Model(mockProvider).
		Tools(testTool).
		Build()

	result, err := agent.Run(ctx, "Use the tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Output != "Tool returned: tool result" {
		t.Errorf("expected tool result in output, got '%s'", result.Output)
	}
	if callCount != 2 {
		t.Errorf("expected 2 provider calls, got %d", callCount)
	}
}

func TestAgent_Run_NoProvider(t *testing.T) {
	ctx := context.Background()

	agent := New("test-agent").Build()

	_, err := agent.Run(ctx, "Hello")
	if err == nil {
		t.Error("expected error when no provider is set")
	}
}

func TestAgent_Stop(t *testing.T) {
	agent := New("test-agent").Build()

	// Should not error even if nothing is running
	if err := agent.Stop(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAgent_TraceID(t *testing.T) {
	ctx := context.Background()
	ctx = core.WithTraceID(ctx, "trace-123")

	mockProvider := provider.NewMockWithResponse("test")

	agent := New("test-agent").
		Model(mockProvider).
		Build()

	result, err := agent.Run(ctx, "Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TraceID != "trace-123" {
		t.Errorf("expected trace ID 'trace-123', got '%s'", result.TraceID)
	}
}

func TestAgent_CallChain(t *testing.T) {
	ctx := context.Background()
	ctx = core.WithCallChain(ctx, "parent-agent")

	mockProvider := provider.NewMockWithResponse("test")

	agent := New("test-agent").
		Model(mockProvider).
		Build()

	result, err := agent.Run(ctx, "Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include both parent and this agent
	if len(result.CallChain) != 2 {
		t.Errorf("expected 2 agents in call chain, got %d", len(result.CallChain))
	}
}

// testToolImpl is a test implementation of core.Tool
type testToolImpl struct {
	name        string
	description string
	schema      json.RawMessage
	executeFunc func(ctx context.Context, params json.RawMessage) (string, error)
}

func (t *testToolImpl) Name() string {
	return t.name
}

func (t *testToolImpl) Description() string {
	return t.description
}

func (t *testToolImpl) Schema() json.RawMessage {
	if t.schema != nil {
		return t.schema
	}
	return json.RawMessage(`{"type": "object"}`)
}

func (t *testToolImpl) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	if t.executeFunc != nil {
		return t.executeFunc(ctx, params)
	}
	return "executed", nil
}
