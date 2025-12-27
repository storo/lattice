# Middleware

Lattice provides middleware for observability: metrics collection, logging, and tracing.

## Metrics

### MetricsCollector

Collects and aggregates metrics across agents.

```go
import "github.com/storo/lettice/pkg/middleware"

collector := middleware.NewMetricsCollector()
```

### Wrapping Agents

```go
// Wrap a single agent
agent := lattice.NewAgent("assistant").Model(llm).Build()
wrappedAgent := middleware.WrapWithMetrics(agent, collector)

// Or use the middleware function
wrap := middleware.MetricsMiddleware(collector)
wrappedAgent := wrap(agent)
```

### Recording Metrics

Metrics are recorded automatically when wrapped agents execute:

```go
result, err := wrappedAgent.Run(ctx, "Hello")
// Metrics are now recorded for this execution
```

### Accessing Metrics

#### Per-Agent Metrics

```go
metrics := collector.GetAgentMetrics("agent-id")

fmt.Printf("Agent: %s\n", metrics.AgentName)
fmt.Printf("Total calls: %d\n", metrics.TotalCalls)
fmt.Printf("Success: %d\n", metrics.SuccessCount)
fmt.Printf("Errors: %d\n", metrics.ErrorCount)
fmt.Printf("Tokens in: %d\n", metrics.TotalTokensIn)
fmt.Printf("Tokens out: %d\n", metrics.TotalTokensOut)
fmt.Printf("Avg duration: %s\n", metrics.AvgDuration)
fmt.Printf("Min duration: %s\n", metrics.MinDuration)
fmt.Printf("Max duration: %s\n", metrics.MaxDuration)
fmt.Printf("Last call: %s\n", metrics.LastCallTime)
```

#### All Agents

```go
allMetrics := collector.AllMetrics()
for _, m := range allMetrics {
    fmt.Printf("%s: %d calls, %s avg\n",
        m.AgentName, m.TotalCalls, m.AvgDuration)
}
```

#### Summary

```go
summary := collector.Summary()

fmt.Printf("Agents: %d\n", summary.AgentCount)
fmt.Printf("Total calls: %d\n", summary.TotalCalls)
fmt.Printf("Success rate: %.2f%%\n", summary.SuccessRate * 100)
fmt.Printf("Avg duration: %s\n", summary.AvgDuration)
```

### Reset Metrics

```go
collector.Reset()
```

---

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/storo/lettice"
    "github.com/storo/lettice/pkg/middleware"
    "github.com/storo/lettice/pkg/provider"
)

func main() {
    ctx := context.Background()

    // Create metrics collector
    collector := middleware.NewMetricsCollector()

    // Create provider and agents
    llm := provider.NewMockWithResponse("Hello!")

    researcher := middleware.WrapWithMetrics(
        lattice.NewAgent("researcher").
            Model(llm).
            Provides(lattice.CapResearch).
            Build(),
        collector,
    )

    writer := middleware.WrapWithMetrics(
        lattice.NewAgent("writer").
            Model(llm).
            Provides(lattice.CapWriting).
            Build(),
        collector,
    )

    // Create mesh with wrapped agents
    mesh := lattice.NewMesh()
    mesh.Register(researcher, writer)

    // Execute some tasks
    for i := 0; i < 10; i++ {
        mesh.Run(ctx, "Do something")
    }

    // Report metrics
    fmt.Println("\n=== Metrics Report ===")

    summary := collector.Summary()
    fmt.Printf("Total calls: %d\n", summary.TotalCalls)
    fmt.Printf("Success rate: %.2f%%\n", summary.SuccessRate*100)
    fmt.Printf("Avg duration: %s\n", summary.AvgDuration)

    fmt.Println("\nPer-agent metrics:")
    for _, m := range collector.AllMetrics() {
        fmt.Printf("  %s: %d calls, %s avg\n",
            m.AgentName, m.TotalCalls, m.AvgDuration)
    }
}
```

---

## Periodic Reporting

```go
func reportMetrics(collector *middleware.MetricsCollector) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        summary := collector.Summary()
        if summary.TotalCalls > 0 {
            log.Printf("Metrics: calls=%d success_rate=%.2f%% avg=%s",
                summary.TotalCalls,
                summary.SuccessRate*100,
                summary.AvgDuration,
            )
        }
    }
}

// Start in background
go reportMetrics(collector)
```

---

## Integration with HTTP Server

```go
package main

import (
    "encoding/json"
    "log"
    "net/http"

    "github.com/storo/lettice"
    latticehttp "github.com/storo/lettice/pkg/protocol/http"
    "github.com/storo/lettice/pkg/middleware"
    "github.com/storo/lettice/pkg/provider"
)

func main() {
    collector := middleware.NewMetricsCollector()
    llm := provider.NewMockWithResponse("Hello!")

    // Wrap agent with metrics
    agent := middleware.WrapWithMetrics(
        lattice.NewAgent("assistant").Model(llm).Build(),
        collector,
    )

    mesh := lattice.NewMesh()
    mesh.Register(agent)

    // Add metrics endpoint
    http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]any{
            "summary": collector.Summary(),
            "agents":  collector.AllMetrics(),
        })
    })

    // Start server
    server := latticehttp.NewServer(mesh)
    go server.ListenAndServe(":8080")

    // Serve metrics on different port
    log.Fatal(http.ListenAndServe(":9090", nil))
}
```

Access metrics at `http://localhost:9090/metrics`

---

## AgentMetrics Structure

```go
type AgentMetrics struct {
    AgentID        string        // Agent identifier
    AgentName      string        // Agent name
    TotalCalls     int64         // Total execution count
    SuccessCount   int64         // Successful executions
    ErrorCount     int64         // Failed executions
    TotalTokensIn  int64         // Total input tokens
    TotalTokensOut int64         // Total output tokens
    TotalDuration  time.Duration // Total execution time
    AvgDuration    time.Duration // Average execution time
    MaxDuration    time.Duration // Maximum execution time
    MinDuration    time.Duration // Minimum execution time
    LastCallTime   time.Time     // Time of last execution
}
```

## MetricsSummary Structure

```go
type MetricsSummary struct {
    AgentCount  int           // Number of agents tracked
    TotalCalls  int64         // Total calls across all agents
    SuccessRate float64       // Success ratio (0.0 - 1.0)
    AvgDuration time.Duration // Average duration across all calls
}
```

## Next Steps

- [Patterns](patterns.md) - Orchestration patterns
- [HTTP API](http-api.md) - REST API reference
- [Agents](agents.md) - Agent configuration
