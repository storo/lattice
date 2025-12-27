# Mesh

The Mesh is the central orchestrator that connects agents, handles delegation, and prevents infinite loops.

## Creating a Mesh

```go
mesh := lattice.NewMesh()
```

## Configuration Options

### MaxHops

Limits how many times agents can delegate to each other:

```go
mesh := lattice.NewMesh(lattice.WithMaxHops(5))
```

Default: 10 hops

### Load Balancer

When multiple agents provide the same capability, the balancer selects one:

```go
// Round-robin (default) - distributes evenly
mesh := lattice.NewMesh(
    lattice.WithBalancer(lattice.NewRoundRobinBalancer()),
)

// Random - random selection
mesh := lattice.NewMesh(
    lattice.WithBalancer(lattice.NewRandomBalancer(rand.Intn)),
)

// First - always picks the first
mesh := lattice.NewMesh(
    lattice.WithBalancer(lattice.NewFirstBalancer()),
)
```

### Custom Registry

Use a custom agent registry:

```go
mesh := lattice.NewMesh(
    lattice.WithRegistry(myCustomRegistry),
)
```

## Registering Agents

```go
mesh := lattice.NewMesh()

// Register single agent
mesh.Register(researcher)

// Register multiple agents
mesh.Register(researcher, writer, coder)
```

## Running Tasks

### Run on Mesh (Auto-select)

The mesh picks the first available agent:

```go
result, err := mesh.Run(ctx, "Research AI trends")
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Output)
```

### Run Specific Agent

Execute a specific agent by ID:

```go
result, err := mesh.RunAgent(ctx, "researcher-123", "Research AI trends")
if err != nil {
    log.Fatal(err)
}
```

## Agent Discovery

### List All Agents

```go
agents, err := mesh.ListAgents(ctx)
for _, a := range agents {
    fmt.Printf("%s: %s\n", a.ID(), a.Name())
}
```

### Get Agent by ID

```go
agent, err := mesh.GetAgent(ctx, "researcher-123")
if err == lattice.ErrAgentNotFound {
    log.Fatal("Agent not found")
}
```

### Find by Capability

```go
researchers, err := mesh.FindProviders(ctx, lattice.CapResearch)
for _, r := range researchers {
    fmt.Printf("Researcher: %s\n", r.Name())
}
```

## Cycle Detection

The mesh prevents infinite delegation loops between agents.

### How It Works

1. Each delegation adds the agent to a call chain
2. If an agent appears twice → `ErrCycleDetected`
3. If hops exceed max → `ErrMaxHopsExceeded`

### Example

```go
// Agent A needs writing, Agent B needs research
agentA := lattice.NewAgent("researcher").
    Provides(lattice.CapResearch).
    Needs(lattice.CapWriting).
    Build()

agentB := lattice.NewAgent("writer").
    Provides(lattice.CapWriting).
    Needs(lattice.CapResearch).
    Build()

mesh := lattice.NewMesh(lattice.WithMaxHops(3))
mesh.Register(agentA, agentB)

// This would cause A → B → A cycle
result, err := mesh.Run(ctx, "Research and write about AI")
if err == lattice.ErrCycleDetected {
    log.Println("Cycle detected, breaking loop")
}
```

## Tool Injection

The mesh automatically injects delegation tools into agents:

```go
// Agent A needs research capability
agentA := lattice.NewAgent("writer").
    Needs(lattice.CapResearch).
    Build()

// Agent B provides research capability
agentB := lattice.NewAgent("researcher").
    Provides(lattice.CapResearch).
    Build()

mesh := lattice.NewMesh()
mesh.Register(agentA, agentB)

// When agentA runs, it automatically gets a tool to call agentB
// The LLM can use this tool to delegate research tasks
```

The injected tool:
- Name: `delegate_research` (based on capability)
- Description: Auto-generated from agent cards
- Execution: Routes through mesh with cycle detection

## Tracing

Each mesh execution gets a trace ID for debugging:

```go
// Auto-generated trace ID
result, _ := mesh.Run(ctx, "task")
fmt.Println(result.TraceID) // "abc123..."

// Custom trace ID
ctx = lattice.WithTraceID(ctx, "my-trace-id")
result, _ = mesh.Run(ctx, "task")
fmt.Println(result.TraceID) // "my-trace-id"
```

## Result Structure

```go
result, _ := mesh.Run(ctx, "task")

fmt.Println(result.Output)     // Agent's response
fmt.Println(result.TokensIn)   // Input tokens used
fmt.Println(result.TokensOut)  // Output tokens generated
fmt.Println(result.Duration)   // Execution time
fmt.Println(result.TraceID)    // Trace identifier
fmt.Println(result.CallChain)  // ["agent-a", "agent-b"]
```

## Error Handling

```go
result, err := mesh.Run(ctx, "task")

switch {
case err == nil:
    // Success
case errors.Is(err, lattice.ErrAgentNotFound):
    log.Println("No agents registered")
case errors.Is(err, lattice.ErrCycleDetected):
    log.Println("Delegation cycle detected")
case errors.Is(err, lattice.ErrMaxHopsExceeded):
    log.Println("Too many delegation hops")
default:
    log.Printf("Execution error: %v", err)
}
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/storo/lettice"
    "github.com/storo/lettice/pkg/provider"
)

func main() {
    ctx := context.Background()
    llm := provider.NewMockWithResponse("Done!")

    // Create specialized agents
    researcher := lattice.NewAgent("researcher").
        Model(llm).
        System("You research topics.").
        Provides(lattice.CapResearch).
        Build()

    writer := lattice.NewAgent("writer").
        Model(llm).
        System("You write content.").
        Provides(lattice.CapWriting).
        Needs(lattice.CapResearch). // Can delegate to researcher
        Build()

    coder := lattice.NewAgent("coder").
        Model(llm).
        System("You write code.").
        Provides(lattice.CapCoding).
        Build()

    // Create mesh with configuration
    mesh := lattice.NewMesh(
        lattice.WithMaxHops(5),
        lattice.WithBalancer(lattice.NewRoundRobinBalancer()),
    )

    // Register all agents
    if err := mesh.Register(researcher, writer, coder); err != nil {
        log.Fatal(err)
    }

    // Run a task
    result, err := mesh.Run(ctx, "Write an article about Go")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Output: %s\n", result.Output)
    fmt.Printf("Duration: %s\n", result.Duration)
    fmt.Printf("Call chain: %v\n", result.CallChain)
}
```

## Next Steps

- [Security](security.md) - Add authentication
- [HTTP API](http-api.md) - Expose mesh via REST
- [Patterns](patterns.md) - Advanced orchestration patterns
