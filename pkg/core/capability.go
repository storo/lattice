package core

import "sort"

// Capability represents a skill or functionality that an agent provides or needs.
type Capability string

// Predefined capabilities
const (
	CapResearch Capability = "research"
	CapWriting  Capability = "writing"
	CapCoding   Capability = "coding"
	CapAnalysis Capability = "analysis"
	CapPlanning Capability = "planning"
)

// Cap creates a custom capability from a string.
func Cap(name string) Capability {
	return Capability(name)
}

// CapabilitySet is a set of capabilities for efficient lookup.
type CapabilitySet struct {
	caps map[Capability]struct{}
}

// NewCapabilitySet creates a new empty CapabilitySet.
func NewCapabilitySet() *CapabilitySet {
	return &CapabilitySet{
		caps: make(map[Capability]struct{}),
	}
}

// Add adds a capability to the set.
func (s *CapabilitySet) Add(cap Capability) {
	s.caps[cap] = struct{}{}
}

// Remove removes a capability from the set.
func (s *CapabilitySet) Remove(cap Capability) {
	delete(s.caps, cap)
}

// Has checks if a capability is in the set.
func (s *CapabilitySet) Has(cap Capability) bool {
	_, ok := s.caps[cap]
	return ok
}

// List returns all capabilities in the set, sorted alphabetically for deterministic output.
func (s *CapabilitySet) List() []Capability {
	result := make([]Capability, 0, len(s.caps))
	for cap := range s.caps {
		result = append(result, cap)
	}
	// Sort for deterministic order
	sort.Slice(result, func(i, j int) bool {
		return string(result[i]) < string(result[j])
	})
	return result
}

// Intersect returns a new set with capabilities present in both sets.
func (s *CapabilitySet) Intersect(other *CapabilitySet) *CapabilitySet {
	result := NewCapabilitySet()
	for cap := range s.caps {
		if other.Has(cap) {
			result.Add(cap)
		}
	}
	return result
}
