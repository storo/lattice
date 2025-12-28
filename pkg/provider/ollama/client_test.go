package ollama

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/storo/lattice/pkg/core"
	"github.com/storo/lattice/pkg/provider"
)

func TestClient_Name(t *testing.T) {
	client := NewClient()
	if client.Name() != "ollama" {
		t.Errorf("expected 'ollama', got '%s'", client.Name())
	}
}

func TestClient_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/chat" {
			t.Errorf("expected /api/chat, got %s", r.URL.Path)
		}

		// Parse request
		body, _ := io.ReadAll(r.Body)
		var req chatRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to parse request: %v", err)
		}

		if req.Model != "llama3.2" {
			t.Errorf("expected model llama3.2, got %s", req.Model)
		}
		if req.Stream {
			t.Error("expected non-streaming request")
		}

		// Return response
		resp := chatResponse{
			Model: "llama3.2",
			Message: message{
				Role:    "assistant",
				Content: "Hello! I'm a helpful assistant.",
			},
			Done:            true,
			PromptEvalCount: 10,
			EvalCount:       8,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	resp, err := client.Chat(context.Background(), &provider.ChatRequest{
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Hello"},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "Hello! I'm a helpful assistant." {
		t.Errorf("unexpected content: %s", resp.Content)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("expected 10 input tokens, got %d", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 8 {
		t.Errorf("expected 8 output tokens, got %d", resp.Usage.OutputTokens)
	}
}

func TestClient_Chat_WithSystem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req chatRequest
		json.Unmarshal(body, &req)

		// Check system message is first
		if len(req.Messages) < 2 {
			t.Errorf("expected at least 2 messages, got %d", len(req.Messages))
		}
		if req.Messages[0].Role != "system" {
			t.Errorf("expected system message first, got %s", req.Messages[0].Role)
		}
		if req.Messages[0].Content != "You are helpful." {
			t.Errorf("unexpected system content: %s", req.Messages[0].Content)
		}

		resp := chatResponse{
			Model:   "llama3.2",
			Message: message{Role: "assistant", Content: "Hi!"},
			Done:    true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	_, err := client.Chat(context.Background(), &provider.ChatRequest{
		System: "You are helpful.",
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Hello"},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Chat_WithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req chatRequest
		json.Unmarshal(body, &req)

		// Check tools are passed
		if len(req.Tools) != 1 {
			t.Errorf("expected 1 tool, got %d", len(req.Tools))
		}
		if req.Tools[0].Function.Name != "get_weather" {
			t.Errorf("unexpected tool name: %s", req.Tools[0].Function.Name)
		}

		// Return tool call
		resp := chatResponse{
			Model: "llama3.2",
			Message: message{
				Role: "assistant",
				ToolCalls: []toolCall{
					{
						Function: functionCall{
							Name:      "get_weather",
							Arguments: json.RawMessage(`{"location": "London"}`),
						},
					},
				},
			},
			Done:       true,
			DoneReason: "tool_use",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	resp, err := client.Chat(context.Background(), &provider.ChatRequest{
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "What's the weather in London?"},
		},
		Tools: []provider.ToolDefinition{
			{
				Name:        "get_weather",
				Description: "Get weather for a location",
				InputSchema: json.RawMessage(`{"type": "object", "properties": {"location": {"type": "string"}}}`),
			},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StopReason != provider.StopReasonToolUse {
		t.Errorf("expected StopReasonToolUse, got %s", resp.StopReason)
	}
	if len(resp.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "get_weather" {
		t.Errorf("unexpected tool name: %s", resp.ToolCalls[0].Name)
	}
}

func TestClient_ChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req chatRequest
		json.Unmarshal(body, &req)

		if !req.Stream {
			t.Error("expected streaming request")
		}

		// Write streaming response (NDJSON)
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected ResponseWriter to support Flush")
		}

		w.Header().Set("Content-Type", "application/x-ndjson")

		// Delta 1
		resp1 := streamResponse{
			Model:   "llama3.2",
			Message: message{Role: "assistant", Content: "Hello"},
		}
		json.NewEncoder(w).Encode(resp1)
		flusher.Flush()

		// Delta 2
		resp2 := streamResponse{
			Model:   "llama3.2",
			Message: message{Role: "assistant", Content: " world!"},
		}
		json.NewEncoder(w).Encode(resp2)
		flusher.Flush()

		// Done
		resp3 := streamResponse{
			Model:           "llama3.2",
			Message:         message{Role: "assistant"},
			Done:            true,
			PromptEvalCount: 5,
			EvalCount:       2,
		}
		json.NewEncoder(w).Encode(resp3)
		flusher.Flush()
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	events, err := client.ChatStream(context.Background(), &provider.ChatRequest{
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Hi"},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var content strings.Builder
	var stopEvent *provider.StreamEvent

	for event := range events {
		switch event.Type {
		case provider.EventTypeDelta:
			content.WriteString(event.Delta)
		case provider.EventTypeStop:
			stopEvent = &event
		case provider.EventTypeError:
			t.Fatalf("unexpected error: %s", event.Error)
		}
	}

	if content.String() != "Hello world!" {
		t.Errorf("expected 'Hello world!', got '%s'", content.String())
	}
	if stopEvent == nil {
		t.Fatal("expected stop event")
	}
	if stopEvent.Usage.InputTokens != 5 {
		t.Errorf("expected 5 input tokens, got %d", stopEvent.Usage.InputTokens)
	}
}

func TestClient_Chat_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: "model not found"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	_, err := client.Chat(context.Background(), &provider.ChatRequest{
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "Hello"},
		},
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "model not found") {
		t.Errorf("expected 'model not found' in error, got: %v", err)
	}
}

func TestClient_Options(t *testing.T) {
	client := NewClient(
		WithBaseURL("http://custom:1234"),
		WithModel("mistral"),
		WithMaxTokens(2048),
	)

	if client.baseURL != "http://custom:1234" {
		t.Errorf("expected custom URL, got %s", client.baseURL)
	}
	if client.model != "mistral" {
		t.Errorf("expected mistral model, got %s", client.model)
	}
	if client.maxTokens != 2048 {
		t.Errorf("expected 2048 max tokens, got %d", client.maxTokens)
	}
}

func TestClient_DefaultOptions(t *testing.T) {
	client := NewClient()

	if client.baseURL != defaultBaseURL {
		t.Errorf("expected %s, got %s", defaultBaseURL, client.baseURL)
	}
	if client.model != defaultModel {
		t.Errorf("expected %s, got %s", defaultModel, client.model)
	}
	if client.maxTokens != defaultMaxTokens {
		t.Errorf("expected %d, got %d", defaultMaxTokens, client.maxTokens)
	}
}
