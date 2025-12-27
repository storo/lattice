package http

// RunRequest is the request body for running an agent.
type RunRequest struct {
	// Input is the task or prompt to send to the agent.
	Input string `json:"input"`

	// Context is optional additional context.
	Context string `json:"context,omitempty"`
}

// RunResponse is the response from running an agent.
type RunResponse struct {
	// Output is the agent's response.
	Output string `json:"output"`

	// TokensIn is the number of input tokens used.
	TokensIn int `json:"tokens_in"`

	// TokensOut is the number of output tokens generated.
	TokensOut int `json:"tokens_out"`

	// Duration is how long the execution took.
	Duration string `json:"duration"`

	// TraceID is the trace identifier for debugging.
	TraceID string `json:"trace_id,omitempty"`
}

// AgentInfo contains information about an agent.
type AgentInfo struct {
	// ID is the unique identifier.
	ID string `json:"id"`

	// Name is the human-readable name.
	Name string `json:"name"`

	// Description explains what the agent does.
	Description string `json:"description"`

	// Provides are the capabilities this agent provides.
	Provides []string `json:"provides"`

	// Needs are the capabilities this agent requires.
	Needs []string `json:"needs"`
}

// ListAgentsResponse is the response for listing agents.
type ListAgentsResponse struct {
	// Agents is the list of registered agents.
	Agents []AgentInfo `json:"agents"`
}

// ErrorResponse is an error response.
type ErrorResponse struct {
	// Error is the error message.
	Error string `json:"error"`
}
