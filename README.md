# Lattice

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Agent Mesh Framework for building distributed AI systems in Go.

## Why Lattice?

- **Zero-Config Local AI** - Use [Ollama](https://ollama.com) for free, local LLMs
- **No Docker Required** - SQLite storage works out of the box
- **Batteries Included** - Built-in tools (fs, http, shell, time)
- **Production Ready** - HTTP server with auth, load balancing, cycle detection

## Getting Started

### Option 1: Local LLMs (Free)

```bash
# 1. Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# 2. Pull a model
ollama pull llama3.2

# 3. Install Lattice
go get github.com/storo/lattice
```

### Option 2: Anthropic API

```bash
export ANTHROPIC_API_KEY=sk-...
go get github.com/storo/lattice
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"

    "github.com/storo/lattice"
    "github.com/storo/lattice/pkg/provider/ollama"
    "github.com/storo/lattice/pkg/tool/builtin"
)

func main() {
    // Free local LLM - no API key needed!
    provider := ollama.NewClient()

    // Create agent with built-in tools
    agent := lattice.NewAgent("assistant").
        Model(provider).
        System("You are a helpful assistant.").
        Tools(builtin.DefaultTools()...).
        Build()

    // Run
    result, _ := agent.Run(context.Background(), "What time is it?")
    fmt.Println(result.Output)
}
```

## CLI Demo

![Lattice Interactive Demo](docs/demo.gif)

```bash
# Build CLI
go build -o lattice ./cmd/lattice

# Start server (uses mock provider by default)
./lattice serve &

# Interactive mode
./lattice interactive
```

## Features

- **Agent Mesh**: Network of AI agents with automatic capability-based routing
- **Multiple Providers**: Ollama (local), Anthropic, or custom
- **Storage Options**: SQLite, Redis, or in-memory
- **Built-in Tools**: Time, HTTP, File System, Shell (with security controls)
- **Cycle Detection**: Prevents infinite loops in agent delegation chains
- **Load Balancing**: Multiple strategies (RoundRobin, Random, First)
- **Security**: API Key and JWT authentication with role-based access
- **HTTP Server**: REST API to expose your mesh
- **Streaming**: Real-time output from agents
- **Patterns**: ReAct, Supervisor, Sequential, Parallel execution

## Configuration

```yaml
# lattice.yaml
provider:
  type: ollama          # or: anthropic, mock
  model: llama3.2

storage:
  type: sqlite          # or: redis, memory
  path: ./lattice.db

agents:
  - name: assistant
    system: You are helpful.
    provides: [general]
```

## Documentation

- [Getting Started](docs/getting-started.md) - Installation and first steps
- [Agents](docs/agents.md) - Creating and configuring agents
- [Mesh](docs/mesh.md) - Mesh orchestration and delegation
- [Security](docs/security.md) - Authentication and authorization
- [HTTP API](docs/http-api.md) - REST API reference
- [Patterns](docs/patterns.md) - ReAct, Supervisor, and more
- [Middleware](docs/middleware.md) - Metrics, logging, tracing

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
│              Cycle Detection + Load Balancing                │
│                    + Tool Injection                          │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                       Providers                              │
│         (Ollama, Anthropic, Mock) + Built-in Tools           │
├─────────────────────────────────────────────────────────────┤
│                        Storage                               │
│              (SQLite, Redis, Memory)                         │
└─────────────────────────────────────────────────────────────┘
```

## License

MIT License - see [LICENSE](LICENSE) for details.
