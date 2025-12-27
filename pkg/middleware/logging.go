package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/storo/lattice/pkg/core"
)

// LogLevel defines the logging verbosity.
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// LogFormat defines the output format.
type LogFormat string

const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

// LoggingAgent wraps an agent with logging.
type LoggingAgent struct {
	agent  core.Agent
	logger *log.Logger
	level  LogLevel
	format LogFormat
}

// LoggingOption configures the logging middleware.
type LoggingOption func(*LoggingAgent)

// WrapWithLogging wraps an agent with logging.
func WrapWithLogging(agent core.Agent, logger *log.Logger, opts ...LoggingOption) *LoggingAgent {
	la := &LoggingAgent{
		agent:  agent,
		logger: logger,
		level:  LogLevelInfo,
		format: LogFormatText,
	}

	for _, opt := range opts {
		opt(la)
	}

	return la
}

// WithLogLevel sets the log level.
func WithLogLevel(level LogLevel) LoggingOption {
	return func(la *LoggingAgent) {
		la.level = level
	}
}

// WithLogFormat sets the log format.
func WithLogFormat(format LogFormat) LoggingOption {
	return func(la *LoggingAgent) {
		la.format = format
	}
}

// ID returns the agent's ID.
func (la *LoggingAgent) ID() string {
	return la.agent.ID()
}

// Name returns the agent's name.
func (la *LoggingAgent) Name() string {
	return la.agent.Name()
}

// Description returns the agent's description.
func (la *LoggingAgent) Description() string {
	return la.agent.Description()
}

// Provides returns capabilities provided.
func (la *LoggingAgent) Provides() []core.Capability {
	return la.agent.Provides()
}

// Needs returns capabilities needed.
func (la *LoggingAgent) Needs() []core.Capability {
	return la.agent.Needs()
}

// Tools returns available tools.
func (la *LoggingAgent) Tools() []core.Tool {
	return la.agent.Tools()
}

// Card returns the agent's card.
func (la *LoggingAgent) Card() *core.AgentCard {
	return la.agent.Card()
}

// Stop stops the agent.
func (la *LoggingAgent) Stop() error {
	return la.agent.Stop()
}

// Run executes the agent with logging.
func (la *LoggingAgent) Run(ctx context.Context, input string) (*core.Result, error) {
	start := time.Now()
	traceID := core.TraceID(ctx)

	la.log(LogLevelInfo, "starting", map[string]any{
		"agent":    la.agent.Name(),
		"trace_id": traceID,
		"input":    truncate(input, 100),
	})

	result, err := la.agent.Run(ctx, input)

	duration := time.Since(start)

	if err != nil {
		la.log(LogLevelError, "error", map[string]any{
			"agent":    la.agent.Name(),
			"trace_id": traceID,
			"error":    err.Error(),
			"duration": duration.String(),
		})
		return result, err
	}

	la.log(LogLevelInfo, "completed", map[string]any{
		"agent":      la.agent.Name(),
		"trace_id":   traceID,
		"output":     truncate(result.Output, 100),
		"tokens_in":  result.TokensIn,
		"tokens_out": result.TokensOut,
		"duration":   duration.String(),
	})

	return result, nil
}

// RunStream executes the agent with streaming and logging.
func (la *LoggingAgent) RunStream(ctx context.Context, input string) (<-chan core.StreamChunk, error) {
	la.log(LogLevelInfo, "starting stream", map[string]any{
		"agent": la.agent.Name(),
		"input": truncate(input, 100),
	})

	return la.agent.RunStream(ctx, input)
}

func (la *LoggingAgent) log(level LogLevel, message string, fields map[string]any) {
	if la.format == LogFormatJSON {
		fields["level"] = string(level)
		fields["message"] = message
		fields["timestamp"] = time.Now().Format(time.RFC3339)
		data, _ := json.Marshal(fields)
		la.logger.Println(string(data))
	} else {
		la.logger.Printf("[%s] %s: %v", level, message, fields)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// Verify LoggingAgent implements core.Agent
var _ core.Agent = (*LoggingAgent)(nil)

// LoggingMiddleware creates a middleware function for logging.
func LoggingMiddleware(logger *log.Logger, opts ...LoggingOption) func(core.Agent) core.Agent {
	return func(agent core.Agent) core.Agent {
		return WrapWithLogging(agent, logger, opts...)
	}
}

// AgentLogger provides structured logging for agents.
type AgentLogger struct {
	logger *log.Logger
	format LogFormat
}

// NewAgentLogger creates a new agent logger.
func NewAgentLogger(logger *log.Logger, format LogFormat) *AgentLogger {
	return &AgentLogger{
		logger: logger,
		format: format,
	}
}

// LogToolCall logs a tool call.
func (al *AgentLogger) LogToolCall(agentID string, toolName string, params map[string]any) {
	al.log(LogLevelDebug, "tool_call", map[string]any{
		"agent": agentID,
		"tool":  toolName,
		"params": fmt.Sprintf("%v", params),
	})
}

// LogToolResult logs a tool result.
func (al *AgentLogger) LogToolResult(agentID string, toolName string, result string, err error) {
	fields := map[string]any{
		"agent":  agentID,
		"tool":   toolName,
		"result": truncate(result, 200),
	}
	if err != nil {
		fields["error"] = err.Error()
		al.log(LogLevelError, "tool_error", fields)
	} else {
		al.log(LogLevelDebug, "tool_result", fields)
	}
}

func (al *AgentLogger) log(level LogLevel, message string, fields map[string]any) {
	if al.format == LogFormatJSON {
		fields["level"] = string(level)
		fields["message"] = message
		fields["timestamp"] = time.Now().Format(time.RFC3339)
		data, _ := json.Marshal(fields)
		al.logger.Println(string(data))
	} else {
		al.logger.Printf("[%s] %s: %v", level, message, fields)
	}
}
