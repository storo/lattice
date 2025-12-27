package registry

import (
	"context"
	"sync"

	"github.com/storo/lattice/pkg/core"
)

// Local is an in-memory registry implementation.
type Local struct {
	mu     sync.RWMutex
	agents map[string]core.Agent
}

// NewLocal creates a new local (in-memory) registry.
func NewLocal() *Local {
	return &Local{
		agents: make(map[string]core.Agent),
	}
}

// Register adds an agent to the registry.
func (r *Local) Register(ctx context.Context, agent core.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.agents[agent.ID()] = agent
	return nil
}

// Deregister removes an agent from the registry.
func (r *Local) Deregister(ctx context.Context, agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.agents, agentID)
	return nil
}

// Get retrieves an agent by ID.
func (r *Local) Get(ctx context.Context, agentID string) (core.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, ok := r.agents[agentID]
	if !ok {
		return nil, ErrAgentNotFound
	}
	return agent, nil
}

// FindByCapability finds all agents that provide a capability.
func (r *Local) FindByCapability(ctx context.Context, cap core.Capability) ([]core.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []core.Agent
	for _, agent := range r.agents {
		for _, provided := range agent.Provides() {
			if provided == cap {
				result = append(result, agent)
				break
			}
		}
	}
	return result, nil
}

// List returns all registered agents.
func (r *Local) List(ctx context.Context) ([]core.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]core.Agent, 0, len(r.agents))
	for _, agent := range r.agents {
		result = append(result, agent)
	}
	return result, nil
}

// Verify Local implements Registry
var _ Registry = (*Local)(nil)
