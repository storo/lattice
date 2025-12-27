package core

import (
	"context"
	"encoding/json"
	"time"
)

// Agent is the core interface for all agents in the mesh.
type Agent interface {
	// ID returns the unique identifier for the agent.
	ID() string

	// Name returns the human-readable name of the agent.
	Name() string

	// Description returns a description of what the agent does.
	Description() string

	// Provides returns the capabilities this agent provides.
	Provides() []Capability

	// Needs returns the capabilities this agent needs from other agents.
	Needs() []Capability

	// Run executes the agent with the given input and returns the result.
	Run(ctx context.Context, input string) (*Result, error)

	// RunStream executes the agent with streaming output.
	RunStream(ctx context.Context, input string) (<-chan StreamChunk, error)

	// Stop gracefully stops the agent.
	Stop() error

	// Card returns the agent's card for discovery.
	Card() *AgentCard

	// Tools returns the tools available to this agent.
	Tools() []Tool
}

// Tool is the interface for tools that can be used by agents.
type Tool interface {
	// Name returns the tool's name.
	Name() string

	// Description returns a description of what the tool does.
	Description() string

	// Schema returns the JSON Schema for the tool's parameters.
	Schema() json.RawMessage

	// Execute runs the tool with the given parameters.
	Execute(ctx context.Context, params json.RawMessage) (string, error)
}

// Result represents the output of an agent execution.
type Result struct {
	// Output is the main output from the agent.
	Output string

	// Metadata contains additional information about the execution.
	Metadata map[string]any

	// TokensIn is the number of input tokens used.
	TokensIn int

	// TokensOut is the number of output tokens generated.
	TokensOut int

	// Duration is how long the execution took.
	Duration time.Duration

	// TraceID is the trace identifier for distributed tracing.
	TraceID string

	// CallChain is the list of agent IDs in the call chain.
	CallChain []string

	// Error contains any error that occurred during execution.
	Error error
}

// StreamChunk represents a chunk of streaming output.
type StreamChunk struct {
	// Content is the text content of this chunk.
	Content string

	// Done indicates if this is the final chunk.
	Done bool

	// Error contains any error that occurred.
	Error error
}

// AgentCard is the discovery card for an agent (A2A compatible).
type AgentCard struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	URL          string            `json:"url"`
	Version      string            `json:"version"`
	Capabilities CardCapabilities  `json:"capabilities"`
	Skills       []Skill           `json:"skills"`
	Tools        []string          `json:"tools"`
	Model        string            `json:"model"`
	Protocols    []string          `json:"protocols"`
	InputSchema  json.RawMessage   `json:"input_schema,omitempty"`
	OutputSchema json.RawMessage   `json:"output_schema,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// CardCapabilities describes an agent's capability declarations.
type CardCapabilities struct {
	Provides  []Capability `json:"provides"`
	Needs     []Capability `json:"needs"`
	Streaming bool         `json:"streaming"`
}

// Skill represents a specific skill that an agent has.
type Skill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Role represents the role of a message in a conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// Message represents a message in a conversation.
type Message struct {
	Role       Role         `json:"role"`
	Content    string       `json:"content"`
	ToolCalls  []ToolCall   `json:"tool_calls,omitempty"`
	ToolResult *ToolResult  `json:"tool_result,omitempty"`
}

// ToolCall represents a request to execute a tool.
type ToolCall struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Params json.RawMessage `json:"params"`
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	CallID  string `json:"call_id"`
	Content string `json:"content"`
	IsError bool   `json:"is_error"`
}
