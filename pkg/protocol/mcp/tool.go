package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
)

// MCP errors
var (
	ErrToolNotFound = errors.New("tool not found")
)

// ToolDefinition describes a tool's interface.
type ToolDefinition struct {
	// Name is the unique identifier for the tool.
	Name string `json:"name"`

	// Description explains what the tool does.
	Description string `json:"description"`

	// InputSchema is the JSON Schema for the tool's parameters.
	InputSchema json.RawMessage `json:"input_schema"`
}

// ToolCall represents a request to execute a tool.
type ToolCall struct {
	// ID is the unique identifier for this call.
	ID string `json:"id"`

	// Name is the tool to execute.
	Name string `json:"name"`

	// Arguments are the parameters for the tool.
	Arguments map[string]any `json:"arguments"`
}

// ToolResult is the result of a tool execution.
type ToolResult struct {
	// CallID is the ID of the original call.
	CallID string `json:"call_id"`

	// Content is the result content.
	Content string `json:"content"`

	// IsError indicates if this is an error result.
	IsError bool `json:"is_error"`
}

// Tool is the interface for MCP tools.
type Tool interface {
	// Definition returns the tool's definition.
	Definition() ToolDefinition

	// Execute runs the tool with the given arguments.
	Execute(ctx context.Context, args map[string]any) (string, error)
}

// ToolExecutor manages and executes tools.
type ToolExecutor struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewToolExecutor creates a new tool executor.
func NewToolExecutor() *ToolExecutor {
	return &ToolExecutor{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the executor.
func (e *ToolExecutor) Register(tool Tool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tools[tool.Definition().Name] = tool
}

// Unregister removes a tool from the executor.
func (e *ToolExecutor) Unregister(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.tools, name)
}

// List returns all registered tool definitions.
func (e *ToolExecutor) List() []ToolDefinition {
	e.mu.RLock()
	defer e.mu.RUnlock()

	defs := make([]ToolDefinition, 0, len(e.tools))
	for _, tool := range e.tools {
		defs = append(defs, tool.Definition())
	}
	return defs
}

// Get returns a tool by name.
func (e *ToolExecutor) Get(name string) (Tool, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	tool, ok := e.tools[name]
	return tool, ok
}

// Execute runs a tool call and returns the result.
func (e *ToolExecutor) Execute(ctx context.Context, call ToolCall) (*ToolResult, error) {
	e.mu.RLock()
	tool, ok := e.tools[call.Name]
	e.mu.RUnlock()

	if !ok {
		return nil, ErrToolNotFound
	}

	content, err := tool.Execute(ctx, call.Arguments)
	result := &ToolResult{
		CallID:  call.ID,
		Content: content,
		IsError: err != nil,
	}

	if err != nil {
		result.Content = err.Error()
	}

	return result, nil
}

// ValidateArguments validates arguments against a JSON Schema.
// This is a simplified validation - a full implementation would use a JSON Schema library.
func ValidateArguments(schema json.RawMessage, args map[string]any) error {
	// Parse schema
	var schemaObj map[string]any
	if err := json.Unmarshal(schema, &schemaObj); err != nil {
		return err
	}

	// Check required fields
	if required, ok := schemaObj["required"].([]any); ok {
		for _, req := range required {
			if reqStr, ok := req.(string); ok {
				if _, exists := args[reqStr]; !exists {
					return errors.New("missing required field: " + reqStr)
				}
			}
		}
	}

	return nil
}

// FunctionTool wraps a simple function as a Tool.
type FunctionTool struct {
	name        string
	description string
	schema      json.RawMessage
	fn          func(ctx context.Context, args map[string]any) (string, error)
}

// NewFunctionTool creates a tool from a function.
func NewFunctionTool(name, description string, schema json.RawMessage, fn func(ctx context.Context, args map[string]any) (string, error)) *FunctionTool {
	return &FunctionTool{
		name:        name,
		description: description,
		schema:      schema,
		fn:          fn,
	}
}

// Definition returns the tool's definition.
func (t *FunctionTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        t.name,
		Description: t.description,
		InputSchema: t.schema,
	}
}

// Execute runs the wrapped function.
func (t *FunctionTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	return t.fn(ctx, args)
}
