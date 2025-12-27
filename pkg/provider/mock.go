package provider

import (
	"context"
)

// MockProvider implements Provider for testing.
type MockProvider struct {
	ChatFunc       func(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	ChatStreamFunc func(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error)
	ProviderName   string
}

// Chat implements Provider.
func (m *MockProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if m.ChatFunc != nil {
		return m.ChatFunc(ctx, req)
	}
	return &ChatResponse{
		Content:    "mock response",
		StopReason: StopReasonEndTurn,
	}, nil
}

// ChatStream implements Provider.
func (m *MockProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error) {
	if m.ChatStreamFunc != nil {
		return m.ChatStreamFunc(ctx, req)
	}
	ch := make(chan StreamEvent)
	go func() {
		defer close(ch)
		ch <- StreamEvent{Type: EventTypeStart}
		ch <- StreamEvent{Type: EventTypeDelta, Delta: "mock response"}
		ch <- StreamEvent{Type: EventTypeStop, StopReason: StopReasonEndTurn}
	}()
	return ch, nil
}

// Name implements Provider.
func (m *MockProvider) Name() string {
	if m.ProviderName != "" {
		return m.ProviderName
	}
	return "mock"
}

// NewMock creates a new mock provider with default behavior.
func NewMock() *MockProvider {
	return &MockProvider{}
}

// NewMockWithResponse creates a mock provider that returns a fixed response.
func NewMockWithResponse(content string) *MockProvider {
	return &MockProvider{
		ChatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			return &ChatResponse{
				Content:    content,
				StopReason: StopReasonEndTurn,
			}, nil
		},
	}
}
