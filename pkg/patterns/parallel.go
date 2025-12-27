package patterns

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/storo/lattice/pkg/core"
)

// Parallel errors
var (
	ErrAllAgentsFailed = errors.New("all agents failed")
)

// AgentResult holds the result from a single agent execution.
type AgentResult struct {
	AgentID string
	Output  string
	Result  *core.Result
	Error   error
}

// Parallel executes multiple agents concurrently.
type Parallel struct {
	agents []core.Agent
}

// NewParallel creates a new parallel executor.
func NewParallel(agents ...core.Agent) *Parallel {
	return &Parallel{
		agents: agents,
	}
}

// Run executes all agents in parallel with the same input.
// Returns all successful results and all errors.
func (p *Parallel) Run(ctx context.Context, input string) ([]*core.Result, []error) {
	if len(p.agents) == 0 {
		return nil, nil
	}

	resultsCh := make(chan AgentResult, len(p.agents))
	var wg sync.WaitGroup

	for _, agent := range p.agents {
		wg.Add(1)
		a := agent
		go func() {
			defer wg.Done()
			result, err := a.Run(ctx, input)
			resultsCh <- AgentResult{
				AgentID: a.ID(),
				Output:  "",
				Result:  result,
				Error:   err,
			}
		}()
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	// Collect results
	var results []*core.Result
	var errs []error

	for ar := range resultsCh {
		if ar.Error != nil {
			errs = append(errs, ar.Error)
		} else {
			results = append(results, ar.Result)
		}
	}

	return results, errs
}

// Race executes all agents and returns the first successful result.
func (p *Parallel) Race(ctx context.Context, input string) (*core.Result, error) {
	if len(p.agents) == 0 {
		return nil, ErrEmptyPipeline
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	resultsCh := make(chan AgentResult, len(p.agents))

	for _, agent := range p.agents {
		a := agent
		go func() {
			result, err := a.Run(ctx, input)
			select {
			case resultsCh <- AgentResult{AgentID: a.ID(), Result: result, Error: err}:
			case <-ctx.Done():
			}
		}()
	}

	// Wait for first successful result or all failures
	var lastErr error
	for i := 0; i < len(p.agents); i++ {
		ar := <-resultsCh
		if ar.Error == nil {
			return ar.Result, nil
		}
		lastErr = ar.Error
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrAllAgentsFailed
}

// Aggregate executes all agents and combines results using a custom function.
func (p *Parallel) Aggregate(ctx context.Context, input string, aggregator func([]*AgentResult) string) string {
	if len(p.agents) == 0 {
		return ""
	}

	resultsCh := make(chan AgentResult, len(p.agents))
	var wg sync.WaitGroup

	start := time.Now()

	for _, agent := range p.agents {
		wg.Add(1)
		a := agent
		go func() {
			defer wg.Done()
			result, err := a.Run(ctx, input)
			ar := AgentResult{
				AgentID: a.ID(),
				Error:   err,
			}
			if result != nil {
				ar.Output = result.Output
				ar.Result = result
			}
			resultsCh <- ar
		}()
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	// Collect all results
	var agentResults []*AgentResult
	for ar := range resultsCh {
		arCopy := ar
		if ar.Error == nil {
			agentResults = append(agentResults, &arCopy)
		}
	}

	_ = time.Since(start)

	return aggregator(agentResults)
}

// Agents returns the agents in the parallel executor.
func (p *Parallel) Agents() []core.Agent {
	return p.agents
}

// Add adds agents to the parallel executor.
func (p *Parallel) Add(agents ...core.Agent) {
	p.agents = append(p.agents, agents...)
}
