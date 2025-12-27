package core

import (
	"encoding/json"
	"reflect"
	"strings"
)

// SchemaProperty represents a property in a JSON Schema.
type SchemaProperty struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// Schema represents a JSON Schema object.
type Schema struct {
	Type       string                    `json:"type"`
	Properties map[string]SchemaProperty `json:"properties,omitempty"`
	Required   []string                  `json:"required,omitempty"`
}

// ToJSON converts the schema to json.RawMessage.
func (s *Schema) ToJSON() json.RawMessage {
	data, _ := json.Marshal(s)
	return data
}

// SchemaBuilder provides a fluent API for building JSON Schemas.
type SchemaBuilder struct {
	schema *Schema
}

// NewObjectSchema creates a new SchemaBuilder for an object schema.
func NewObjectSchema() *SchemaBuilder {
	return &SchemaBuilder{
		schema: &Schema{
			Type:       "object",
			Properties: make(map[string]SchemaProperty),
			Required:   []string{},
		},
	}
}

// Property adds a property to the schema.
func (b *SchemaBuilder) Property(name, typ, desc string) *SchemaBuilder {
	b.schema.Properties[name] = SchemaProperty{
		Type:        typ,
		Description: desc,
	}
	return b
}

// Required marks fields as required.
func (b *SchemaBuilder) Required(names ...string) *SchemaBuilder {
	b.schema.Required = append(b.schema.Required, names...)
	return b
}

// Build returns the schema as json.RawMessage.
func (b *SchemaBuilder) Build() json.RawMessage {
	return b.schema.ToJSON()
}

// SchemaFromStruct generates a JSON Schema from a Go struct using reflection.
// It uses `json` tags for property names and `schema` tags for descriptions.
// Fields without json tags or with json:"-" are ignored.
// Fields with json:",omitempty" are not marked as required.
func SchemaFromStruct(v any) json.RawMessage {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]SchemaProperty),
		Required:   []string{},
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Get json tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse json tag (name,options)
		name, options := parseJSONTag(jsonTag)
		if name == "" {
			continue
		}

		// Get description from schema tag
		desc := field.Tag.Get("schema")

		// Map Go type to JSON Schema type
		jsonType := goTypeToJSONType(field.Type)

		schema.Properties[name] = SchemaProperty{
			Type:        jsonType,
			Description: desc,
		}

		// If not omitempty, mark as required
		if !strings.Contains(options, "omitempty") {
			schema.Required = append(schema.Required, name)
		}
	}

	return schema.ToJSON()
}

// parseJSONTag parses a json tag into name and options.
func parseJSONTag(tag string) (name, options string) {
	parts := strings.SplitN(tag, ",", 2)
	name = parts[0]
	if len(parts) > 1 {
		options = parts[1]
	}
	return
}

// goTypeToJSONType maps Go types to JSON Schema types.
func goTypeToJSONType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	case reflect.Ptr:
		return goTypeToJSONType(t.Elem())
	default:
		return "string"
	}
}
