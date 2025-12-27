package mesh

import (
	"sync/atomic"

	"github.com/storo/lettice/pkg/core"
)

// Balancer selects an agent from a list of providers.
type Balancer interface {
	Select(agents []core.Agent) core.Agent
}

// RoundRobinBalancer distributes requests evenly across agents.
type RoundRobinBalancer struct {
	counter uint64
}

// NewRoundRobinBalancer creates a new round-robin balancer.
func NewRoundRobinBalancer() *RoundRobinBalancer {
	return &RoundRobinBalancer{}
}

// Select picks the next agent in round-robin order.
func (b *RoundRobinBalancer) Select(agents []core.Agent) core.Agent {
	if len(agents) == 0 {
		return nil
	}
	idx := atomic.AddUint64(&b.counter, 1) - 1
	return agents[idx%uint64(len(agents))]
}

// RandomBalancer selects a random agent.
type RandomBalancer struct {
	randFunc func(n int) int
}

// NewRandomBalancer creates a new random balancer.
func NewRandomBalancer(randFunc func(n int) int) *RandomBalancer {
	return &RandomBalancer{randFunc: randFunc}
}

// Select picks a random agent.
func (b *RandomBalancer) Select(agents []core.Agent) core.Agent {
	if len(agents) == 0 {
		return nil
	}
	return agents[b.randFunc(len(agents))]
}

// FirstBalancer always selects the first agent.
type FirstBalancer struct{}

// NewFirstBalancer creates a new first balancer.
func NewFirstBalancer() *FirstBalancer {
	return &FirstBalancer{}
}

// Select picks the first agent.
func (b *FirstBalancer) Select(agents []core.Agent) core.Agent {
	if len(agents) == 0 {
		return nil
	}
	return agents[0]
}
