package core

import (
	"encoding/json"
	"testing"
	"time"
)

func TestResult_Basic(t *testing.T) {
	result := &Result{
		Output:    "Hello, World!",
		TokensIn:  10,
		TokensOut: 5,
		Duration:  100 * time.Millisecond,
		TraceID:   "trace-123",
		CallChain: []string{"agent-1", "agent-2"},
	}

	if result.Output != "Hello, World!" {
		t.Errorf("expected output 'Hello, World!', got '%s'", result.Output)
	}
	if result.TokensIn != 10 {
		t.Errorf("expected tokens in 10, got %d", result.TokensIn)
	}
	if result.TokensOut != 5 {
		t.Errorf("expected tokens out 5, got %d", result.TokensOut)
	}
	if result.Duration != 100*time.Millisecond {
		t.Errorf("expected duration 100ms, got %v", result.Duration)
	}
	if result.TraceID != "trace-123" {
		t.Errorf("expected trace ID 'trace-123', got '%s'", result.TraceID)
	}
	if len(result.CallChain) != 2 {
		t.Errorf("expected call chain length 2, got %d", len(result.CallChain))
	}
}

func TestResult_WithMetadata(t *testing.T) {
	result := &Result{
		Output: "test",
		Metadata: map[string]any{
			"model":       "claude-3",
			"temperature": 0.7,
		},
	}

	if result.Metadata["model"] != "claude-3" {
		t.Error("expected model metadata")
	}
	if result.Metadata["temperature"] != 0.7 {
		t.Error("expected temperature metadata")
	}
}

func TestStreamChunk_Basic(t *testing.T) {
	chunk := StreamChunk{
		Content: "Hello",
		Done:    false,
		Error:   nil,
	}

	if chunk.Content != "Hello" {
		t.Errorf("expected content 'Hello', got '%s'", chunk.Content)
	}
	if chunk.Done {
		t.Error("expected done to be false")
	}
	if chunk.Error != nil {
		t.Errorf("expected no error, got %v", chunk.Error)
	}
}

func TestAgentCard_JSON(t *testing.T) {
	card := &AgentCard{
		Name:        "researcher",
		Description: "A research agent",
		URL:         "http://localhost:8080",
		Version:     "1.0.0",
		Capabilities: CardCapabilities{
			Provides:  []Capability{CapResearch},
			Needs:     []Capability{},
			Streaming: true,
		},
		Skills: []Skill{
			{ID: "web-search", Name: "Web Search", Description: "Search the web"},
		},
		Tools:     []string{"search", "summarize"},
		Model:     "claude-3-opus",
		Protocols: []string{"a2a", "mcp"},
	}

	data, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("failed to marshal card: %v", err)
	}

	var parsed AgentCard
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal card: %v", err)
	}

	if parsed.Name != "researcher" {
		t.Errorf("expected name 'researcher', got '%s'", parsed.Name)
	}
	if parsed.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", parsed.Version)
	}
	if len(parsed.Capabilities.Provides) != 1 {
		t.Error("expected 1 provided capability")
	}
	if len(parsed.Skills) != 1 {
		t.Error("expected 1 skill")
	}
}

func TestAgentCard_WithInputSchema(t *testing.T) {
	schema := NewObjectSchema().
		Property("query", "string", "The search query").
		Required("query").
		Build()

	card := &AgentCard{
		Name:        "searcher",
		InputSchema: schema,
	}

	data, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("failed to marshal card: %v", err)
	}

	var parsed AgentCard
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal card: %v", err)
	}

	if parsed.InputSchema == nil {
		t.Error("expected input schema to be set")
	}
}

func TestSkill_Basic(t *testing.T) {
	skill := Skill{
		ID:          "web-search",
		Name:        "Web Search",
		Description: "Search the web for information",
	}

	if skill.ID != "web-search" {
		t.Errorf("expected ID 'web-search', got '%s'", skill.ID)
	}
	if skill.Name != "Web Search" {
		t.Errorf("expected name 'Web Search', got '%s'", skill.Name)
	}
}

func TestMessage_Roles(t *testing.T) {
	tests := []struct {
		role     Role
		expected string
	}{
		{RoleUser, "user"},
		{RoleAssistant, "assistant"},
		{RoleSystem, "system"},
		{RoleTool, "tool"},
	}

	for _, tt := range tests {
		if string(tt.role) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, string(tt.role))
		}
	}
}

func TestMessage_Basic(t *testing.T) {
	msg := Message{
		Role:    RoleUser,
		Content: "Hello, agent!",
	}

	if msg.Role != RoleUser {
		t.Errorf("expected role user, got %s", msg.Role)
	}
	if msg.Content != "Hello, agent!" {
		t.Errorf("expected content 'Hello, agent!', got '%s'", msg.Content)
	}
}

func TestToolCall_Basic(t *testing.T) {
	tc := ToolCall{
		ID:     "call-123",
		Name:   "search",
		Params: json.RawMessage(`{"query": "test"}`),
	}

	if tc.ID != "call-123" {
		t.Errorf("expected ID 'call-123', got '%s'", tc.ID)
	}
	if tc.Name != "search" {
		t.Errorf("expected name 'search', got '%s'", tc.Name)
	}
}

func TestToolResult_Basic(t *testing.T) {
	tr := ToolResult{
		CallID:  "call-123",
		Content: "Search results here",
		IsError: false,
	}

	if tr.CallID != "call-123" {
		t.Errorf("expected call ID 'call-123', got '%s'", tr.CallID)
	}
	if tr.IsError {
		t.Error("expected is_error to be false")
	}
}
