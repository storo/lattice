package core

import (
	"encoding/json"
	"testing"
)

func TestSchemaFromStruct_BasicTypes(t *testing.T) {
	type TestStruct struct {
		Name    string  `json:"name" schema:"The name"`
		Age     int     `json:"age" schema:"The age"`
		Score   float64 `json:"score" schema:"The score"`
		Active  bool    `json:"active" schema:"Is active"`
	}

	schema := SchemaFromStruct(TestStruct{})

	var parsed Schema
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	if parsed.Type != "object" {
		t.Errorf("expected type object, got %s", parsed.Type)
	}

	tests := []struct {
		name     string
		propType string
		desc     string
	}{
		{"name", "string", "The name"},
		{"age", "integer", "The age"},
		{"score", "number", "The score"},
		{"active", "boolean", "Is active"},
	}

	for _, tt := range tests {
		prop, ok := parsed.Properties[tt.name]
		if !ok {
			t.Errorf("expected property %s to exist", tt.name)
			continue
		}
		if prop.Type != tt.propType {
			t.Errorf("expected %s type %s, got %s", tt.name, tt.propType, prop.Type)
		}
		if prop.Description != tt.desc {
			t.Errorf("expected %s description '%s', got '%s'", tt.name, tt.desc, prop.Description)
		}
	}
}

func TestSchemaFromStruct_RequiredFields(t *testing.T) {
	type TestStruct struct {
		Required string `json:"required" schema:"Required field"`
		Optional string `json:"optional,omitempty" schema:"Optional field"`
	}

	schema := SchemaFromStruct(TestStruct{})

	var parsed Schema
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	// Check that "required" is in the required list
	found := false
	for _, name := range parsed.Required {
		if name == "required" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'required' field to be in required list")
	}

	// Check that "optional" is NOT in the required list
	for _, name := range parsed.Required {
		if name == "optional" {
			t.Error("expected 'optional' field to NOT be in required list")
		}
	}
}

func TestSchemaFromStruct_IgnoresUntaggedFields(t *testing.T) {
	type TestStruct struct {
		Tagged   string `json:"tagged"`
		Untagged string
		Ignored  string `json:"-"`
	}

	schema := SchemaFromStruct(TestStruct{})

	var parsed Schema
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	if _, ok := parsed.Properties["tagged"]; !ok {
		t.Error("expected tagged field to be in properties")
	}
	if _, ok := parsed.Properties["Untagged"]; ok {
		t.Error("expected untagged field to NOT be in properties")
	}
	if _, ok := parsed.Properties["Ignored"]; ok {
		t.Error("expected ignored field to NOT be in properties")
	}
}

func TestSchemaFromStruct_PointerInput(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
	}

	schema := SchemaFromStruct(&TestStruct{})

	var parsed Schema
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	if _, ok := parsed.Properties["name"]; !ok {
		t.Error("expected name field to be in properties when using pointer")
	}
}

func TestSchemaBuilder_Basic(t *testing.T) {
	schema := NewObjectSchema().
		Property("task", "string", "The task to perform").
		Property("priority", "integer", "Priority level").
		Required("task").
		Build()

	var parsed Schema
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	if parsed.Type != "object" {
		t.Errorf("expected type object, got %s", parsed.Type)
	}

	taskProp, ok := parsed.Properties["task"]
	if !ok {
		t.Fatal("expected task property to exist")
	}
	if taskProp.Type != "string" {
		t.Errorf("expected task type string, got %s", taskProp.Type)
	}
	if taskProp.Description != "The task to perform" {
		t.Errorf("expected task description 'The task to perform', got '%s'", taskProp.Description)
	}

	if len(parsed.Required) != 1 || parsed.Required[0] != "task" {
		t.Errorf("expected required [task], got %v", parsed.Required)
	}
}

func TestSchemaBuilder_MultipleRequired(t *testing.T) {
	schema := NewObjectSchema().
		Property("a", "string", "").
		Property("b", "string", "").
		Property("c", "string", "").
		Required("a", "b").
		Build()

	var parsed Schema
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	if len(parsed.Required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(parsed.Required))
	}
}

func TestSchemaFromStruct_ArrayAndMap(t *testing.T) {
	type TestStruct struct {
		Items []string          `json:"items" schema:"List of items"`
		Tags  map[string]string `json:"tags" schema:"Key-value tags"`
	}

	schema := SchemaFromStruct(TestStruct{})

	var parsed Schema
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	itemsProp, ok := parsed.Properties["items"]
	if !ok {
		t.Fatal("expected items property to exist")
	}
	if itemsProp.Type != "array" {
		t.Errorf("expected items type array, got %s", itemsProp.Type)
	}

	tagsProp, ok := parsed.Properties["tags"]
	if !ok {
		t.Fatal("expected tags property to exist")
	}
	if tagsProp.Type != "object" {
		t.Errorf("expected tags type object, got %s", tagsProp.Type)
	}
}
