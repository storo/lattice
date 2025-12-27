# Patterns

Lattice provides several agent orchestration patterns for complex workflows.

## ReAct Pattern

ReAct (Reasoning + Acting) enables agents to alternate between thinking and acting, iterating until a final answer is reached.

### Basic Usage

```go
import "github.com/storo/lettice/pkg/patterns"

// Wrap an agent with ReAct pattern
agent := lattice.NewAgent("thinker").
    Model(llm).
    System("Think step by step.").
    Build()

reactAgent := patterns.NewReActAgent(agent)

// Run with reasoning iterations
result, err := reactAgent.Run(ctx, "Solve this problem...")
```

### Configuration

```go
reactAgent := patterns.NewReActAgent(agent,
    patterns.WithMaxIterations(5), // Default: 10
)
```

### How It Works

1. Agent receives input
2. Generates response in format:
   ```
   Thought: I need to...
   Action: tool_name
   ```
3. Continues iterating until no Action is specified
4. Returns final result

### Formatting Prompts

```go
prompt := patterns.FormatReActPrompt(
    "You are a problem solver.",
    agent.Tools(),
)
// Creates ReAct-formatted system prompt with tool descriptions
```

### Parsing Responses

```go
thought, action := patterns.ParseThoughtAction(response)
if action == "" {
    // Final answer reached
}
```

---

## Supervisor Pattern

Supervisors manage groups of worker agents, delegating tasks based on capabilities.

### Basic Usage

```go
supervisor := patterns.NewSupervisor(
    patterns.WithWorkers(researcher, writer, coder),
)

// Delegate to a worker with research capability
result, err := supervisor.Delegate(ctx, lattice.CapResearch, "Find info on AI")
```

### Strategies

```go
// Round-robin (default) - distributes evenly
supervisor := patterns.NewSupervisor(
    patterns.WithStrategy(patterns.StrategyRoundRobin),
)

// Race first - sends to all, returns first success
supervisor := patterns.NewSupervisor(
    patterns.WithStrategy(patterns.StrategyRaceFirst),
)

// All - sends to all, returns all results
supervisor := patterns.NewSupervisor(
    patterns.WithStrategy(patterns.StrategyAll),
)
```

### Dynamic Worker Management

```go
// Add workers
supervisor.AddWorker(newAgent)

// Remove workers
supervisor.RemoveWorker("agent-id")

// Get workers for a capability
workers := supervisor.WorkersFor(lattice.CapResearch)
```

### Broadcast to All Workers

```go
results, errs := supervisor.Broadcast(ctx, lattice.CapAnalysis, "Analyze this data")
for _, result := range results {
    fmt.Println(result.Output)
}
```

### Health Monitoring

```go
// Check health status
health := supervisor.Health()
// {"agent-1": true, "agent-2": false}

// Set health status
supervisor.SetHealth("agent-1", false)
```

---

## Sequential Pattern

Sequential pipelines pass output from one agent to the next.

### Basic Usage

```go
pipeline := patterns.NewSequential(
    researcher,  // Step 1: Research
    writer,      // Step 2: Write (receives research output)
    reviewer,    // Step 3: Review (receives written content)
)

result, err := pipeline.Run(ctx, "Research and write about AI")
// result.Output contains final reviewed content
```

### Pipeline Management

```go
// Get agents in pipeline
agents := pipeline.Agents()

// Add agents to end
pipeline.Append(editor, publisher)

// Add agents to beginning
pipeline.Prepend(planner)
```

### Result Aggregation

The final result includes:
- Output from the last agent
- Accumulated token counts from all agents
- Total duration for the entire pipeline

```go
result, _ := pipeline.Run(ctx, "input")

fmt.Println(result.Output)     // Final agent's output
fmt.Println(result.TokensIn)   // Sum of all agents' input tokens
fmt.Println(result.TokensOut)  // Sum of all agents' output tokens
fmt.Println(result.Duration)   // Total pipeline duration
```

---

## Parallel Pattern

Execute multiple agents concurrently with the same input.

### Run All

```go
parallel := patterns.NewParallel(analyst1, analyst2, analyst3)

results, errs := parallel.Run(ctx, "Analyze this data")

for _, result := range results {
    fmt.Println(result.Output)
}
for _, err := range errs {
    log.Printf("Error: %v", err)
}
```

### Race

Get the first successful result:

```go
result, err := parallel.Race(ctx, "Quick question")
// Returns as soon as one agent succeeds
```

### Aggregate

Combine results with a custom function:

```go
combined := parallel.Aggregate(ctx, "Analyze X", func(results []*patterns.AgentResult) string {
    var summary strings.Builder
    for _, r := range results {
        summary.WriteString(r.AgentID + ": " + r.Output + "\n")
    }
    return summary.String()
})
```

### Agent Management

```go
// Get agents
agents := parallel.Agents()

// Add agents
parallel.Add(newAgent1, newAgent2)
```

---

## Combining Patterns

Patterns can be combined for complex workflows:

```go
// ReAct agent that can delegate to a supervisor
reactAgent := patterns.NewReActAgent(mainAgent)

supervisor := patterns.NewSupervisor(
    patterns.WithWorkers(worker1, worker2, worker3),
    patterns.WithStrategy(patterns.StrategyRaceFirst),
)

// Pipeline: ReAct reasoning -> Parallel analysis -> Supervisor delegation
pipeline := patterns.NewSequential(reactAgent, ...)

result, _ := pipeline.Run(ctx, "Complex task...")
```

---

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/storo/lettice"
    "github.com/storo/lettice/pkg/patterns"
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
        Build()

    editor := lattice.NewAgent("editor").
        Model(llm).
        System("You edit and polish content.").
        Build()

    // Create pipeline: Research -> Write -> Edit
    pipeline := patterns.NewSequential(researcher, writer, editor)

    result, err := pipeline.Run(ctx, "Create an article about Go")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Output: %s\n", result.Output)
    fmt.Printf("Total tokens: %d in, %d out\n", result.TokensIn, result.TokensOut)
    fmt.Printf("Duration: %s\n", result.Duration)
}
```

## Next Steps

- [Mesh](mesh.md) - Mesh orchestration
- [Agents](agents.md) - Agent configuration
- [Middleware](middleware.md) - Add observability
