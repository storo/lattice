package patterns

import (
	"context"
	"errors"
	"time"

	"github.com/storo/lettice/pkg/core"
)

// Pipeline errors
var (
	ErrEmptyPipeline = errors.New("pipeline has no agents")
)

// Sequential executes agents in sequence, passing output to the next input.
type Sequential struct {
	agents []core.Agent
}

// NewSequential creates a new sequential pipeline.
func NewSequential(agents ...core.Agent) *Sequential {
	return &Sequential{
		agents: agents,
	}
}

// Run executes the pipeline sequentially.
// Each agent receives the previous agent's output as input.
func (s *Sequential) Run(ctx context.Context, input string) (*core.Result, error) {
	if len(s.agents) == 0 {
		return nil, ErrEmptyPipeline
	}

	start := time.Now()
	currentInput := input

	var totalTokensIn, totalTokensOut int
	var lastResult *core.Result

	for _, agent := range s.agents {
		result, err := agent.Run(ctx, currentInput)
		if err != nil {
			return nil, err
		}

		totalTokensIn += result.TokensIn
		totalTokensOut += result.TokensOut
		currentInput = result.Output
		lastResult = result
	}

	// Return the final result with accumulated metrics
	return &core.Result{
		Output:    lastResult.Output,
		TokensIn:  totalTokensIn,
		TokensOut: totalTokensOut,
		Duration:  time.Since(start),
		TraceID:   lastResult.TraceID,
		CallChain: lastResult.CallChain,
		Metadata:  lastResult.Metadata,
	}, nil
}

// Agents returns the agents in the pipeline.
func (s *Sequential) Agents() []core.Agent {
	return s.agents
}

// Append adds agents to the end of the pipeline.
func (s *Sequential) Append(agents ...core.Agent) {
	s.agents = append(s.agents, agents...)
}

// Prepend adds agents to the beginning of the pipeline.
func (s *Sequential) Prepend(agents ...core.Agent) {
	s.agents = append(agents, s.agents...)
}
