package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPTool makes HTTP requests.
type HTTPTool struct {
	client          *http.Client
	allowedDomains  []string
	maxResponseSize int64
	userAgent       string
}

// HTTPOption configures the HTTP tool.
type HTTPOption func(*HTTPTool)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) HTTPOption {
	return func(t *HTTPTool) {
		t.client = client
	}
}

// WithAllowedDomains restricts requests to specific domains.
func WithAllowedDomains(domains ...string) HTTPOption {
	return func(t *HTTPTool) {
		t.allowedDomains = domains
	}
}

// WithMaxResponseSize limits the response body size.
func WithMaxResponseSize(size int64) HTTPOption {
	return func(t *HTTPTool) {
		t.maxResponseSize = size
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) HTTPOption {
	return func(t *HTTPTool) {
		t.userAgent = ua
	}
}

// NewHTTPTool creates a new HTTP tool.
func NewHTTPTool(opts ...HTTPOption) *HTTPTool {
	t := &HTTPTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxResponseSize: 1024 * 1024, // 1MB
		userAgent:       "Lattice-Agent/1.0",
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Name returns the tool name.
func (t *HTTPTool) Name() string {
	return "http"
}

// Description returns the tool description.
func (t *HTTPTool) Description() string {
	return "Make HTTP requests. Supports GET, POST, PUT, DELETE methods with headers and body."
}

// Schema returns the JSON Schema for the tool parameters.
func (t *HTTPTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"method": {
				"type": "string",
				"enum": ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"],
				"description": "HTTP method"
			},
			"url": {
				"type": "string",
				"description": "The URL to request"
			},
			"headers": {
				"type": "object",
				"additionalProperties": {"type": "string"},
				"description": "Request headers"
			},
			"body": {
				"type": "string",
				"description": "Request body (for POST, PUT, PATCH)"
			}
		},
		"required": ["method", "url"]
	}`)
}

// httpParams are the parameters for the HTTP tool.
type httpParams struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// Execute runs the HTTP tool.
func (t *HTTPTool) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	var p httpParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	// Validate domain if restrictions are set
	if len(t.allowedDomains) > 0 {
		allowed := false
		for _, domain := range t.allowedDomains {
			if strings.Contains(p.URL, domain) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("domain not allowed: %s", p.URL)
		}
	}

	// Create request
	var body io.Reader
	if p.Body != "" {
		body = strings.NewReader(p.Body)
	}

	req, err := http.NewRequestWithContext(ctx, p.Method, p.URL, body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", t.userAgent)
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}

	// Execute request
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response (with size limit)
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, t.maxResponseSize))
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Format response
	result := struct {
		Status  int               `json:"status"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	}{
		Status:  resp.StatusCode,
		Headers: make(map[string]string),
		Body:    string(respBody),
	}

	// Copy relevant headers
	for _, h := range []string{"Content-Type", "Content-Length", "Last-Modified", "ETag"} {
		if v := resp.Header.Get(h); v != "" {
			result.Headers[h] = v
		}
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return string(output), nil
}
