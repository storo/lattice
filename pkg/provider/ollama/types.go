package ollama

import "encoding/json"

// chatRequest is the request body for Ollama's chat API.
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream"`
	Tools    []tool    `json:"tools,omitempty"`
	Options  *options  `json:"options,omitempty"`
}

// message represents a chat message in Ollama format.
type message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
}

// tool represents a tool definition for Ollama.
type tool struct {
	Type     string   `json:"type"` // "function"
	Function function `json:"function"`
}

// function describes a tool's function.
type function struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// options contains model parameters.
type options struct {
	NumPredict  int     `json:"num_predict,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

// chatResponse is the response from Ollama's chat API (non-streaming).
type chatResponse struct {
	Model              string  `json:"model"`
	CreatedAt          string  `json:"created_at"`
	Message            message `json:"message"`
	Done               bool    `json:"done"`
	DoneReason         string  `json:"done_reason,omitempty"`
	TotalDuration      int64   `json:"total_duration"`
	LoadDuration       int64   `json:"load_duration"`
	PromptEvalCount    int     `json:"prompt_eval_count"`
	PromptEvalDuration int64   `json:"prompt_eval_duration"`
	EvalCount          int     `json:"eval_count"`
	EvalDuration       int64   `json:"eval_duration"`
}

// streamResponse is a single event in the streaming response.
type streamResponse struct {
	Model     string  `json:"model"`
	CreatedAt string  `json:"created_at"`
	Message   message `json:"message"`
	Done      bool    `json:"done"`

	// Final response fields (only when done=true)
	DoneReason         string `json:"done_reason,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
}

// toolCall represents a tool call from the model.
type toolCall struct {
	Function functionCall `json:"function"`
}

// functionCall contains the function name and arguments.
type functionCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// errorResponse represents an error from the Ollama API.
type errorResponse struct {
	Error string `json:"error"`
}
