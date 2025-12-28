// Package ollama provides a client for the Ollama API.
// Ollama enables running LLMs locally without requiring an API key.
package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/storo/lattice/pkg/core"
	"github.com/storo/lattice/pkg/provider"
)

const (
	defaultBaseURL   = "http://localhost:11434"
	defaultModel     = "llama3.2"
	defaultMaxTokens = 4096
)

// Client is an Ollama API client implementing provider.Provider.
type Client struct {
	baseURL    string
	model      string
	maxTokens  int
	httpClient *http.Client
}

// Option configures the Ollama client.
type Option func(*Client)

// NewClient creates a new Ollama client.
// By default connects to localhost:11434 with llama3.2 model.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL:   defaultBaseURL,
		model:     defaultModel,
		maxTokens: defaultMaxTokens,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Longer timeout for local inference
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithBaseURL sets a custom base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithModel sets the model to use.
func WithModel(model string) Option {
	return func(c *Client) {
		c.model = model
	}
}

// WithMaxTokens sets the maximum number of tokens to generate.
func WithMaxTokens(n int) Option {
	return func(c *Client) {
		c.maxTokens = n
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// Name returns the provider name.
func (c *Client) Name() string {
	return "ollama"
}

// Chat sends a chat request and returns a response.
func (c *Client) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	ollamaReq := c.buildRequest(req, false)

	respBody, err := c.doRequest(ctx, ollamaReq)
	if err != nil {
		return nil, err
	}

	var resp chatResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return c.convertResponse(&resp), nil
}

// ChatStream sends a chat request and returns a stream of events.
func (c *Client) ChatStream(ctx context.Context, req *provider.ChatRequest) (<-chan provider.StreamEvent, error) {
	ollamaReq := c.buildRequest(req, true)

	jsonBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(body))
	}

	events := make(chan provider.StreamEvent)

	go func() {
		defer close(events)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		var fullContent string
		var inputTokens, outputTokens int

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var streamResp streamResponse
			if err := json.Unmarshal(line, &streamResp); err != nil {
				events <- provider.StreamEvent{
					Type:  provider.EventTypeError,
					Error: fmt.Sprintf("failed to parse stream: %v", err),
				}
				return
			}

			if streamResp.Done {
				// Final event
				inputTokens = streamResp.PromptEvalCount
				outputTokens = streamResp.EvalCount

				// Check for tool calls
				if len(streamResp.Message.ToolCalls) > 0 {
					for _, tc := range streamResp.Message.ToolCalls {
						events <- provider.StreamEvent{
							Type: provider.EventTypeToolCall,
							ToolCall: &core.ToolCall{
								ID:     uuid.New().String(),
								Name:   tc.Function.Name,
								Params: tc.Function.Arguments,
							},
						}
					}
					events <- provider.StreamEvent{
						Type:       provider.EventTypeStop,
						StopReason: provider.StopReasonToolUse,
						Usage: &provider.Usage{
							InputTokens:  inputTokens,
							OutputTokens: outputTokens,
						},
					}
				} else {
					events <- provider.StreamEvent{
						Type:       provider.EventTypeStop,
						StopReason: provider.StopReasonEndTurn,
						Usage: &provider.Usage{
							InputTokens:  inputTokens,
							OutputTokens: outputTokens,
						},
					}
				}
				return
			}

			// Delta event
			if streamResp.Message.Content != "" {
				fullContent += streamResp.Message.Content
				events <- provider.StreamEvent{
					Type:  provider.EventTypeDelta,
					Delta: streamResp.Message.Content,
				}
			}
		}

		if err := scanner.Err(); err != nil {
			events <- provider.StreamEvent{
				Type:  provider.EventTypeError,
				Error: fmt.Sprintf("stream error: %v", err),
			}
		}
	}()

	return events, nil
}

// buildRequest converts a provider.ChatRequest to an Ollama request.
func (c *Client) buildRequest(req *provider.ChatRequest, stream bool) *chatRequest {
	model := c.model
	if req.Model != "" {
		model = req.Model
	}

	messages := c.convertMessages(req.Messages, req.System)

	ollamaReq := &chatRequest{
		Model:    model,
		Messages: messages,
		Stream:   stream,
	}

	// Add options
	if c.maxTokens > 0 || req.Temperature > 0 || len(req.StopSequences) > 0 {
		ollamaReq.Options = &options{}
		if c.maxTokens > 0 {
			ollamaReq.Options.NumPredict = c.maxTokens
		}
		if req.MaxTokens > 0 {
			ollamaReq.Options.NumPredict = req.MaxTokens
		}
		if req.Temperature > 0 {
			ollamaReq.Options.Temperature = req.Temperature
		}
		if len(req.StopSequences) > 0 {
			ollamaReq.Options.Stop = req.StopSequences
		}
	}

	// Add tools
	if len(req.Tools) > 0 {
		ollamaReq.Tools = c.convertTools(req.Tools)
	}

	return ollamaReq
}

// convertMessages converts core.Message to Ollama format.
func (c *Client) convertMessages(messages []core.Message, systemPrompt string) []message {
	var result []message

	// Add system message first if present
	if systemPrompt != "" {
		result = append(result, message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	for _, m := range messages {
		msg := message{
			Role:    string(m.Role),
			Content: m.Content,
		}

		// Handle tool calls
		if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				msg.ToolCalls = append(msg.ToolCalls, toolCall{
					Function: functionCall{
						Name:      tc.Name,
						Arguments: tc.Params,
					},
				})
			}
		}

		// Handle tool results - convert to assistant acknowledgment format
		if m.ToolResult != nil {
			msg.Role = "tool"
			msg.Content = m.ToolResult.Content
		}

		result = append(result, msg)
	}

	return result
}

// convertTools converts provider.ToolDefinition to Ollama format.
func (c *Client) convertTools(tools []provider.ToolDefinition) []tool {
	var result []tool
	for _, t := range tools {
		result = append(result, tool{
			Type: "function",
			Function: function{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}
	return result
}

// convertResponse converts Ollama response to provider.ChatResponse.
func (c *Client) convertResponse(resp *chatResponse) *provider.ChatResponse {
	result := &provider.ChatResponse{
		Content:    resp.Message.Content,
		StopReason: provider.StopReasonEndTurn,
		Usage: provider.Usage{
			InputTokens:  resp.PromptEvalCount,
			OutputTokens: resp.EvalCount,
		},
	}

	// Handle tool calls
	if len(resp.Message.ToolCalls) > 0 {
		result.StopReason = provider.StopReasonToolUse
		for _, tc := range resp.Message.ToolCalls {
			result.ToolCalls = append(result.ToolCalls, core.ToolCall{
				ID:     uuid.New().String(),
				Name:   tc.Function.Name,
				Params: tc.Function.Arguments,
			})
		}
	}

	// Map done_reason if present
	switch resp.DoneReason {
	case "stop":
		result.StopReason = provider.StopReasonEndTurn
	case "length":
		result.StopReason = provider.StopReasonMaxTokens
	}

	return result
}

// doRequest sends a request to the Ollama API.
func (c *Client) doRequest(ctx context.Context, req *chatRequest) ([]byte, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("ollama error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// Compile-time check that Client implements provider.Provider.
var _ provider.Provider = (*Client)(nil)
