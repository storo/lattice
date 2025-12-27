package middleware

import (
	"context"

	"github.com/storo/lettice/pkg/core"
	"github.com/storo/lettice/pkg/protocol"
)

// TracingAgent wraps an agent with distributed tracing.
type TracingAgent struct {
	agent  core.Agent
	tracer *protocol.Tracer
}

// WrapWithTracing wraps an agent with tracing.
func WrapWithTracing(agent core.Agent, tracer *protocol.Tracer) *TracingAgent {
	return &TracingAgent{
		agent:  agent,
		tracer: tracer,
	}
}

// ID returns the agent's ID.
func (ta *TracingAgent) ID() string {
	return ta.agent.ID()
}

// Name returns the agent's name.
func (ta *TracingAgent) Name() string {
	return ta.agent.Name()
}

// Description returns the agent's description.
func (ta *TracingAgent) Description() string {
	return ta.agent.Description()
}

// Provides returns capabilities provided.
func (ta *TracingAgent) Provides() []core.Capability {
	return ta.agent.Provides()
}

// Needs returns capabilities needed.
func (ta *TracingAgent) Needs() []core.Capability {
	return ta.agent.Needs()
}

// Tools returns available tools.
func (ta *TracingAgent) Tools() []core.Tool {
	return ta.agent.Tools()
}

// Card returns the agent's card.
func (ta *TracingAgent) Card() *core.AgentCard {
	return ta.agent.Card()
}

// Stop stops the agent.
func (ta *TracingAgent) Stop() error {
	return ta.agent.Stop()
}

// Run executes the agent with tracing.
func (ta *TracingAgent) Run(ctx context.Context, input string) (*core.Result, error) {
	ctx, span := ta.tracer.StartSpan(ctx, "agent.run")
	defer span.End()

	span.SetAttribute("agent.id", ta.agent.ID())
	span.SetAttribute("agent.name", ta.agent.Name())
	span.SetAttribute("input.length", len(input))

	result, err := ta.agent.Run(ctx, input)

	if err != nil {
		span.SetStatus(protocol.StatusError, err.Error())
		span.SetAttribute("error", err.Error())
		return result, err
	}

	span.SetStatus(protocol.StatusOK, "")
	span.SetAttribute("output.length", len(result.Output))
	span.SetAttribute("tokens.in", result.TokensIn)
	span.SetAttribute("tokens.out", result.TokensOut)
	span.SetAttribute("duration.ms", result.Duration.Milliseconds())

	return result, nil
}

// RunStream executes the agent with streaming and tracing.
func (ta *TracingAgent) RunStream(ctx context.Context, input string) (<-chan core.StreamChunk, error) {
	ctx, span := ta.tracer.StartSpan(ctx, "agent.run_stream")

	span.SetAttribute("agent.id", ta.agent.ID())
	span.SetAttribute("agent.name", ta.agent.Name())
	span.SetAttribute("input.length", len(input))

	chunks, err := ta.agent.RunStream(ctx, input)
	if err != nil {
		span.SetStatus(protocol.StatusError, err.Error())
		span.End()
		return nil, err
	}

	// Wrap the channel to track completion
	tracedCh := make(chan core.StreamChunk)
	go func() {
		defer close(tracedCh)
		defer span.End()

		for chunk := range chunks {
			tracedCh <- chunk
			if chunk.Error != nil {
				span.SetStatus(protocol.StatusError, chunk.Error.Error())
			}
		}
		span.SetStatus(protocol.StatusOK, "")
	}()

	return tracedCh, nil
}

// Verify TracingAgent implements core.Agent
var _ core.Agent = (*TracingAgent)(nil)

// TracingMiddleware creates a middleware function for tracing.
func TracingMiddleware(tracer *protocol.Tracer) func(core.Agent) core.Agent {
	return func(agent core.Agent) core.Agent {
		return WrapWithTracing(agent, tracer)
	}
}
