package provider

import (
	"context"
	"encoding/json"

	"github.com/storo/lettice/pkg/core"
)

// Provider is the interface for LLM providers.
type Provider interface {
	// Chat sends a chat request and returns a response.
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ChatStream sends a chat request and returns a stream of events.
	ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error)

	// Name returns the provider name.
	Name() string
}

// ChatRequest represents a request to the LLM.
type ChatRequest struct {
	// Model is the model identifier (e.g., "claude-3-opus").
	Model string

	// Messages is the conversation history.
	Messages []core.Message

	// System is the system prompt.
	System string

	// Tools available for the model to use.
	Tools []ToolDefinition

	// MaxTokens is the maximum number of tokens to generate.
	MaxTokens int

	// Temperature controls randomness (0.0 to 1.0).
	Temperature float64

	// StopSequences are strings that stop generation.
	StopSequences []string

	// Metadata for the request.
	Metadata map[string]string
}

// ToolDefinition describes a tool available to the model.
type ToolDefinition struct {
	// Name is the tool's identifier.
	Name string `json:"name"`

	// Description explains what the tool does.
	Description string `json:"description"`

	// InputSchema is the JSON Schema for the tool's parameters.
	InputSchema json.RawMessage `json:"input_schema"`
}

// ChatResponse represents a response from the LLM.
type ChatResponse struct {
	// Content is the text response.
	Content string

	// StopReason indicates why generation stopped.
	StopReason StopReason

	// ToolCalls are requests from the model to use tools.
	ToolCalls []core.ToolCall

	// Usage contains token usage information.
	Usage Usage
}

// StopReason indicates why the model stopped generating.
type StopReason string

const (
	StopReasonEndTurn      StopReason = "end_turn"
	StopReasonMaxTokens    StopReason = "max_tokens"
	StopReasonStopSequence StopReason = "stop_sequence"
	StopReasonToolUse      StopReason = "tool_use"
)

// Usage contains token usage information.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// StreamEvent represents an event in a streaming response.
type StreamEvent struct {
	// Type is the event type.
	Type EventType

	// Delta is the text content delta (for delta events).
	Delta string

	// ToolCall is the tool call (for tool_call events).
	ToolCall *core.ToolCall

	// StopReason is set for stop events.
	StopReason StopReason

	// Usage is set for final events.
	Usage *Usage

	// Error is set if an error occurred.
	Error string
}

// EventType is the type of streaming event.
type EventType string

const (
	EventTypeStart    EventType = "start"
	EventTypeDelta    EventType = "delta"
	EventTypeToolCall EventType = "tool_call"
	EventTypeStop     EventType = "stop"
	EventTypeError    EventType = "error"
)
