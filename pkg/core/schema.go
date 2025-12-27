package core

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
)

// SchemaProperty represents a property in a JSON Schema.
// Supports nested objects with their own properties.
type SchemaProperty struct {
	Type        string                    `json:"type"`
	Description string                    `json:"description,omitempty"`
	Properties  map[string]SchemaProperty `json:"properties,omitempty"`
	Required    []string                  `json:"required,omitempty"`
	Items       *SchemaProperty           `json:"items,omitempty"`
}

// Schema represents a JSON Schema object.
type Schema struct {
	Type       string                    `json:"type"`
	Properties map[string]SchemaProperty `json:"properties,omitempty"`
	Required   []string                  `json:"required,omitempty"`
}

// ToJSON converts the schema to json.RawMessage with sorted keys.
func (s *Schema) ToJSON() json.RawMessage {
	// Sort required fields
	sort.Strings(s.Required)

	// Use a custom marshaler for deterministic property order
	data, _ := json.Marshal(s.toOrderedMap())
	return data
}

// toOrderedMap converts schema to a map with sorted keys for deterministic JSON.
func (s *Schema) toOrderedMap() map[string]any {
	result := map[string]any{
		"type": s.Type,
	}

	if len(s.Properties) > 0 {
		props := make(map[string]any)
		for name, prop := range s.Properties {
			props[name] = propToOrderedMap(prop)
		}
		result["properties"] = props
	}

	if len(s.Required) > 0 {
		result["required"] = s.Required
	}

	return result
}

func propToOrderedMap(p SchemaProperty) map[string]any {
	result := map[string]any{
		"type": p.Type,
	}

	if p.Description != "" {
		result["description"] = p.Description
	}

	if len(p.Properties) > 0 {
		props := make(map[string]any)
		for name, prop := range p.Properties {
			props[name] = propToOrderedMap(prop)
		}
		result["properties"] = props
	}

	if len(p.Required) > 0 {
		sort.Strings(p.Required)
		result["required"] = p.Required
	}

	if p.Items != nil {
		result["items"] = propToOrderedMap(*p.Items)
	}

	return result
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
// Nested structs are recursively processed.
func SchemaFromStruct(v any) json.RawMessage {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schema := generateSchemaForType(t)
	return schema.ToJSON()
}

// generateSchemaForType creates a Schema from a reflect.Type.
func generateSchemaForType(t reflect.Type) *Schema {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]SchemaProperty),
		Required:   []string{},
	}

	if t.Kind() != reflect.Struct {
		return schema
	}

	// Collect field names and sort them for deterministic order
	type fieldInfo struct {
		name       string
		jsonName   string
		options    string
		fieldType  reflect.Type
		desc       string
		isRequired bool
	}

	var fields []fieldInfo

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

		isRequired := !strings.Contains(options, "omitempty")

		fields = append(fields, fieldInfo{
			name:       field.Name,
			jsonName:   name,
			options:    options,
			fieldType:  field.Type,
			desc:       desc,
			isRequired: isRequired,
		})
	}

	// Sort fields by json name for deterministic output
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].jsonName < fields[j].jsonName
	})

	for _, f := range fields {
		prop := generatePropertyForType(f.fieldType, f.desc)
		schema.Properties[f.jsonName] = prop

		if f.isRequired {
			schema.Required = append(schema.Required, f.jsonName)
		}
	}

	// Required is already sorted because fields are sorted
	return schema
}

// generatePropertyForType creates a SchemaProperty for a reflect.Type.
func generatePropertyForType(t reflect.Type, desc string) SchemaProperty {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	prop := SchemaProperty{
		Type:        goTypeToJSONType(t),
		Description: desc,
	}

	// Handle nested structs
	if t.Kind() == reflect.Struct {
		nestedSchema := generateSchemaForType(t)
		prop.Properties = nestedSchema.Properties
		prop.Required = nestedSchema.Required
	}

	// Handle slices/arrays of structs
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		elemType := t.Elem()
		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}
		if elemType.Kind() == reflect.Struct {
			itemProp := generatePropertyForType(elemType, "")
			prop.Items = &itemProp
		} else {
			prop.Items = &SchemaProperty{Type: goTypeToJSONType(elemType)}
		}
	}

	return prop
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
