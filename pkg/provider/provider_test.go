package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/storo/lattice/pkg/core"
)

func TestChatRequest_Basic(t *testing.T) {
	req := &ChatRequest{
		Model: "claude-3-opus",
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Hello"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	if req.Model != "claude-3-opus" {
		t.Errorf("expected model claude-3-opus, got %s", req.Model)
	}
	if len(req.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(req.Messages))
	}
	if req.MaxTokens != 1000 {
		t.Errorf("expected max tokens 1000, got %d", req.MaxTokens)
	}
}

func TestChatRequest_WithTools(t *testing.T) {
	tool := ToolDefinition{
		Name:        "search",
		Description: "Search the web",
		InputSchema: json.RawMessage(`{"type": "object"}`),
	}

	req := &ChatRequest{
		Model:    "claude-3-opus",
		Messages: []core.Message{},
		Tools:    []ToolDefinition{tool},
	}

	if len(req.Tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(req.Tools))
	}
	if req.Tools[0].Name != "search" {
		t.Errorf("expected tool name 'search', got '%s'", req.Tools[0].Name)
	}
}

func TestChatResponse_Basic(t *testing.T) {
	resp := &ChatResponse{
		Content:   "Hello, human!",
		StopReason: StopReasonEndTurn,
		Usage: Usage{
			InputTokens:  10,
			OutputTokens: 5,
		},
	}

	if resp.Content != "Hello, human!" {
		t.Errorf("expected content 'Hello, human!', got '%s'", resp.Content)
	}
	if resp.StopReason != StopReasonEndTurn {
		t.Errorf("expected stop reason end_turn, got %s", resp.StopReason)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("expected input tokens 10, got %d", resp.Usage.InputTokens)
	}
}

func TestChatResponse_WithToolCalls(t *testing.T) {
	resp := &ChatResponse{
		Content:    "",
		StopReason: StopReasonToolUse,
		ToolCalls: []core.ToolCall{
			{
				ID:     "call-123",
				Name:   "search",
				Params: json.RawMessage(`{"query": "test"}`),
			},
		},
	}

	if resp.StopReason != StopReasonToolUse {
		t.Errorf("expected stop reason tool_use, got %s", resp.StopReason)
	}
	if len(resp.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
}

func TestStreamEvent_Types(t *testing.T) {
	events := []StreamEvent{
		{Type: EventTypeStart},
		{Type: EventTypeDelta, Delta: "Hello"},
		{Type: EventTypeToolCall, ToolCall: &core.ToolCall{ID: "call-1"}},
		{Type: EventTypeStop, StopReason: StopReasonEndTurn},
		{Type: EventTypeError, Error: "something went wrong"},
	}

	if events[0].Type != EventTypeStart {
		t.Error("expected start event")
	}
	if events[1].Delta != "Hello" {
		t.Error("expected delta content")
	}
	if events[2].ToolCall == nil {
		t.Error("expected tool call")
	}
	if events[3].StopReason != StopReasonEndTurn {
		t.Error("expected stop reason")
	}
	if events[4].Error != "something went wrong" {
		t.Error("expected error message")
	}
}

func TestMockProvider_ImplementsInterface(t *testing.T) {
	var _ Provider = (*MockProvider)(nil)
}

func TestMockProvider_Chat(t *testing.T) {
	ctx := context.Background()
	provider := &MockProvider{
		ChatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			return &ChatResponse{
				Content:    "test response",
				StopReason: StopReasonEndTurn,
			}, nil
		},
	}

	resp, err := provider.Chat(ctx, &ChatRequest{
		Model:    "test",
		Messages: []core.Message{{Role: core.RoleUser, Content: "test"}},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "test response" {
		t.Errorf("expected 'test response', got '%s'", resp.Content)
	}
}
