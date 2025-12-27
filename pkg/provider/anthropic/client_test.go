package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/storo/lattice/pkg/core"
	"github.com/storo/lattice/pkg/provider"
)

func TestClient_Name(t *testing.T) {
	client := NewClient("test-key")

	if client.Name() != "anthropic" {
		t.Errorf("expected name 'anthropic', got '%s'", client.Name())
	}
}

func TestClient_Chat(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Header.Get("x-api-key") != "test-key" {
			t.Error("missing or wrong API key")
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Error("missing anthropic-version header")
		}

		// Return mock response
		resp := messagesResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []contentBlock{
				{Type: "text", Text: "Hello, human!"},
			},
			StopReason: "end_turn",
			Usage: usage{
				InputTokens:  10,
				OutputTokens: 5,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	ctx := context.Background()
	req := &provider.ChatRequest{
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Hello!"},
		},
	}

	resp, err := client.Chat(ctx, req)
	if err != nil {
		t.Fatalf("failed to chat: %v", err)
	}

	if resp.Content != "Hello, human!" {
		t.Errorf("expected 'Hello, human!', got '%s'", resp.Content)
	}

	if resp.StopReason != provider.StopReasonEndTurn {
		t.Errorf("expected end_turn, got %s", resp.StopReason)
	}
}

func TestClient_ChatWithTools(t *testing.T) {
	// Mock server that returns a tool call
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := messagesResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []contentBlock{
				{
					Type: "tool_use",
					ID:   "tool_123",
					Name: "calculator",
					Input: map[string]any{
						"expression": "2 + 2",
					},
				},
			},
			StopReason: "tool_use",
			Usage: usage{
				InputTokens:  15,
				OutputTokens: 10,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	ctx := context.Background()
	req := &provider.ChatRequest{
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "What is 2 + 2?"},
		},
		Tools: []provider.ToolDefinition{
			{
				Name:        "calculator",
				Description: "Performs calculations",
				InputSchema: json.RawMessage(`{"type":"object"}`),
			},
		},
	}

	resp, err := client.Chat(ctx, req)
	if err != nil {
		t.Fatalf("failed to chat: %v", err)
	}

	if resp.StopReason != provider.StopReasonToolUse {
		t.Errorf("expected tool_use, got %s", resp.StopReason)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].Name != "calculator" {
		t.Errorf("expected tool 'calculator', got '%s'", resp.ToolCalls[0].Name)
	}
}

func TestClient_ChatError(t *testing.T) {
	// Mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "invalid_request_error",
				"message": "Invalid API key",
			},
		})
	}))
	defer server.Close()

	client := NewClient("bad-key", WithBaseURL(server.URL))

	ctx := context.Background()
	req := &provider.ChatRequest{
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Hello!"},
		},
	}

	_, err := client.Chat(ctx, req)
	if err == nil {
		t.Error("expected error for bad request")
	}
}

func TestClient_WithModel(t *testing.T) {
	client := NewClient("test-key", WithModel("claude-3-sonnet-20240229"))

	if client.model != "claude-3-sonnet-20240229" {
		t.Errorf("expected model to be set")
	}
}

func TestClient_WithMaxRetries(t *testing.T) {
	client := NewClient("test-key", WithMaxRetries(5))

	if client.maxRetries != 5 {
		t.Errorf("expected maxRetries 5, got %d", client.maxRetries)
	}
}

func TestConvertMessages(t *testing.T) {
	messages := []core.Message{
		{Role: core.RoleUser, Content: "Hello"},
		{Role: core.RoleAssistant, Content: "Hi there!"},
		{Role: core.RoleUser, Content: "How are you?"},
	}

	converted := convertMessages(messages)

	if len(converted) != 3 {
		t.Errorf("expected 3 messages, got %d", len(converted))
	}

	if converted[0].Role != "user" {
		t.Errorf("expected first role 'user', got '%s'", converted[0].Role)
	}

	if converted[1].Role != "assistant" {
		t.Errorf("expected second role 'assistant', got '%s'", converted[1].Role)
	}
}

func TestConvertTools(t *testing.T) {
	tools := []provider.ToolDefinition{
		{
			Name:        "search",
			Description: "Search the web",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`),
		},
	}

	converted := convertTools(tools)

	if len(converted) != 1 {
		t.Errorf("expected 1 tool, got %d", len(converted))
	}

	if converted[0].Name != "search" {
		t.Errorf("expected name 'search', got '%s'", converted[0].Name)
	}
}
