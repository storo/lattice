package patterns

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/storo/lettice/pkg/agent"
	"github.com/storo/lettice/pkg/core"
	"github.com/storo/lettice/pkg/provider"
)

func TestReActAgent_BasicReasoning(t *testing.T) {
	ctx := context.Background()

	// Mock provider that simulates ReAct reasoning
	callCount := 0
	mockProvider := &provider.MockProvider{
		ChatFunc: func(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
			callCount++
			// Return a final answer
			return &provider.ChatResponse{
				Content:    "Based on my reasoning, the answer is 42.",
				StopReason: provider.StopReasonEndTurn,
			}, nil
		},
	}

	a := agent.New("react-agent").
		Model(mockProvider).
		Build()

	react := NewReActAgent(a)

	result, err := react.Run(ctx, "What is the answer to life?")
	if err != nil {
		t.Fatalf("failed to run: %v", err)
	}

	if result.Output == "" {
		t.Error("expected non-empty output")
	}
}

func TestReActAgent_WithToolUse(t *testing.T) {
	ctx := context.Background()

	// Track tool calls
	calculatorCalled := false
	calculator := &mockTool{
		name:        "calculator",
		description: "Performs calculations",
		executeFunc: func(ctx context.Context, params json.RawMessage) (string, error) {
			calculatorCalled = true
			return "42", nil
		},
	}

	callCount := 0
	mockProvider := &provider.MockProvider{
		ChatFunc: func(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
			callCount++
			if callCount == 1 {
				// First call: use tool
				return &provider.ChatResponse{
					Content:    "Thought: I need to calculate this.\nAction: calculator",
					StopReason: provider.StopReasonToolUse,
					ToolCalls: []core.ToolCall{
						{
							ID:     "call-1",
							Name:   "calculator",
							Params: []byte(`{"expression": "6 * 7"}`),
						},
					},
				}, nil
			}
			// Second call: return final answer
			return &provider.ChatResponse{
				Content:    "Thought: The calculation returned 42.\nAnswer: The result is 42.",
				StopReason: provider.StopReasonEndTurn,
			}, nil
		},
	}

	a := agent.New("react-agent").
		Model(mockProvider).
		Tools(calculator).
		Build()

	react := NewReActAgent(a)

	result, err := react.Run(ctx, "What is 6 * 7?")
	if err != nil {
		t.Fatalf("failed to run: %v", err)
	}

	if !calculatorCalled {
		t.Error("expected calculator tool to be called")
	}

	if result.Output == "" {
		t.Error("expected non-empty output")
	}
}

func TestReActAgent_MaxIterations(t *testing.T) {
	ctx := context.Background()

	// For the iteration limit test, we need to simulate multi-step reasoning
	// where each agent.Run() completes but returns an Action that signals
	// more work is needed.
	roundCount := 0
	mockProvider := &provider.MockProvider{
		ChatFunc: func(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
			roundCount++
			// Return a response with an Action (meaning more iterations needed)
			return &provider.ChatResponse{
				Content:    "Thought: I need more info.\nAction: search",
				StopReason: provider.StopReasonEndTurn,
			}, nil
		},
	}

	a := agent.New("react-agent").
		Model(mockProvider).
		Build()

	react := NewReActAgent(a, WithMaxIterations(3))

	_, err := react.Run(ctx, "Search forever")
	if err != ErrMaxIterationsReached {
		t.Errorf("expected ErrMaxIterationsReached, got %v", err)
	}

	if roundCount != 3 {
		t.Errorf("expected exactly 3 iterations, got %d", roundCount)
	}
}

func TestReActAgent_Streaming(t *testing.T) {
	ctx := context.Background()

	mockProvider := &provider.MockProvider{
		ChatStreamFunc: func(ctx context.Context, req *provider.ChatRequest) (<-chan provider.StreamEvent, error) {
			ch := make(chan provider.StreamEvent, 3)
			go func() {
				defer close(ch)
				ch <- provider.StreamEvent{Type: provider.EventTypeStart}
				ch <- provider.StreamEvent{Type: provider.EventTypeDelta, Delta: "Thought: Analyzing..."}
				ch <- provider.StreamEvent{Type: provider.EventTypeStop}
			}()
			return ch, nil
		},
		ChatFunc: func(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
			return &provider.ChatResponse{
				Content:    "Thought: Analyzing...\nAnswer: Done!",
				StopReason: provider.StopReasonEndTurn,
			}, nil
		},
	}

	a := agent.New("react-agent").
		Model(mockProvider).
		Build()

	react := NewReActAgent(a)

	chunks, err := react.RunStream(ctx, "Test streaming")
	if err != nil {
		t.Fatalf("failed to run stream: %v", err)
	}

	count := 0
	for range chunks {
		count++
	}

	if count == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestReActAgent_ParseThoughtAction(t *testing.T) {
	tests := []struct {
		input   string
		thought string
		action  string
	}{
		{
			input:   "Thought: I need to search.\nAction: search",
			thought: "I need to search.",
			action:  "search",
		},
		{
			input:   "Thought: Let me think about this carefully.\nAnswer: The result is 42.",
			thought: "Let me think about this carefully.",
			action:  "",
		},
		{
			input:   "Just a plain response without format",
			thought: "",
			action:  "",
		},
	}

	for _, tt := range tests {
		thought, action := ParseThoughtAction(tt.input)
		if thought != tt.thought {
			t.Errorf("ParseThoughtAction(%q) thought = %q, want %q", tt.input, thought, tt.thought)
		}
		if action != tt.action {
			t.Errorf("ParseThoughtAction(%q) action = %q, want %q", tt.input, action, tt.action)
		}
	}
}

// mockTool implements core.Tool for testing
type mockTool struct {
	name        string
	description string
	executeFunc func(ctx context.Context, params json.RawMessage) (string, error)
}

func (t *mockTool) Name() string             { return t.name }
func (t *mockTool) Description() string      { return t.description }
func (t *mockTool) Schema() json.RawMessage  { return []byte(`{"type":"object"}`) }
func (t *mockTool) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	if t.executeFunc != nil {
		return t.executeFunc(ctx, params)
	}
	return "", nil
}
