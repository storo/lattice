# HTTP API

Lattice provides a REST API to expose your mesh to external clients.

## Quick Start

```go
import (
    "github.com/storo/lettice/pkg/protocol/http"
    "github.com/storo/lettice/pkg/security"
)

// Create server with auth
server := http.NewServer(mesh, http.WithAuth(auth))
server.ListenAndServe(":8080")
```

## Endpoints

### Health Check

Check if the server is running.

```
GET /health
```

**No authentication required.**

**Response:**

```json
{
  "status": "ok",
  "time": "2024-01-15T10:30:00Z"
}
```

**Example:**

```bash
curl http://localhost:8080/health
```

---

### List Agents

Get all registered agents.

```
GET /agents
```

**Authentication required.**

**Response:**

```json
{
  "agents": [
    {
      "id": "abc-123",
      "name": "researcher",
      "description": "Research expert",
      "provides": ["research"],
      "needs": []
    },
    {
      "id": "def-456",
      "name": "writer",
      "description": "Content writer",
      "provides": ["writing"],
      "needs": ["research"]
    }
  ]
}
```

**Example:**

```bash
curl -H "X-API-Key: demo-key" http://localhost:8080/agents
```

---

### Get Agent

Get info about a specific agent.

```
GET /agents/{id}
```

**Authentication required.**

**Response:**

```json
{
  "id": "abc-123",
  "name": "researcher",
  "description": "Research expert",
  "provides": ["research"],
  "needs": []
}
```

**Example:**

```bash
curl -H "X-API-Key: demo-key" http://localhost:8080/agents/abc-123
```

**Errors:**

- `404 Not Found` - Agent doesn't exist

---

### Run Agent

Execute a specific agent.

```
POST /agents/{id}/run
```

**Authentication required.**

**Request Body:**

```json
{
  "input": "Research the latest AI trends",
  "context": "Optional additional context"
}
```

**Response:**

```json
{
  "output": "Here are the latest AI trends...",
  "tokens_in": 150,
  "tokens_out": 500,
  "duration": "2.5s",
  "trace_id": "trace-abc-123"
}
```

**Example:**

```bash
curl -X POST \
  -H "X-API-Key: demo-key" \
  -H "Content-Type: application/json" \
  -d '{"input": "Research AI trends"}' \
  http://localhost:8080/agents/abc-123/run
```

**Errors:**

- `400 Bad Request` - Invalid request body
- `404 Not Found` - Agent doesn't exist
- `500 Internal Server Error` - Execution failed

---

### Run on Mesh

Execute on the mesh (auto-selects agent).

```
POST /mesh/run
```

**Authentication required.**

**Request Body:**

```json
{
  "input": "What is machine learning?"
}
```

**Response:**

```json
{
  "output": "Machine learning is...",
  "tokens_in": 50,
  "tokens_out": 200,
  "duration": "1.2s",
  "trace_id": "trace-def-456"
}
```

**Example:**

```bash
curl -X POST \
  -H "X-API-Key: demo-key" \
  -H "Content-Type: application/json" \
  -d '{"input": "What is machine learning?"}' \
  http://localhost:8080/mesh/run
```

---

## Authentication

All endpoints except `/health` require authentication.

### API Key

```bash
curl -H "X-API-Key: your-api-key" http://localhost:8080/agents
```

### JWT Token

```bash
curl -H "Authorization: Bearer eyJhbG..." http://localhost:8080/agents
```

### Errors

**401 Unauthorized:**

```json
{
  "error": "unauthorized"
}
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": "error message here"
}
```

| Status | Meaning |
|--------|---------|
| 400 | Bad request (invalid JSON, missing fields) |
| 401 | Unauthorized (missing or invalid credentials) |
| 404 | Not found (agent doesn't exist) |
| 405 | Method not allowed (wrong HTTP method) |
| 500 | Internal server error (execution failed) |

---

## Server Configuration

### Basic Server

```go
server := http.NewServer(mesh)
server.ListenAndServe(":8080")
```

### With Authentication

```go
server := http.NewServer(mesh, http.WithAuth(auth))
server.ListenAndServe(":8080")
```

### Graceful Shutdown

```go
server := http.NewServer(mesh)

go func() {
    server.ListenAndServe(":8080")
}()

// Wait for signal
<-quit

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
server.Shutdown(ctx)
```

---

## Complete Example

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/storo/lettice"
    "github.com/storo/lettice/pkg/protocol/http"
    "github.com/storo/lettice/pkg/provider"
    "github.com/storo/lettice/pkg/security"
)

func main() {
    // Create mock provider
    llm := provider.NewMockWithResponse("Hello from the mesh!")

    // Create agents
    agent := lattice.NewAgent("assistant").
        Model(llm).
        System("You are helpful.").
        Provides(lattice.CapWriting).
        Build()

    // Create mesh
    mesh := lattice.NewMesh()
    mesh.Register(agent)

    // Setup auth
    apiKeyAuth := security.NewAPIKeyAuth()
    apiKeyAuth.RegisterKey("demo-key", &security.KeyEntry{
        AgentID: "demo",
        Roles:   []string{"user"},
    })
    auth := security.NewAuth(security.WithAPIKeyAuth(apiKeyAuth))

    // Create and start server
    server := http.NewServer(mesh, http.WithAuth(auth))

    go func() {
        log.Println("Server starting on :8080")
        if err := server.ListenAndServe(":8080"); err != nil {
            log.Printf("Server error: %v", err)
        }
    }()

    // Wait for shutdown signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down...")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    server.Shutdown(ctx)
}
```

---

## Testing with curl

```bash
# Health check
curl http://localhost:8080/health

# List agents
curl -H "X-API-Key: demo-key" http://localhost:8080/agents

# Get specific agent
curl -H "X-API-Key: demo-key" http://localhost:8080/agents/abc-123

# Run specific agent
curl -X POST \
  -H "X-API-Key: demo-key" \
  -H "Content-Type: application/json" \
  -d '{"input": "Hello!"}' \
  http://localhost:8080/agents/abc-123/run

# Run on mesh
curl -X POST \
  -H "X-API-Key: demo-key" \
  -H "Content-Type: application/json" \
  -d '{"input": "Hello!"}' \
  http://localhost:8080/mesh/run
```

## Next Steps

- [Security](security.md) - Configure authentication
- [Mesh](mesh.md) - Mesh configuration
- [Getting Started](getting-started.md) - Quick start guide
