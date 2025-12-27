package patterns

import (
	"context"
	"errors"
	"sync"

	"github.com/storo/lattice/pkg/core"
)

// Supervisor errors
var (
	ErrNoWorkerAvailable = errors.New("no worker available for capability")
)

// Strategy defines how the supervisor selects workers.
type Strategy string

const (
	// StrategyRoundRobin cycles through available workers.
	StrategyRoundRobin Strategy = "round_robin"

	// StrategyRandom selects a random worker.
	StrategyRandom Strategy = "random"

	// StrategyRaceFirst sends to all workers and returns the first result.
	StrategyRaceFirst Strategy = "race_first"

	// StrategyAll sends to all workers and aggregates results.
	StrategyAll Strategy = "all"
)

// Supervisor manages a group of worker agents and delegates tasks.
type Supervisor struct {
	mu       sync.RWMutex
	workers  map[string]core.Agent
	strategy Strategy
	rrIndex  map[core.Capability]int // round-robin indices
	healthy  map[string]bool
}

// SupervisorOption configures the supervisor.
type SupervisorOption func(*Supervisor)

// NewSupervisor creates a new supervisor.
func NewSupervisor(opts ...SupervisorOption) *Supervisor {
	s := &Supervisor{
		workers:  make(map[string]core.Agent),
		strategy: StrategyRoundRobin,
		rrIndex:  make(map[core.Capability]int),
		healthy:  make(map[string]bool),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// WithWorkers adds initial workers to the supervisor.
func WithWorkers(workers ...core.Agent) SupervisorOption {
	return func(s *Supervisor) {
		for _, w := range workers {
			s.workers[w.ID()] = w
			s.healthy[w.ID()] = true
		}
	}
}

// WithStrategy sets the delegation strategy.
func WithStrategy(strategy Strategy) SupervisorOption {
	return func(s *Supervisor) {
		s.strategy = strategy
	}
}

// AddWorker adds a worker to the supervisor.
func (s *Supervisor) AddWorker(worker core.Agent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.workers[worker.ID()] = worker
	s.healthy[worker.ID()] = true
}

// RemoveWorker removes a worker from the supervisor.
func (s *Supervisor) RemoveWorker(workerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.workers, workerID)
	delete(s.healthy, workerID)
}

// WorkersFor returns all workers that provide a capability.
func (s *Supervisor) WorkersFor(cap core.Capability) []core.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []core.Agent
	for _, worker := range s.workers {
		for _, provided := range worker.Provides() {
			if provided == cap {
				result = append(result, worker)
				break
			}
		}
	}
	return result
}

// Delegate sends a task to a worker with the required capability.
func (s *Supervisor) Delegate(ctx context.Context, cap core.Capability, input string) (*core.Result, error) {
	workers := s.WorkersFor(cap)
	if len(workers) == 0 {
		return nil, ErrNoWorkerAvailable
	}

	switch s.strategy {
	case StrategyRaceFirst:
		return s.raceFirst(ctx, workers, input)
	case StrategyRoundRobin:
		return s.roundRobin(ctx, cap, workers, input)
	default:
		// Default to first worker
		return workers[0].Run(ctx, input)
	}
}

// Broadcast sends a task to all workers with the capability.
func (s *Supervisor) Broadcast(ctx context.Context, cap core.Capability, input string) ([]*core.Result, []error) {
	workers := s.WorkersFor(cap)
	if len(workers) == 0 {
		return nil, []error{ErrNoWorkerAvailable}
	}

	// Run all workers in parallel
	type resultPair struct {
		result *core.Result
		err    error
	}

	resultsCh := make(chan resultPair, len(workers))

	for _, worker := range workers {
		w := worker
		go func() {
			result, err := w.Run(ctx, input)
			resultsCh <- resultPair{result, err}
		}()
	}

	// Collect results
	var results []*core.Result
	var errs []error

	for i := 0; i < len(workers); i++ {
		pair := <-resultsCh
		if pair.err != nil {
			errs = append(errs, pair.err)
		} else {
			results = append(results, pair.result)
		}
	}

	return results, errs
}

// Health returns the health status of all workers.
func (s *Supervisor) Health() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]bool)
	for id, healthy := range s.healthy {
		result[id] = healthy
	}
	return result
}

// SetHealth sets the health status of a worker.
func (s *Supervisor) SetHealth(workerID string, healthy bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthy[workerID] = healthy
}

// roundRobin selects the next worker in round-robin order.
func (s *Supervisor) roundRobin(ctx context.Context, cap core.Capability, workers []core.Agent, input string) (*core.Result, error) {
	s.mu.Lock()
	idx := s.rrIndex[cap]
	s.rrIndex[cap] = (idx + 1) % len(workers)
	s.mu.Unlock()

	return workers[idx].Run(ctx, input)
}

// raceFirst sends to all workers and returns the first result.
func (s *Supervisor) raceFirst(ctx context.Context, workers []core.Agent, input string) (*core.Result, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type resultPair struct {
		result *core.Result
		err    error
	}

	resultsCh := make(chan resultPair, len(workers))

	for _, worker := range workers {
		w := worker
		go func() {
			result, err := w.Run(ctx, input)
			select {
			case resultsCh <- resultPair{result, err}:
			case <-ctx.Done():
			}
		}()
	}

	// Wait for first successful result
	var lastErr error
	for i := 0; i < len(workers); i++ {
		pair := <-resultsCh
		if pair.err == nil {
			return pair.result, nil
		}
		lastErr = pair.err
	}

	return nil, lastErr
}
