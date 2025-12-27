package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestToolDefinition(t *testing.T) {
	def := ToolDefinition{
		Name:        "calculator",
		Description: "Performs math calculations",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"expression":{"type":"string"}}}`),
	}

	if def.Name != "calculator" {
		t.Errorf("expected name 'calculator', got '%s'", def.Name)
	}
}

func TestToolCall(t *testing.T) {
	call := ToolCall{
		ID:   "call-1",
		Name: "calculator",
		Arguments: map[string]any{
			"expression": "2 + 2",
		},
	}

	if call.ID != "call-1" {
		t.Errorf("expected ID 'call-1', got '%s'", call.ID)
	}

	if call.Arguments["expression"] != "2 + 2" {
		t.Error("expected expression argument")
	}
}

func TestToolResult(t *testing.T) {
	result := ToolResult{
		CallID:  "call-1",
		Content: "4",
		IsError: false,
	}

	if result.CallID != "call-1" {
		t.Error("call ID mismatch")
	}

	if result.IsError {
		t.Error("should not be error")
	}
}

func TestToolExecutor(t *testing.T) {
	executor := NewToolExecutor()

	// Register a tool
	executor.Register(&mockMCPTool{
		def: ToolDefinition{
			Name:        "echo",
			Description: "Echoes input",
		},
		exec: func(ctx context.Context, args map[string]any) (string, error) {
			return args["message"].(string), nil
		},
	})

	// List tools
	tools := executor.List()
	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}

	// Execute tool
	ctx := context.Background()
	call := ToolCall{
		ID:   "call-1",
		Name: "echo",
		Arguments: map[string]any{
			"message": "Hello!",
		},
	}

	result, err := executor.Execute(ctx, call)
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}

	if result.Content != "Hello!" {
		t.Errorf("expected 'Hello!', got '%s'", result.Content)
	}
}

func TestToolExecutor_NotFound(t *testing.T) {
	executor := NewToolExecutor()

	ctx := context.Background()
	call := ToolCall{
		ID:   "call-1",
		Name: "nonexistent",
	}

	_, err := executor.Execute(ctx, call)
	if err != ErrToolNotFound {
		t.Errorf("expected ErrToolNotFound, got %v", err)
	}
}

func TestToolExecutor_Unregister(t *testing.T) {
	executor := NewToolExecutor()

	executor.Register(&mockMCPTool{
		def: ToolDefinition{Name: "tool1"},
	})
	executor.Register(&mockMCPTool{
		def: ToolDefinition{Name: "tool2"},
	})

	if len(executor.List()) != 2 {
		t.Error("expected 2 tools")
	}

	executor.Unregister("tool1")

	if len(executor.List()) != 1 {
		t.Error("expected 1 tool after unregister")
	}
}

func TestValidateArguments(t *testing.T) {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "number"}
		},
		"required": ["name"]
	}`)

	// Valid arguments
	args := map[string]any{
		"name": "Alice",
		"age":  30,
	}

	if err := ValidateArguments(schema, args); err != nil {
		t.Errorf("expected valid arguments: %v", err)
	}
}

// mockMCPTool implements Tool for testing
type mockMCPTool struct {
	def  ToolDefinition
	exec func(ctx context.Context, args map[string]any) (string, error)
}

func (t *mockMCPTool) Definition() ToolDefinition {
	return t.def
}

func (t *mockMCPTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.exec != nil {
		return t.exec(ctx, args)
	}
	return "", nil
}
