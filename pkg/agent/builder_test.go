package agent

import (
	"testing"

	"github.com/storo/lattice/pkg/core"
	"github.com/storo/lattice/pkg/provider"
	"github.com/storo/lattice/pkg/storage"
)

func TestBuilder_Basic(t *testing.T) {
	agent := New("test-agent").Build()

	if agent.Name() != "test-agent" {
		t.Errorf("expected name 'test-agent', got '%s'", agent.Name())
	}
	if agent.ID() == "" {
		t.Error("expected ID to be set")
	}
}

func TestBuilder_WithDescription(t *testing.T) {
	agent := New("test-agent").
		Description("A test agent").
		Build()

	if agent.Description() != "A test agent" {
		t.Errorf("expected description 'A test agent', got '%s'", agent.Description())
	}
}

func TestBuilder_WithSystem(t *testing.T) {
	agent := New("test-agent").
		System("You are a helpful assistant").
		Build()

	if agent.system != "You are a helpful assistant" {
		t.Errorf("expected system prompt to be set")
	}
}

func TestBuilder_WithCapabilities(t *testing.T) {
	agent := New("test-agent").
		Provides(core.CapResearch, core.CapWriting).
		Needs(core.CapCoding).
		Build()

	provides := agent.Provides()
	if len(provides) != 2 {
		t.Errorf("expected 2 provides, got %d", len(provides))
	}

	needs := agent.Needs()
	if len(needs) != 1 {
		t.Errorf("expected 1 needs, got %d", len(needs))
	}
	if needs[0] != core.CapCoding {
		t.Errorf("expected needs to contain coding")
	}
}

func TestBuilder_WithProvider(t *testing.T) {
	mockProvider := &provider.MockProvider{}

	agent := New("test-agent").
		Model(mockProvider).
		Build()

	if agent.provider == nil {
		t.Error("expected provider to be set")
	}
}

func TestBuilder_WithStore(t *testing.T) {
	store := storage.NewMemoryStore()

	agent := New("test-agent").
		Store(store).
		Build()

	if agent.store == nil {
		t.Error("expected store to be set")
	}
}

func TestBuilder_GeneratesCard(t *testing.T) {
	agent := New("test-agent").
		Description("A test agent").
		Provides(core.CapResearch).
		Needs(core.CapWriting).
		Build()

	card := agent.Card()
	if card == nil {
		t.Fatal("expected card to be generated")
	}

	if card.Name != "test-agent" {
		t.Errorf("expected card name 'test-agent', got '%s'", card.Name)
	}
	if card.Description != "A test agent" {
		t.Errorf("expected card description 'A test agent', got '%s'", card.Description)
	}
	if len(card.Capabilities.Provides) != 1 {
		t.Errorf("expected 1 provides in card, got %d", len(card.Capabilities.Provides))
	}
	if len(card.Capabilities.Needs) != 1 {
		t.Errorf("expected 1 needs in card, got %d", len(card.Capabilities.Needs))
	}
}

func TestBuilder_WithMaxTokens(t *testing.T) {
	agent := New("test-agent").
		MaxTokens(2000).
		Build()

	if agent.maxTokens != 2000 {
		t.Errorf("expected max tokens 2000, got %d", agent.maxTokens)
	}
}

func TestBuilder_WithTemperature(t *testing.T) {
	agent := New("test-agent").
		Temperature(0.8).
		Build()

	if agent.temperature != 0.8 {
		t.Errorf("expected temperature 0.8, got %f", agent.temperature)
	}
}

func TestBuilder_DefaultValues(t *testing.T) {
	agent := New("test-agent").Build()

	// Should have default max tokens
	if agent.maxTokens != DefaultMaxTokens {
		t.Errorf("expected default max tokens %d, got %d", DefaultMaxTokens, agent.maxTokens)
	}

	// Should have default temperature
	if agent.temperature != DefaultTemperature {
		t.Errorf("expected default temperature %f, got %f", DefaultTemperature, agent.temperature)
	}

	// Should have default store
	if agent.store == nil {
		t.Error("expected default store to be set")
	}
}
