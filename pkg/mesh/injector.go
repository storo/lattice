package mesh

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/storo/lettice/pkg/core"
	"github.com/storo/lettice/pkg/registry"
)

// Injector creates delegation tools for agents.
type Injector struct {
	registry      registry.Registry
	balancer      Balancer
	cycleDetector *CycleDetector
}

// NewInjector creates a new injector.
func NewInjector(reg registry.Registry, balancer Balancer, cd *CycleDetector) *Injector {
	return &Injector{
		registry:      reg,
		balancer:      balancer,
		cycleDetector: cd,
	}
}

// InjectTools creates delegation tools for an agent based on its needs.
// For each capability the agent needs, it creates a tool that delegates
// to agents that provide that capability.
func (i *Injector) InjectTools(ctx context.Context, agent core.Agent) ([]core.Tool, error) {
	var tools []core.Tool

	for _, need := range agent.Needs() {
		// Find agents that provide this capability
		providers, err := i.registry.FindByCapability(ctx, need)
		if err != nil {
			return nil, fmt.Errorf("failed to find providers for %s: %w", need, err)
		}

		if len(providers) == 0 {
			// No providers for this capability, skip
			continue
		}

		// Create a delegation tool
		tool := &AgentTool{
			capability:    need,
			providers:     providers,
			balancer:      i.balancer,
			cycleDetector: i.cycleDetector,
		}
		tools = append(tools, tool)
	}

	return tools, nil
}

// AgentToolInput defines the schema for delegation tool input.
type AgentToolInput struct {
	Task    string `json:"task" schema:"The specific task or question to delegate to the specialized agent"`
	Context string `json:"context,omitempty" schema:"Additional context that might help the agent understand the task better"`
}

// AgentTool wraps an agent as a tool for delegation.
type AgentTool struct {
	capability    core.Capability
	providers     []core.Agent
	balancer      Balancer
	cycleDetector *CycleDetector
}

// Name returns the tool name.
func (t *AgentTool) Name() string {
	return fmt.Sprintf("delegate_to_%s", t.capability)
}

// Description returns the tool description.
func (t *AgentTool) Description() string {
	return fmt.Sprintf(
		"Delegate a task to a specialized agent that provides '%s' capability. "+
			"Use this when you need expert help with %s-related tasks. "+
			"The agent will process your request and return the result.",
		t.capability, t.capability,
	)
}

// Schema returns the JSON Schema for the tool's parameters.
func (t *AgentTool) Schema() json.RawMessage {
	return core.SchemaFromStruct(AgentToolInput{})
}

// Execute runs the delegation tool.
func (t *AgentTool) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	var input AgentToolInput
	if err := json.Unmarshal(params, &input); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	// Select a provider
	provider := t.balancer.Select(t.providers)
	if provider == nil {
		return "", fmt.Errorf("no provider available for %s", t.capability)
	}

	// Check for cycles BEFORE executing
	if err := t.cycleDetector.Check(ctx, provider.ID()); err != nil {
		return "", err
	}

	// Prepare context for the delegated agent
	ctx = t.cycleDetector.PrepareContext(ctx, provider.ID())

	// Build the full input
	fullInput := input.Task
	if input.Context != "" {
		fullInput = fmt.Sprintf("Context: %s\n\nTask: %s", input.Context, input.Task)
	}

	// Execute the delegated agent
	result, err := provider.Run(ctx, fullInput)
	if err != nil {
		return "", fmt.Errorf("agent %s failed: %w", provider.Name(), err)
	}

	return result.Output, nil
}

// Verify AgentTool implements core.Tool
var _ core.Tool = (*AgentTool)(nil)
