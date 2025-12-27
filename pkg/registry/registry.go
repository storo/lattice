package registry

import (
	"context"
	"errors"

	"github.com/storo/lettice/pkg/core"
)

// Errors
var (
	ErrAgentNotFound = errors.New("agent not found")
)

// Registry is the interface for agent discovery and registration.
type Registry interface {
	// Register adds an agent to the registry.
	Register(ctx context.Context, agent core.Agent) error

	// Deregister removes an agent from the registry.
	Deregister(ctx context.Context, agentID string) error

	// Get retrieves an agent by ID.
	Get(ctx context.Context, agentID string) (core.Agent, error)

	// FindByCapability finds all agents that provide a capability.
	FindByCapability(ctx context.Context, cap core.Capability) ([]core.Agent, error)

	// List returns all registered agents.
	List(ctx context.Context) ([]core.Agent, error)
}
