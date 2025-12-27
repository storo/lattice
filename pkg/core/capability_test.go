package core

import (
	"testing"
)

func TestCapability_String(t *testing.T) {
	tests := []struct {
		name     string
		cap      Capability
		expected string
	}{
		{"research capability", CapResearch, "research"},
		{"writing capability", CapWriting, "writing"},
		{"coding capability", CapCoding, "coding"},
		{"analysis capability", CapAnalysis, "analysis"},
		{"planning capability", CapPlanning, "planning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.cap) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.cap))
			}
		})
	}
}

func TestCap_CreatesCustomCapability(t *testing.T) {
	custom := Cap("custom-skill")
	if string(custom) != "custom-skill" {
		t.Errorf("expected custom-skill, got %s", string(custom))
	}
}

func TestCapability_Equality(t *testing.T) {
	cap1 := Cap("research")
	cap2 := CapResearch

	if cap1 != cap2 {
		t.Errorf("expected capabilities to be equal")
	}
}

func TestCapabilitySet_Add(t *testing.T) {
	set := NewCapabilitySet()
	set.Add(CapResearch)
	set.Add(CapWriting)

	if !set.Has(CapResearch) {
		t.Error("expected set to contain research")
	}
	if !set.Has(CapWriting) {
		t.Error("expected set to contain writing")
	}
	if set.Has(CapCoding) {
		t.Error("expected set to not contain coding")
	}
}

func TestCapabilitySet_Remove(t *testing.T) {
	set := NewCapabilitySet()
	set.Add(CapResearch)
	set.Add(CapWriting)
	set.Remove(CapResearch)

	if set.Has(CapResearch) {
		t.Error("expected set to not contain research after removal")
	}
	if !set.Has(CapWriting) {
		t.Error("expected set to still contain writing")
	}
}

func TestCapabilitySet_List(t *testing.T) {
	set := NewCapabilitySet()
	set.Add(CapResearch)
	set.Add(CapWriting)

	list := set.List()
	if len(list) != 2 {
		t.Errorf("expected 2 capabilities, got %d", len(list))
	}
}

func TestCapabilitySet_Intersect(t *testing.T) {
	set1 := NewCapabilitySet()
	set1.Add(CapResearch)
	set1.Add(CapWriting)

	set2 := NewCapabilitySet()
	set2.Add(CapWriting)
	set2.Add(CapCoding)

	intersection := set1.Intersect(set2)
	if !intersection.Has(CapWriting) {
		t.Error("expected intersection to contain writing")
	}
	if intersection.Has(CapResearch) {
		t.Error("expected intersection to not contain research")
	}
	if intersection.Has(CapCoding) {
		t.Error("expected intersection to not contain coding")
	}
}

func TestCapabilitySet_List_DeterministicOrder(t *testing.T) {
	set := NewCapabilitySet()
	// Add in non-alphabetical order
	set.Add(Cap("zebra"))
	set.Add(Cap("alpha"))
	set.Add(Cap("mango"))

	// Get list multiple times and verify consistency
	list1 := set.List()
	list2 := set.List()
	list3 := set.List()

	// Should be the same each time
	for i := range list1 {
		if list1[i] != list2[i] || list2[i] != list3[i] {
			t.Error("expected List() to be deterministic")
		}
	}

	// Should be alphabetically sorted
	expected := []Capability{Cap("alpha"), Cap("mango"), Cap("zebra")}
	for i, cap := range expected {
		if list1[i] != cap {
			t.Errorf("expected list[%d] = %s, got %s", i, cap, list1[i])
		}
	}
}
