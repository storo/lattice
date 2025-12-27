package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"strings"
	"testing"

	"github.com/storo/lattice/pkg/agent"
	"github.com/storo/lattice/pkg/provider"
)

func TestLoggingMiddleware(t *testing.T) {
	ctx := context.Background()

	// Capture log output
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	mockProvider := provider.NewMockWithResponse("Hello!")

	a := agent.New("test-agent").
		Model(mockProvider).
		Build()

	wrapped := WrapWithLogging(a, logger)

	_, err := wrapped.Run(ctx, "Say hello")
	if err != nil {
		t.Fatalf("failed to run: %v", err)
	}

	logOutput := buf.String()

	if !strings.Contains(logOutput, "test-agent") {
		t.Error("expected log to contain agent name")
	}

	if !strings.Contains(logOutput, "starting") {
		t.Error("expected log to contain 'starting'")
	}

	if !strings.Contains(logOutput, "completed") {
		t.Error("expected log to contain 'completed'")
	}
}

func TestLoggingMiddleware_Error(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	// Agent without provider will fail
	a := agent.New("failing-agent").Build()

	wrapped := WrapWithLogging(a, logger)

	_, err := wrapped.Run(ctx, "Test")
	if err == nil {
		t.Error("expected error")
	}

	logOutput := buf.String()

	if !strings.Contains(logOutput, "error") {
		t.Error("expected log to contain 'error'")
	}
}

func TestLoggingMiddleware_JSONFormat(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	mockProvider := provider.NewMockWithResponse("Result")

	a := agent.New("test-agent").
		Model(mockProvider).
		Build()

	wrapped := WrapWithLogging(a, logger, WithLogFormat(LogFormatJSON))

	_, err := wrapped.Run(ctx, "Test")
	if err != nil {
		t.Fatalf("failed to run: %v", err)
	}

	// Each line should be valid JSON
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("line is not valid JSON: %s", line)
		}
	}
}

func TestLoggingMiddleware_Levels(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	mockProvider := provider.NewMockWithResponse("Result")

	a := agent.New("test-agent").
		Model(mockProvider).
		Build()

	wrapped := WrapWithLogging(a, logger, WithLogLevel(LogLevelDebug))

	_, err := wrapped.Run(ctx, "Test")
	if err != nil {
		t.Fatalf("failed to run: %v", err)
	}

	logOutput := buf.String()

	// Debug level should include more details
	if !strings.Contains(logOutput, "DEBUG") && !strings.Contains(logOutput, "INFO") {
		t.Error("expected level in log output")
	}
}
