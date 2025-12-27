# Agents

Agents are the core building blocks of Lattice. Each agent is an AI-powered entity that can execute tasks, use tools, and delegate work to other agents.

## Creating an Agent

Use the fluent builder pattern to create agents:

```go
agent := lattice.NewAgent("researcher").
    Model(provider).
    System("You are a research expert.").
    Description("Researches topics and provides summaries").
    Provides(lattice.CapResearch).
    Build()
```

## Builder Methods

### Required

| Method | Description |
|--------|-------------|
| `Model(provider)` | Sets the LLM provider (required for execution) |

### Optional

| Method | Default | Description |
|--------|---------|-------------|
| `System(prompt)` | `""` | System prompt defining agent behavior |
| `Description(desc)` | `""` | Human-readable description |
| `Provides(caps...)` | `[]` | Capabilities this agent provides |
| `Needs(caps...)` | `[]` | Capabilities this agent requires |
| `Tools(tools...)` | `[]` | Tools the agent can use |
| `Store(store)` | `MemoryStore` | State storage backend |
| `MaxTokens(n)` | `4096` | Maximum response tokens |
| `Temperature(t)` | `0.7` | Response randomness (0.0-1.0) |

## Capabilities

Capabilities define what an agent can do and what it needs from others.

### Built-in Capabilities

```go
lattice.CapResearch  // Research and information gathering
lattice.CapWriting   // Content creation and writing
lattice.CapCoding    // Programming and code generation
lattice.CapAnalysis  // Data analysis and insights
lattice.CapPlanning  // Planning and strategy
```

### Custom Capabilities

```go
agent := lattice.NewAgent("translator").
    Provides(lattice.Cap("translation")).
    Provides(lattice.Cap("localization")).
    Build()
```

### Capability Matching

When an agent `Needs` a capability, the mesh automatically finds agents that `Provide` it:

```go
// Agent A needs research capability
agentA := lattice.NewAgent("writer").
    Needs(lattice.CapResearch).  // Will delegate to researchers
    Provides(lattice.CapWriting).
    Build()

// Agent B provides research capability
agentB := lattice.NewAgent("researcher").
    Provides(lattice.CapResearch).  // Can fulfill A's needs
    Build()
```

## System Prompts

The system prompt defines the agent's personality and behavior:

```go
agent := lattice.NewAgent("coder").
    System(`You are an expert Go programmer.

Rules:
- Write clean, idiomatic Go code
- Include error handling
- Add comments for complex logic
- Follow Go naming conventions`).
    Build()
```

## Tools

Agents can use tools to extend their capabilities:

```go
// Define a tool
type CalculatorTool struct{}

func (t *CalculatorTool) Name() string { return "calculator" }
func (t *CalculatorTool) Description() string { return "Performs calculations" }
func (t *CalculatorTool) Schema() json.RawMessage {
    return json.RawMessage(`{
        "type": "object",
        "properties": {
            "expression": {"type": "string"}
        }
    }`)
}
func (t *CalculatorTool) Execute(ctx context.Context, params json.RawMessage) (string, error) {
    // Implementation
    return "42", nil
}

// Add tool to agent
agent := lattice.NewAgent("math-assistant").
    Tools(&CalculatorTool{}).
    Build()
```

## Execution

### Synchronous Execution

```go
result, err := agent.Run(ctx, "Calculate 2 + 2")
if err != nil {
    log.Fatal(err)
}

fmt.Println(result.Output)
fmt.Printf("Tokens: %d in, %d out\n", result.TokensIn, result.TokensOut)
fmt.Printf("Duration: %s\n", result.Duration)
```

### Streaming Execution

```go
stream, err := agent.RunStream(ctx, "Write a long article")
if err != nil {
    log.Fatal(err)
}

for chunk := range stream {
    if chunk.Error != nil {
        log.Fatal(chunk.Error)
    }
    fmt.Print(chunk.Content)
    if chunk.Done {
        break
    }
}
```

## Agent Card

Each agent has a discovery card for mesh registration:

```go
card := agent.Card()

fmt.Println(card.Name)        // "researcher"
fmt.Println(card.Description) // "Researches topics..."
fmt.Println(card.Version)     // "1.0.0"
fmt.Println(card.Capabilities.Provides) // [CapResearch]
fmt.Println(card.Capabilities.Streaming) // true
```

## Configuration Patterns

### High-Quality Responses

```go
agent := lattice.NewAgent("analyst").
    Temperature(0.2).  // More focused, less random
    MaxTokens(8192).   // Longer responses
    Build()
```

### Creative Responses

```go
agent := lattice.NewAgent("creative-writer").
    Temperature(0.9).  // More creative, varied
    Build()
```

### With Custom Storage

```go
import "github.com/storo/lettice/pkg/storage"

// Use Redis for distributed state
redis, _ := storage.NewRedisStore("localhost:6379")

agent := lattice.NewAgent("stateful-agent").
    Store(redis).
    Build()
```

## Next Steps

- [Mesh](mesh.md) - Connect agents in a mesh
- [Patterns](patterns.md) - Use advanced patterns
- [Middleware](middleware.md) - Add observability
