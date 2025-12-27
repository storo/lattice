package patterns

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/storo/lattice/pkg/core"
)

// ReAct pattern errors
var (
	ErrMaxIterationsReached = errors.New("maximum iterations reached")
)

// DefaultMaxIterations is the default maximum number of reasoning iterations.
const DefaultMaxIterations = 10

// ReActAgent wraps an agent with the ReAct (Reasoning + Acting) pattern.
// The agent alternates between thinking (reasoning) and acting (tool use)
// until it reaches a final answer or exceeds max iterations.
type ReActAgent struct {
	agent         core.Agent
	maxIterations int
}

// ReActOption configures the ReAct agent.
type ReActOption func(*ReActAgent)

// NewReActAgent creates a new ReAct-pattern agent.
func NewReActAgent(agent core.Agent, opts ...ReActOption) *ReActAgent {
	r := &ReActAgent{
		agent:         agent,
		maxIterations: DefaultMaxIterations,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// WithMaxIterations sets the maximum number of reasoning iterations.
func WithMaxIterations(n int) ReActOption {
	return func(r *ReActAgent) {
		r.maxIterations = n
	}
}

// Run executes the ReAct agent with reasoning iterations.
func (r *ReActAgent) Run(ctx context.Context, input string) (*core.Result, error) {
	// The underlying agent handles the tool loop.
	// ReAct adds iteration limiting on top for multi-step reasoning.
	iterations := 0
	currentInput := input
	var lastResult *core.Result

	for iterations < r.maxIterations {
		iterations++

		result, err := r.agent.Run(ctx, currentInput)
		if err != nil {
			return nil, err
		}
		lastResult = result

		// Check if the output contains an Action (meaning more reasoning needed)
		_, action := ParseThoughtAction(result.Output)
		if action == "" {
			// No action specified, this is the final answer
			return result, nil
		}

		// Continue with the result as additional context
		currentInput = result.Output
	}

	// If we have a result from the last iteration, return it with the error
	if lastResult != nil {
		return lastResult, ErrMaxIterationsReached
	}

	return nil, ErrMaxIterationsReached
}

// RunStream executes the ReAct agent with streaming output.
func (r *ReActAgent) RunStream(ctx context.Context, input string) (<-chan core.StreamChunk, error) {
	return r.agent.RunStream(ctx, input)
}

// ID returns the underlying agent's ID.
func (r *ReActAgent) ID() string {
	return r.agent.ID()
}

// Name returns the underlying agent's name.
func (r *ReActAgent) Name() string {
	return r.agent.Name()
}

// Description returns the underlying agent's description.
func (r *ReActAgent) Description() string {
	return r.agent.Description()
}

// Provides returns capabilities provided by the underlying agent.
func (r *ReActAgent) Provides() []core.Capability {
	return r.agent.Provides()
}

// Needs returns capabilities needed by the underlying agent.
func (r *ReActAgent) Needs() []core.Capability {
	return r.agent.Needs()
}

// Tools returns tools available to the underlying agent.
func (r *ReActAgent) Tools() []core.Tool {
	return r.agent.Tools()
}

// Stop stops the underlying agent.
func (r *ReActAgent) Stop() error {
	return r.agent.Stop()
}

// Card returns the underlying agent's card.
func (r *ReActAgent) Card() *core.AgentCard {
	return r.agent.Card()
}

// ParseThoughtAction parses a ReAct-formatted response into thought and action.
// Returns empty strings if the format is not recognized.
func ParseThoughtAction(response string) (thought, action string) {
	lines := strings.Split(response, "\n")

	thoughtPattern := regexp.MustCompile(`(?i)^Thought:\s*(.+)$`)
	actionPattern := regexp.MustCompile(`(?i)^Action:\s*(.+)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if matches := thoughtPattern.FindStringSubmatch(line); len(matches) > 1 {
			thought = matches[1]
		}
		if matches := actionPattern.FindStringSubmatch(line); len(matches) > 1 {
			action = matches[1]
		}
	}

	return thought, action
}

// FormatReActPrompt creates a ReAct-style system prompt.
func FormatReActPrompt(basePrompt string, tools []core.Tool) string {
	var sb strings.Builder
	sb.WriteString(basePrompt)
	sb.WriteString("\n\n")
	sb.WriteString("You are a ReAct agent. For each step:\n")
	sb.WriteString("1. Think about what you need to do (Thought: ...)\n")
	sb.WriteString("2. Choose an action to take (Action: tool_name)\n")
	sb.WriteString("3. Observe the result (Observation: ...)\n")
	sb.WriteString("4. Repeat until you have an answer (Answer: ...)\n\n")

	if len(tools) > 0 {
		sb.WriteString("Available tools:\n")
		for _, tool := range tools {
			sb.WriteString("- ")
			sb.WriteString(tool.Name())
			sb.WriteString(": ")
			sb.WriteString(tool.Description())
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// Verify ReActAgent implements core.Agent
var _ core.Agent = (*ReActAgent)(nil)
