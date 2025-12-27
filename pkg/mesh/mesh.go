package mesh

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/storo/lattice/pkg/agent"
	"github.com/storo/lattice/pkg/core"
	"github.com/storo/lattice/pkg/registry"
)

// Mesh is the central orchestrator for the agent mesh.
type Mesh struct {
	mu            sync.RWMutex
	registry      registry.Registry
	injector      *Injector
	balancer      Balancer
	cycleDetector *CycleDetector
}

// Option is a function that configures the mesh.
type Option func(*Mesh)

// New creates a new mesh with the given options.
func New(opts ...Option) *Mesh {
	reg := registry.NewLocal()
	balancer := NewRoundRobinBalancer()
	cycleDetector := NewCycleDetector(DefaultMaxHops)

	m := &Mesh{
		registry:      reg,
		balancer:      balancer,
		cycleDetector: cycleDetector,
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	// Create injector after options are applied
	m.injector = NewInjector(m.registry, m.balancer, m.cycleDetector)

	return m
}

// WithMaxHops sets the maximum number of hops for cycle detection.
func WithMaxHops(n int) Option {
	return func(m *Mesh) {
		m.cycleDetector = NewCycleDetector(n)
	}
}

// WithBalancer sets the load balancer strategy.
func WithBalancer(b Balancer) Option {
	return func(m *Mesh) {
		m.balancer = b
	}
}

// WithRegistry sets a custom registry.
func WithRegistry(r registry.Registry) Option {
	return func(m *Mesh) {
		m.registry = r
	}
}

// Register adds agents to the mesh.
func (m *Mesh) Register(agents ...core.Agent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx := context.Background()
	for _, a := range agents {
		if err := m.registry.Register(ctx, a); err != nil {
			return err
		}
	}
	return nil
}

// GetAgent retrieves an agent by ID.
func (m *Mesh) GetAgent(ctx context.Context, agentID string) (core.Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.registry.Get(ctx, agentID)
}

// ListAgents returns all registered agents.
func (m *Mesh) ListAgents(ctx context.Context) ([]core.Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.registry.List(ctx)
}

// FindProviders finds agents that provide a capability.
func (m *Mesh) FindProviders(ctx context.Context, cap core.Capability) ([]core.Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.registry.FindByCapability(ctx, cap)
}

// PrepareAgent injects delegation tools into an agent.
func (m *Mesh) PrepareAgent(ctx context.Context, a core.Agent) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools, err := m.injector.InjectTools(ctx, a)
	if err != nil {
		return err
	}

	// Add tools to the agent
	if agentImpl, ok := a.(*agent.Agent); ok {
		agentImpl.AddTools(tools...)
	}

	return nil
}

// RunAgent executes an agent by ID.
func (m *Mesh) RunAgent(ctx context.Context, agentID string, input string) (*core.Result, error) {
	// Ensure we have a trace ID
	if core.TraceID(ctx) == "" {
		ctx = core.WithTraceID(ctx, uuid.New().String())
	}

	// Get the agent
	a, err := m.GetAgent(ctx, agentID)
	if err != nil {
		return nil, err
	}

	// Prepare the agent with injected tools
	if err := m.PrepareAgent(ctx, a); err != nil {
		return nil, err
	}

	// Run the agent
	return a.Run(ctx, input)
}

// Run executes a task on the mesh by finding an appropriate agent.
// This is a simplified entry point that delegates to the first capable agent.
func (m *Mesh) Run(ctx context.Context, task string) (*core.Result, error) {
	// Ensure we have a trace ID
	if core.TraceID(ctx) == "" {
		ctx = core.WithTraceID(ctx, uuid.New().String())
	}

	// Get all agents
	agents, err := m.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	if len(agents) == 0 {
		return nil, registry.ErrAgentNotFound
	}

	// Use the first agent (simple strategy for now)
	a := agents[0]

	// Prepare and run
	if err := m.PrepareAgent(ctx, a); err != nil {
		return nil, err
	}

	return a.Run(ctx, task)
}
