// Package lattice provides an Agent Mesh Framework for building distributed AI systems.
//
// Lattice enables creating networks of AI agents that can discover, communicate with,
// and delegate tasks to each other based on their capabilities.
//
// Basic usage:
//
//	// Create an agent
//	agent := lattice.Agent("researcher").
//		Model(claude).
//		System("You are a research expert").
//		Provides(lattice.CapResearch).
//		Build()
//
//	// Create a mesh and register agents
//	mesh := lattice.NewMesh(lattice.WithMaxHops(5))
//	mesh.Register(agent)
//
//	// Run a task
//	result, err := mesh.Run(ctx, "Research AI trends")
package lattice

import (
	"github.com/storo/lattice/pkg/agent"
	"github.com/storo/lattice/pkg/core"
	"github.com/storo/lattice/pkg/mesh"
	"github.com/storo/lattice/pkg/patterns"
	"github.com/storo/lattice/pkg/provider"
	"github.com/storo/lattice/pkg/registry"
	"github.com/storo/lattice/pkg/security"
)

// Re-export core types for convenience
type (
	// Agent is the agent interface.
	Agent = core.Agent

	// Tool is the tool interface.
	Tool = core.Tool

	// Result is the execution result.
	Result = core.Result

	// Capability is an agent capability.
	Capability = core.Capability

	// StreamChunk is a streaming output chunk.
	StreamChunk = core.StreamChunk

	// AgentCard is the agent's discovery card.
	AgentCard = core.AgentCard

	// Mesh is the agent mesh orchestrator.
	Mesh = mesh.Mesh

	// Provider is the LLM provider interface.
	Provider = provider.Provider
)

// Common capabilities
const (
	CapResearch Capability = core.CapResearch
	CapWriting  Capability = core.CapWriting
	CapCoding   Capability = core.CapCoding
	CapAnalysis Capability = core.CapAnalysis
	CapPlanning Capability = core.CapPlanning
)

// Cap creates a custom capability.
func Cap(name string) Capability {
	return core.Capability(name)
}

// NewAgent creates a new agent builder.
func NewAgent(name string) *agent.Builder {
	return agent.New(name)
}

// NewMesh creates a new agent mesh.
func NewMesh(opts ...mesh.Option) *mesh.Mesh {
	return mesh.New(opts...)
}

// WithMaxHops sets the maximum number of delegation hops.
func WithMaxHops(n int) mesh.Option {
	return mesh.WithMaxHops(n)
}

// WithBalancer sets a custom load balancer.
func WithBalancer(b mesh.Balancer) mesh.Option {
	return mesh.WithBalancer(b)
}

// WithRegistry sets a custom agent registry.
func WithRegistry(r registry.Registry) mesh.Option {
	return mesh.WithRegistry(r)
}

// Balancer strategies
var (
	NewRoundRobinBalancer = mesh.NewRoundRobinBalancer
	NewRandomBalancer     = mesh.NewRandomBalancer
	NewFirstBalancer      = mesh.NewFirstBalancer
)

// Pattern constructors
var (
	// NewReActAgent creates a ReAct pattern agent.
	NewReActAgent = patterns.NewReActAgent

	// NewSupervisor creates a supervisor for managing worker agents.
	NewSupervisor = patterns.NewSupervisor

	// NewSequential creates a sequential pipeline.
	NewSequential = patterns.NewSequential

	// NewParallel creates a parallel executor.
	NewParallel = patterns.NewParallel
)

// Security constructors
var (
	// NewAPIKeyAuth creates API key authentication.
	NewAPIKeyAuth = security.NewAPIKeyAuth

	// NewJWTAuth creates JWT authentication.
	NewJWTAuth = security.NewJWTAuth

	// NewAuth creates a unified authenticator.
	NewAuth = security.NewAuth
)

// Schema utilities
var (
	// SchemaFromStruct generates JSON Schema from a struct.
	SchemaFromStruct = core.SchemaFromStruct
)

// Context utilities
var (
	// WithTraceID adds a trace ID to context.
	WithTraceID = core.WithTraceID

	// TraceID retrieves the trace ID from context.
	TraceID = core.TraceID

	// CallChain retrieves the call chain from context.
	CallChain = core.CallChain
)

// Errors
var (
	// ErrCycleDetected is returned when a delegation cycle is detected.
	ErrCycleDetected = mesh.ErrCycleDetected

	// ErrMaxHopsExceeded is returned when max delegation hops is exceeded.
	ErrMaxHopsExceeded = mesh.ErrMaxHopsExceeded

	// ErrAgentNotFound is returned when an agent is not found.
	ErrAgentNotFound = registry.ErrAgentNotFound
)
