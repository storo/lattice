# Lattice

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Agent Mesh Framework for building distributed AI systems in Go.

Lattice enables creating networks of AI agents that can discover, communicate with, and delegate tasks to each other based on their capabilities.

## Features

- **Agent Mesh**: Network of AI agents with automatic capability-based routing
- **Cycle Detection**: Prevents infinite loops in agent delegation chains
- **Load Balancing**: Multiple strategies (RoundRobin, Random, First)
- **Security**: API Key and JWT authentication with role-based access
- **HTTP Server**: REST API to expose your mesh
- **Streaming**: Real-time output from agents
- **Patterns**: ReAct, Supervisor, Sequential, Parallel execution

## Installation

```bash
go get github.com/storo/lettice
```

## Quick Start

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

    // Create a mock provider (or use anthropic.NewClient for real API)
    llm := provider.NewMockWithResponse("Hello! I'm a research expert.")

    // Create an agent
    researcher := lattice.NewAgent("researcher").
        Model(llm).
        System("You are a research expert.").
        Provides(lattice.CapResearch).
        Build()

    // Create mesh and register agent
    mesh := lattice.NewMesh(lattice.WithMaxHops(5))
    mesh.Register(researcher)

    // Run a task
    result, err := mesh.Run(ctx, "Research AI trends")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Output)
}
```

## Documentation

- [Getting Started](docs/getting-started.md) - Installation and first steps
- [Agents](docs/agents.md) - Creating and configuring agents
- [Mesh](docs/mesh.md) - Mesh orchestration and delegation
- [Security](docs/security.md) - Authentication and authorization
- [HTTP API](docs/http-api.md) - REST API reference
- [Patterns](docs/patterns.md) - ReAct, Supervisor, and more
- [Middleware](docs/middleware.md) - Metrics, logging, tracing

## Example Server

Run a complete server with authentication:

```bash
# With mock provider
go run examples/server/main.go

# With Anthropic API
ANTHROPIC_API_KEY=sk-... go run examples/server/main.go
```

Test the endpoints:

```bash
# Health check (no auth)
curl http://localhost:8080/health

# List agents
curl -H "X-API-Key: demo-key" http://localhost:8080/agents

# Run on mesh
curl -X POST -H "X-API-Key: demo-key" \
  -H "Content-Type: application/json" \
  -d '{"input":"What is 2+2?"}' \
  http://localhost:8080/mesh/run
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         HTTP Server                          │
│                    (REST API + Auth)                         │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                          Mesh                                │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│   │  Researcher │  │   Writer    │  │    Coder    │        │
│   │ (Research)  │  │  (Writing)  │  │  (Coding)   │        │
│   └──────┬──────┘  └──────┬──────┘  └──────┬──────┘        │
│          │                │                │                │
│          └────────────────┼────────────────┘                │
│                           │                                  │
│                    Cycle Detection                           │
│                    Load Balancing                            │
│                    Tool Injection                            │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                       Providers                              │
│              (Anthropic, Mock, Custom)                       │
└─────────────────────────────────────────────────────────────┘
```

## License

MIT License - see [LICENSE](LICENSE) for details.
