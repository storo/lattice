package mesh

import (
	"context"
	"errors"

	"github.com/storo/lettice/pkg/core"
)

// Default values
const (
	DefaultMaxHops = 10
)

// Errors
var (
	ErrCycleDetected   = errors.New("execution cycle detected")
	ErrMaxHopsExceeded = errors.New("maximum hop count exceeded")
)

// CycleDetector prevents infinite loops between agents.
type CycleDetector struct {
	maxHops int
}

// NewCycleDetector creates a new cycle detector with the specified max hops.
// If maxHops <= 0, DefaultMaxHops is used.
func NewCycleDetector(maxHops int) *CycleDetector {
	if maxHops <= 0 {
		maxHops = DefaultMaxHops
	}
	return &CycleDetector{maxHops: maxHops}
}

// Check verifies if it's safe to execute the given agent.
// Returns ErrCycleDetected if the agent is already in the call chain.
// Returns ErrMaxHopsExceeded if the maximum hop count would be exceeded.
func (cd *CycleDetector) Check(ctx context.Context, agentID string) error {
	// Check if agent is already in the call chain (cycle)
	if core.InCallChain(ctx, agentID) {
		return ErrCycleDetected
	}

	// Check hop count
	if core.HopCount(ctx) >= cd.maxHops {
		return ErrMaxHopsExceeded
	}

	return nil
}

// PrepareContext prepares the context for executing the agent.
// It adds the agent to the call chain and increments the hop count.
func (cd *CycleDetector) PrepareContext(ctx context.Context, agentID string) context.Context {
	ctx = core.WithCallChain(ctx, agentID)
	ctx = core.WithHopCount(ctx)
	return ctx
}

// MaxHops returns the configured maximum hop count.
func (cd *CycleDetector) MaxHops() int {
	return cd.maxHops
}
