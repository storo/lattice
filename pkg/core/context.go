package core

import "context"

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	callChainKey        contextKey = "lattice.call_chain"
	hopCountKey         contextKey = "lattice.hop_count"
	traceIDKey          contextKey = "lattice.trace_id"
	remainingTimeoutKey contextKey = "lattice.remaining_timeout"
)

// CallChain returns the chain of agent IDs that have been called.
// Returns nil if no chain exists.
func CallChain(ctx context.Context) []string {
	if chain, ok := ctx.Value(callChainKey).([]string); ok {
		return chain
	}
	return nil
}

// WithCallChain adds an agent ID to the call chain and returns a new context.
// The original context is not modified.
func WithCallChain(ctx context.Context, agentID string) context.Context {
	chain := CallChain(ctx)
	newChain := make([]string, len(chain)+1)
	copy(newChain, chain)
	newChain[len(chain)] = agentID
	return context.WithValue(ctx, callChainKey, newChain)
}

// InCallChain checks if an agent ID is already in the call chain.
// This is used for cycle detection.
func InCallChain(ctx context.Context, agentID string) bool {
	for _, id := range CallChain(ctx) {
		if id == agentID {
			return true
		}
	}
	return false
}

// HopCount returns the number of hops in the current call chain.
// Returns 0 if no hops have been recorded.
func HopCount(ctx context.Context) int {
	if count, ok := ctx.Value(hopCountKey).(int); ok {
		return count
	}
	return 0
}

// WithHopCount increments the hop count and returns a new context.
func WithHopCount(ctx context.Context) context.Context {
	return context.WithValue(ctx, hopCountKey, HopCount(ctx)+1)
}

// TraceID returns the trace ID for the current execution.
// Returns empty string if no trace ID is set.
func TraceID(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey).(string); ok {
		return id
	}
	return ""
}

// WithTraceID sets the trace ID and returns a new context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// RemainingTimeout returns the remaining timeout in milliseconds.
// Returns 0 if no timeout is set.
func RemainingTimeout(ctx context.Context) int64 {
	if timeout, ok := ctx.Value(remainingTimeoutKey).(int64); ok {
		return timeout
	}
	return 0
}

// WithRemainingTimeout sets the remaining timeout in milliseconds.
func WithRemainingTimeout(ctx context.Context, timeoutMs int64) context.Context {
	return context.WithValue(ctx, remainingTimeoutKey, timeoutMs)
}
