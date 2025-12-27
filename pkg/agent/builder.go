package agent

import (
	"github.com/google/uuid"
	"github.com/storo/lettice/pkg/core"
	"github.com/storo/lettice/pkg/provider"
	"github.com/storo/lettice/pkg/storage"
)

// Default values
const (
	DefaultMaxTokens   = 4096
	DefaultTemperature = 0.7
)

// Builder is a fluent builder for creating agents.
type Builder struct {
	agent *Agent
}

// New creates a new agent builder with the given name.
func New(name string) *Builder {
	return &Builder{
		agent: &Agent{
			id:          uuid.New().String(),
			name:        name,
			maxTokens:   DefaultMaxTokens,
			temperature: DefaultTemperature,
			store:       storage.NewMemoryStore(),
		},
	}
}

// Description sets the agent's description.
func (b *Builder) Description(desc string) *Builder {
	b.agent.description = desc
	return b
}

// System sets the agent's system prompt.
func (b *Builder) System(prompt string) *Builder {
	b.agent.system = prompt
	return b
}

// Model sets the LLM provider for the agent.
func (b *Builder) Model(p provider.Provider) *Builder {
	b.agent.provider = p
	return b
}

// Tools adds tools to the agent.
func (b *Builder) Tools(tools ...core.Tool) *Builder {
	b.agent.tools = append(b.agent.tools, tools...)
	return b
}

// Provides declares capabilities this agent provides.
func (b *Builder) Provides(caps ...core.Capability) *Builder {
	b.agent.provides = append(b.agent.provides, caps...)
	return b
}

// Needs declares capabilities this agent needs from others.
func (b *Builder) Needs(caps ...core.Capability) *Builder {
	b.agent.needs = append(b.agent.needs, caps...)
	return b
}

// Store sets the storage backend for the agent.
func (b *Builder) Store(s storage.Store) *Builder {
	b.agent.store = s
	return b
}

// MaxTokens sets the maximum tokens for responses.
func (b *Builder) MaxTokens(n int) *Builder {
	b.agent.maxTokens = n
	return b
}

// Temperature sets the temperature for responses.
func (b *Builder) Temperature(t float64) *Builder {
	b.agent.temperature = t
	return b
}

// Build creates the agent and generates its card.
func (b *Builder) Build() *Agent {
	b.generateCard()
	return b.agent
}

// generateCard creates the agent card for discovery.
func (b *Builder) generateCard() {
	toolNames := make([]string, len(b.agent.tools))
	for i, t := range b.agent.tools {
		toolNames[i] = t.Name()
	}

	b.agent.card = &core.AgentCard{
		Name:        b.agent.name,
		Description: b.agent.description,
		Version:     "1.0.0",
		Capabilities: core.CardCapabilities{
			Provides:  b.agent.provides,
			Needs:     b.agent.needs,
			Streaming: true,
		},
		Tools: toolNames,
	}
}
