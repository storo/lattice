package registry

import (
	"context"
	"testing"

	"github.com/storo/lattice/pkg/agent"
	"github.com/storo/lattice/pkg/core"
)

func TestLocalRegistry_Register(t *testing.T) {
	ctx := context.Background()
	reg := NewLocal()

	a := agent.New("test-agent").
		Provides(core.CapResearch).
		Build()

	if err := reg.Register(ctx, a); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Should be able to find it
	found, err := reg.Get(ctx, a.ID())
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if found.ID() != a.ID() {
		t.Errorf("expected ID %s, got %s", a.ID(), found.ID())
	}
}

func TestLocalRegistry_Deregister(t *testing.T) {
	ctx := context.Background()
	reg := NewLocal()

	a := agent.New("test-agent").Build()

	reg.Register(ctx, a)
	reg.Deregister(ctx, a.ID())

	_, err := reg.Get(ctx, a.ID())
	if err != ErrAgentNotFound {
		t.Errorf("expected ErrAgentNotFound, got %v", err)
	}
}

func TestLocalRegistry_FindByCapability(t *testing.T) {
	ctx := context.Background()
	reg := NewLocal()

	researcher := agent.New("researcher").
		Provides(core.CapResearch).
		Build()

	writer := agent.New("writer").
		Provides(core.CapWriting).
		Build()

	coder := agent.New("coder").
		Provides(core.CapCoding, core.CapResearch).
		Build()

	reg.Register(ctx, researcher)
	reg.Register(ctx, writer)
	reg.Register(ctx, coder)

	// Find research providers
	researchProviders, err := reg.FindByCapability(ctx, core.CapResearch)
	if err != nil {
		t.Fatalf("failed to find: %v", err)
	}
	if len(researchProviders) != 2 {
		t.Errorf("expected 2 research providers, got %d", len(researchProviders))
	}

	// Find writing providers
	writingProviders, err := reg.FindByCapability(ctx, core.CapWriting)
	if err != nil {
		t.Fatalf("failed to find: %v", err)
	}
	if len(writingProviders) != 1 {
		t.Errorf("expected 1 writing provider, got %d", len(writingProviders))
	}

	// Find nonexistent capability
	noneProviders, err := reg.FindByCapability(ctx, core.CapAnalysis)
	if err != nil {
		t.Fatalf("failed to find: %v", err)
	}
	if len(noneProviders) != 0 {
		t.Errorf("expected 0 analysis providers, got %d", len(noneProviders))
	}
}

func TestLocalRegistry_List(t *testing.T) {
	ctx := context.Background()
	reg := NewLocal()

	a1 := agent.New("agent-1").Build()
	a2 := agent.New("agent-2").Build()

	reg.Register(ctx, a1)
	reg.Register(ctx, a2)

	agents, err := reg.List(ctx)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
}

func TestLocalRegistry_RegisterDuplicate(t *testing.T) {
	ctx := context.Background()
	reg := NewLocal()

	a := agent.New("test-agent").Build()

	reg.Register(ctx, a)

	// Registering same agent again should update
	err := reg.Register(ctx, a)
	if err != nil {
		t.Errorf("re-registering should not error: %v", err)
	}

	agents, _ := reg.List(ctx)
	if len(agents) != 1 {
		t.Errorf("expected 1 agent after re-registration, got %d", len(agents))
	}
}
