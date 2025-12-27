package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/storo/lettice/pkg/core"
	"github.com/storo/lettice/pkg/provider"
)

const (
	defaultBaseURL    = "https://api.anthropic.com"
	defaultModel      = "claude-sonnet-4-20250514"
	defaultMaxTokens  = 4096
	apiVersion        = "2023-06-01"
	defaultMaxRetries = 3
)

// Client is the Anthropic API client.
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	maxTokens  int
	maxRetries int
	httpClient *http.Client
}

// Option configures the client.
type Option func(*Client)

// NewClient creates a new Anthropic client.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		model:      defaultModel,
		maxTokens:  defaultMaxTokens,
		maxRetries: defaultMaxRetries,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithBaseURL sets a custom base URL (useful for testing).
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

// WithMaxTokens sets the maximum tokens to generate.
func WithMaxTokens(n int) Option {
	return func(c *Client) {
		c.maxTokens = n
	}
}

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(n int) Option {
	return func(c *Client) {
		c.maxRetries = n
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
	return "anthropic"
}

// Chat sends a chat request and returns the response.
func (c *Client) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	// Build request body
	body := messagesRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		Messages:  convertMessages(req.Messages),
	}

	if req.System != "" {
		body.System = req.System
	}

	if len(req.Tools) > 0 {
		body.Tools = convertTools(req.Tools)
	}

	if req.MaxTokens > 0 {
		body.MaxTokens = req.MaxTokens
	}

	// Make request with retries
	var resp *messagesResponse
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			time.Sleep(time.Duration(attempt*attempt) * 100 * time.Millisecond)
		}

		resp, lastErr = c.doRequest(ctx, body)
		if lastErr == nil {
			break
		}

		// Don't retry on client errors
		if apiErr, ok := lastErr.(*APIError); ok && apiErr.StatusCode < 500 {
			return nil, lastErr
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return convertResponse(resp), nil
}

// ChatStream sends a chat request and returns a streaming response.
func (c *Client) ChatStream(ctx context.Context, req *provider.ChatRequest) (<-chan provider.StreamEvent, error) {
	// Build request body with streaming
	body := messagesRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		Messages:  convertMessages(req.Messages),
		Stream:    true,
	}

	if req.System != "" {
		body.System = req.System
	}

	if len(req.Tools) > 0 {
		body.Tools = convertTools(req.Tools)
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, parseError(resp)
	}

	ch := make(chan provider.StreamEvent)

	go func() {
		defer close(ch)
		defer resp.Body.Close()
		c.processStream(ctx, resp.Body, ch)
	}()

	return ch, nil
}

// doRequest makes a single API request.
func (c *Client) doRequest(ctx context.Context, body messagesRequest) (*messagesResponse, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var result messagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// setHeaders sets the required API headers.
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", apiVersion)
}

// processStream processes a streaming response.
func (c *Client) processStream(ctx context.Context, body io.Reader, ch chan<- provider.StreamEvent) {
	decoder := json.NewDecoder(body)

	for {
		select {
		case <-ctx.Done():
			ch <- provider.StreamEvent{
				Type:  provider.EventTypeError,
				Error: ctx.Err().Error(),
			}
			return
		default:
		}

		var event streamEvent
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				return
			}
			ch <- provider.StreamEvent{
				Type:  provider.EventTypeError,
				Error: err.Error(),
			}
			return
		}

		ch <- convertStreamEvent(&event)

		if event.Type == "message_stop" {
			return
		}
	}
}

// parseError parses an API error response.
func parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errResp struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Type:       errResp.Error.Type,
		Message:    errResp.Error.Message,
	}
}

// convertMessages converts core.Message to API format.
func convertMessages(messages []core.Message) []message {
	result := make([]message, 0, len(messages))

	for _, m := range messages {
		msg := message{
			Role: string(m.Role),
		}

		// Handle tool results
		if m.ToolResult != nil {
			msg.Content = []contentBlock{
				{
					Type:      "tool_result",
					ToolUseID: m.ToolResult.CallID,
					Content:   m.ToolResult.Content,
				},
			}
		} else if len(m.ToolCalls) > 0 {
			// Handle tool calls from assistant
			for _, tc := range m.ToolCalls {
				msg.Content = append(msg.Content, contentBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: parseToolInput(tc.Params),
				})
			}
		} else {
			// Regular text content
			msg.Content = []contentBlock{
				{Type: "text", Text: m.Content},
			}
		}

		result = append(result, msg)
	}

	return result
}

// parseToolInput parses JSON tool input.
func parseToolInput(params json.RawMessage) map[string]any {
	var result map[string]any
	json.Unmarshal(params, &result)
	return result
}

// convertTools converts tool definitions to API format.
func convertTools(tools []provider.ToolDefinition) []tool {
	result := make([]tool, 0, len(tools))

	for _, t := range tools {
		result = append(result, tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}

	return result
}

// convertResponse converts API response to provider.ChatResponse.
func convertResponse(resp *messagesResponse) *provider.ChatResponse {
	result := &provider.ChatResponse{
		Usage: provider.Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
	}

	// Map stop reason
	switch resp.StopReason {
	case "end_turn":
		result.StopReason = provider.StopReasonEndTurn
	case "tool_use":
		result.StopReason = provider.StopReasonToolUse
	case "max_tokens":
		result.StopReason = provider.StopReasonMaxTokens
	default:
		result.StopReason = provider.StopReasonEndTurn
	}

	// Extract content and tool calls
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			result.Content = block.Text
		case "tool_use":
			params, _ := json.Marshal(block.Input)
			result.ToolCalls = append(result.ToolCalls, core.ToolCall{
				ID:     block.ID,
				Name:   block.Name,
				Params: params,
			})
		}
	}

	return result
}

// convertStreamEvent converts a stream event to provider format.
func convertStreamEvent(event *streamEvent) provider.StreamEvent {
	switch event.Type {
	case "message_start":
		return provider.StreamEvent{Type: provider.EventTypeStart}
	case "content_block_delta":
		if event.Delta.Type == "text_delta" {
			return provider.StreamEvent{
				Type:  provider.EventTypeDelta,
				Delta: event.Delta.Text,
			}
		}
	case "message_stop":
		return provider.StreamEvent{Type: provider.EventTypeStop}
	}
	return provider.StreamEvent{}
}

// APIError represents an API error.
type APIError struct {
	StatusCode int
	Type       string
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("anthropic: %s - %s (status %d)", e.Type, e.Message, e.StatusCode)
}

// Verify Client implements provider.Provider
var _ provider.Provider = (*Client)(nil)
