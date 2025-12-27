package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/storo/lattice/pkg/core"
	"github.com/storo/lattice/pkg/provider"
	"github.com/storo/lattice/pkg/storage"
)

// Errors
var (
	ErrNoProvider = errors.New("no provider configured")
)

// Agent is the implementation of core.Agent.
type Agent struct {
	id          string
	name        string
	description string
	system      string
	provider    provider.Provider
	tools       []core.Tool
	provides    []core.Capability
	needs       []core.Capability
	store       storage.Store
	maxTokens   int
	temperature float64
	card        *core.AgentCard

	mu       sync.Mutex
	cancelFn context.CancelFunc
}

// ID returns the agent's unique identifier.
func (a *Agent) ID() string {
	return a.id
}

// Name returns the agent's name.
func (a *Agent) Name() string {
	return a.name
}

// Description returns the agent's description.
func (a *Agent) Description() string {
	return a.description
}

// Provides returns the capabilities this agent provides.
func (a *Agent) Provides() []core.Capability {
	return a.provides
}

// Needs returns the capabilities this agent needs.
func (a *Agent) Needs() []core.Capability {
	return a.needs
}

// Tools returns the tools available to this agent.
func (a *Agent) Tools() []core.Tool {
	return a.tools
}

// Card returns the agent's card for discovery.
func (a *Agent) Card() *core.AgentCard {
	return a.card
}

// Run executes the agent with the given input.
func (a *Agent) Run(ctx context.Context, input string) (*core.Result, error) {
	start := time.Now()

	if a.provider == nil {
		return nil, ErrNoProvider
	}

	// Add this agent to the call chain
	ctx = core.WithCallChain(ctx, a.id)

	// Build initial messages
	messages := []core.Message{
		{Role: core.RoleUser, Content: input},
	}

	// Build tool definitions
	toolDefs := a.buildToolDefinitions()

	// Execute the agentic loop
	var totalInputTokens, totalOutputTokens int
	var finalContent string

	for {
		// Create the request
		req := &provider.ChatRequest{
			Model:       "", // Will be set by provider
			System:      a.system,
			Messages:    messages,
			Tools:       toolDefs,
			MaxTokens:   a.maxTokens,
			Temperature: a.temperature,
		}

		// Call the provider
		resp, err := a.provider.Chat(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("provider error: %w", err)
		}

		totalInputTokens += resp.Usage.InputTokens
		totalOutputTokens += resp.Usage.OutputTokens

		// Handle tool calls
		if resp.StopReason == provider.StopReasonToolUse && len(resp.ToolCalls) > 0 {
			// Add assistant message with tool calls
			messages = append(messages, core.Message{
				Role:      core.RoleAssistant,
				ToolCalls: resp.ToolCalls,
			})

			// Execute each tool and add results
			for _, toolCall := range resp.ToolCalls {
				result, err := a.executeTool(ctx, toolCall)

				toolResult := &core.ToolResult{
					CallID:  toolCall.ID,
					Content: result,
					IsError: err != nil,
				}
				if err != nil {
					toolResult.Content = err.Error()
				}

				messages = append(messages, core.Message{
					Role:       core.RoleTool,
					ToolResult: toolResult,
				})
			}

			// Continue the loop to get the final response
			continue
		}

		// No more tool calls, we're done
		finalContent = resp.Content
		break
	}

	return &core.Result{
		Output:    finalContent,
		TokensIn:  totalInputTokens,
		TokensOut: totalOutputTokens,
		Duration:  time.Since(start),
		TraceID:   core.TraceID(ctx),
		CallChain: core.CallChain(ctx),
	}, nil
}

// RunStream executes the agent with streaming output.
func (a *Agent) RunStream(ctx context.Context, input string) (<-chan core.StreamChunk, error) {
	if a.provider == nil {
		return nil, ErrNoProvider
	}

	ctx, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	a.cancelFn = cancel
	a.mu.Unlock()

	ch := make(chan core.StreamChunk)

	go func() {
		defer close(ch)

		// Add this agent to the call chain
		ctx = core.WithCallChain(ctx, a.id)

		// Build initial messages
		messages := []core.Message{
			{Role: core.RoleUser, Content: input},
		}

		// Build tool definitions
		toolDefs := a.buildToolDefinitions()

		// Create the request
		req := &provider.ChatRequest{
			Model:       "",
			System:      a.system,
			Messages:    messages,
			Tools:       toolDefs,
			MaxTokens:   a.maxTokens,
			Temperature: a.temperature,
		}

		// Get stream from provider
		stream, err := a.provider.ChatStream(ctx, req)
		if err != nil {
			ch <- core.StreamChunk{Error: err}
			return
		}

		// Forward events
		for event := range stream {
			select {
			case <-ctx.Done():
				ch <- core.StreamChunk{Error: ctx.Err()}
				return
			default:
			}

			switch event.Type {
			case provider.EventTypeDelta:
				ch <- core.StreamChunk{Content: event.Delta}
			case provider.EventTypeStop:
				ch <- core.StreamChunk{Done: true}
			case provider.EventTypeError:
				ch <- core.StreamChunk{Error: errors.New(event.Error)}
			}
		}
	}()

	return ch, nil
}

// Stop gracefully stops the agent.
func (a *Agent) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cancelFn != nil {
		a.cancelFn()
		a.cancelFn = nil
	}

	return nil
}

// buildToolDefinitions converts core.Tool to provider.ToolDefinition.
func (a *Agent) buildToolDefinitions() []provider.ToolDefinition {
	defs := make([]provider.ToolDefinition, len(a.tools))
	for i, tool := range a.tools {
		defs[i] = provider.ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.Schema(),
		}
	}
	return defs
}

// executeTool finds and executes a tool by name.
func (a *Agent) executeTool(ctx context.Context, call core.ToolCall) (string, error) {
	for _, tool := range a.tools {
		if tool.Name() == call.Name {
			return tool.Execute(ctx, call.Params)
		}
	}
	return "", fmt.Errorf("tool not found: %s", call.Name)
}

// AddTools adds tools to the agent dynamically.
// This is used by the mesh to inject delegation tools.
func (a *Agent) AddTools(tools ...core.Tool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tools = append(a.tools, tools...)
}

// Verify Agent implements core.Agent
var _ core.Agent = (*Agent)(nil)
