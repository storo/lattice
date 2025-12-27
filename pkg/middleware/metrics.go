package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/storo/lettice/pkg/core"
)

// AgentMetrics holds metrics for a single agent.
type AgentMetrics struct {
	AgentID      string
	AgentName    string
	TotalCalls   int64
	SuccessCount int64
	ErrorCount   int64
	TotalTokensIn  int64
	TotalTokensOut int64
	TotalDuration  time.Duration
	AvgDuration    time.Duration
	MaxDuration    time.Duration
	MinDuration    time.Duration
	LastCallTime   time.Time
}

// MetricsSummary provides aggregated metrics across all agents.
type MetricsSummary struct {
	AgentCount  int
	TotalCalls  int64
	SuccessRate float64
	AvgDuration time.Duration
}

// MetricsCollector collects and aggregates agent metrics.
type MetricsCollector struct {
	mu      sync.RWMutex
	metrics map[string]*AgentMetrics
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*AgentMetrics),
	}
}

// Record records a single execution.
func (mc *MetricsCollector) Record(agentID, agentName string, duration time.Duration, tokensIn, tokensOut int, err error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	m, ok := mc.metrics[agentID]
	if !ok {
		m = &AgentMetrics{
			AgentID:     agentID,
			AgentName:   agentName,
			MinDuration: duration,
		}
		mc.metrics[agentID] = m
	}

	m.TotalCalls++
	m.TotalDuration += duration
	m.TotalTokensIn += int64(tokensIn)
	m.TotalTokensOut += int64(tokensOut)
	m.LastCallTime = time.Now()

	if err != nil {
		m.ErrorCount++
	} else {
		m.SuccessCount++
	}

	if duration > m.MaxDuration {
		m.MaxDuration = duration
	}
	if duration < m.MinDuration {
		m.MinDuration = duration
	}

	m.AvgDuration = m.TotalDuration / time.Duration(m.TotalCalls)
}

// GetAgentMetrics returns metrics for a specific agent.
func (mc *MetricsCollector) GetAgentMetrics(agentID string) AgentMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if m, ok := mc.metrics[agentID]; ok {
		return *m
	}
	return AgentMetrics{AgentID: agentID}
}

// AllMetrics returns metrics for all agents.
func (mc *MetricsCollector) AllMetrics() []AgentMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make([]AgentMetrics, 0, len(mc.metrics))
	for _, m := range mc.metrics {
		result = append(result, *m)
	}
	return result
}

// Summary returns aggregated metrics.
func (mc *MetricsCollector) Summary() MetricsSummary {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	summary := MetricsSummary{
		AgentCount: len(mc.metrics),
	}

	var totalDuration time.Duration
	var totalSuccess int64

	for _, m := range mc.metrics {
		summary.TotalCalls += m.TotalCalls
		totalDuration += m.TotalDuration
		totalSuccess += m.SuccessCount
	}

	if summary.TotalCalls > 0 {
		summary.AvgDuration = totalDuration / time.Duration(summary.TotalCalls)
		summary.SuccessRate = float64(totalSuccess) / float64(summary.TotalCalls)
	}

	return summary
}

// Reset clears all metrics.
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics = make(map[string]*AgentMetrics)
}

// MetricsAgent wraps an agent with metrics collection.
type MetricsAgent struct {
	agent     core.Agent
	collector *MetricsCollector
}

// WrapWithMetrics wraps an agent with metrics collection.
func WrapWithMetrics(agent core.Agent, collector *MetricsCollector) *MetricsAgent {
	return &MetricsAgent{
		agent:     agent,
		collector: collector,
	}
}

// ID returns the agent's ID.
func (ma *MetricsAgent) ID() string {
	return ma.agent.ID()
}

// Name returns the agent's name.
func (ma *MetricsAgent) Name() string {
	return ma.agent.Name()
}

// Description returns the agent's description.
func (ma *MetricsAgent) Description() string {
	return ma.agent.Description()
}

// Provides returns capabilities provided.
func (ma *MetricsAgent) Provides() []core.Capability {
	return ma.agent.Provides()
}

// Needs returns capabilities needed.
func (ma *MetricsAgent) Needs() []core.Capability {
	return ma.agent.Needs()
}

// Tools returns available tools.
func (ma *MetricsAgent) Tools() []core.Tool {
	return ma.agent.Tools()
}

// Card returns the agent's card.
func (ma *MetricsAgent) Card() *core.AgentCard {
	return ma.agent.Card()
}

// Stop stops the agent.
func (ma *MetricsAgent) Stop() error {
	return ma.agent.Stop()
}

// Run executes the agent with metrics collection.
func (ma *MetricsAgent) Run(ctx context.Context, input string) (*core.Result, error) {
	start := time.Now()

	result, err := ma.agent.Run(ctx, input)

	duration := time.Since(start)

	tokensIn, tokensOut := 0, 0
	if result != nil {
		tokensIn = result.TokensIn
		tokensOut = result.TokensOut
	}

	ma.collector.Record(
		ma.agent.ID(),
		ma.agent.Name(),
		duration,
		tokensIn,
		tokensOut,
		err,
	)

	return result, err
}

// RunStream executes the agent with streaming.
func (ma *MetricsAgent) RunStream(ctx context.Context, input string) (<-chan core.StreamChunk, error) {
	start := time.Now()

	chunks, err := ma.agent.RunStream(ctx, input)
	if err != nil {
		ma.collector.Record(ma.agent.ID(), ma.agent.Name(), time.Since(start), 0, 0, err)
		return nil, err
	}

	// Wrap to track completion
	metricsCh := make(chan core.StreamChunk)
	go func() {
		defer close(metricsCh)
		var finalErr error
		for chunk := range chunks {
			metricsCh <- chunk
			if chunk.Error != nil {
				finalErr = chunk.Error
			}
		}
		ma.collector.Record(ma.agent.ID(), ma.agent.Name(), time.Since(start), 0, 0, finalErr)
	}()

	return metricsCh, nil
}

// Verify MetricsAgent implements core.Agent
var _ core.Agent = (*MetricsAgent)(nil)

// MetricsMiddleware creates a middleware function for metrics.
func MetricsMiddleware(collector *MetricsCollector) func(core.Agent) core.Agent {
	return func(agent core.Agent) core.Agent {
		return WrapWithMetrics(agent, collector)
	}
}
