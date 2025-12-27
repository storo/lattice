# Getting Started

This guide will help you install Lattice and create your first agent mesh.

## Installation

```bash
go get github.com/storo/lettice
```

## Prerequisites

- Go 1.22 or later
- (Optional) Anthropic API key for real LLM responses

## Your First Agent

Create a simple agent that responds to queries:

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

    // Use mock provider for development
    llm := provider.NewMockWithResponse("I'm a helpful assistant!")

    // Create agent with builder pattern
    assistant := lattice.NewAgent("assistant").
        Model(llm).
        System("You are a helpful assistant.").
        Build()

    // Run directly
    result, err := assistant.Run(ctx, "Hello!")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Output)
    // Output: I'm a helpful assistant!
}
```

## Using the Mesh

The mesh enables agents to discover and delegate to each other:

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
    llm := provider.NewMockWithResponse("Task completed!")

    // Create agents with capabilities
    researcher := lattice.NewAgent("researcher").
        Model(llm).
        System("You are a research expert.").
        Provides(lattice.CapResearch).
        Build()

    writer := lattice.NewAgent("writer").
        Model(llm).
        System("You are a skilled writer.").
        Provides(lattice.CapWriting).
        Build()

    // Create mesh with max 5 delegation hops
    mesh := lattice.NewMesh(lattice.WithMaxHops(5))

    // Register agents
    if err := mesh.Register(researcher, writer); err != nil {
        log.Fatal(err)
    }

    // Run on mesh (auto-selects agent)
    result, err := mesh.Run(ctx, "Write an article about AI")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Output)
}
```

## Using Real Anthropic API

To use the real Claude API:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/storo/lettice"
    "github.com/storo/lettice/pkg/provider/anthropic"
)

func main() {
    ctx := context.Background()

    // Create Anthropic client
    apiKey := os.Getenv("ANTHROPIC_API_KEY")
    claude := anthropic.NewClient(apiKey)

    // Create agent with real provider
    assistant := lattice.NewAgent("assistant").
        Model(claude).
        System("You are a helpful assistant.").
        Build()

    result, err := assistant.Run(ctx, "What is the capital of France?")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Output)
}
```

Run with:

```bash
ANTHROPIC_API_KEY=sk-... go run main.go
```

## Running the Example Server

The examples/server directory contains a complete server with authentication:

```bash
# Start server with mock provider
go run examples/server/main.go

# Or with real Anthropic API
ANTHROPIC_API_KEY=sk-... go run examples/server/main.go
```

Test it:

```bash
# Health check
curl http://localhost:8080/health

# List agents (requires auth)
curl -H "X-API-Key: demo-key" http://localhost:8080/agents

# Run a task
curl -X POST -H "X-API-Key: demo-key" \
  -H "Content-Type: application/json" \
  -d '{"input":"Hello!"}' \
  http://localhost:8080/mesh/run
```

## Next Steps

- [Agents](agents.md) - Deep dive into agent configuration
- [Mesh](mesh.md) - Learn about mesh orchestration
- [Security](security.md) - Add authentication to your server
- [HTTP API](http-api.md) - Full API reference
