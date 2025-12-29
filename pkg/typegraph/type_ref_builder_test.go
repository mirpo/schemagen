package typegraph

import (
	"testing"

	"github.com/kaptinlin/jsonschema"
	"github.com/stretchr/testify/assert"
)

func TestTypeRefBuilder_Ref(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	config := &BuildConfig{}

	trb := NewTypeRefBuilder(registry, resolver, config)

	schema := &jsonschema.Schema{
		Ref: "#/$defs/MyType",
	}

	ref := trb.BuildTypeRef(schema, "")

	assert.Equal(t, KindRef, ref.Kind)
	assert.Equal(t, "MyType", ref.TypeName)
}

func TestTypeRefBuilder_Enum(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	config := &BuildConfig{ExtractInlined: false}

	trb := NewTypeRefBuilder(registry, resolver, config)

	schema := &jsonschema.Schema{
		Enum: []interface{}{"active", "inactive", "pending"},
	}

	ref := trb.BuildTypeRef(schema, "status")

	// With ExtractInlined=false, should remain inline
	assert.Equal(t, KindEnum, ref.Kind)
	assert.Len(t, ref.EnumValues, 3)
	assert.Equal(t, "string", ref.GoType)
}

func TestTypeRefBuilder_EnumExtracted(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	config := &BuildConfig{ExtractInlined: true}

	trb := NewTypeRefBuilder(registry, resolver, config)

	schema := &jsonschema.Schema{
		Enum: []interface{}{"active", "inactive", "pending"},
	}

	ref := trb.BuildTypeRef(schema, "status")

	// With ExtractInlined=true, should be extracted as separate type
	assert.Equal(t, KindRef, ref.Kind)
	assert.Equal(t, "Status", ref.TypeName)
	assert.Len(t, registry.All(), 1)
}

func TestTypeRefBuilder_Union_OneOf(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	config := &BuildConfig{}

	trb := NewTypeRefBuilder(registry, resolver, config)

	schema := &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{Type: []string{"string"}},
			{Type: []string{"integer"}},
		},
	}

	ref := trb.BuildTypeRef(schema, "")

	assert.Equal(t, KindUnion, ref.Kind)
	assert.Len(t, ref.UnionMembers, 2)
}

func TestTypeRefBuilder_Array(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	config := &BuildConfig{}

	trb := NewTypeRefBuilder(registry, resolver, config)

	schema := &jsonschema.Schema{
		Type: []string{"array"},
		Items: &jsonschema.Schema{
			Type: []string{"string"},
		},
	}

	ref := trb.BuildTypeRef(schema, "tags")

	assert.Equal(t, KindArray, ref.Kind)
	assert.NotNil(t, ref.ItemType)
	assert.Equal(t, KindPrimitive, ref.ItemType.Kind)
	assert.Equal(t, "string", ref.ItemType.GoType)
}

func TestTypeRefBuilder_ObjectAsMap(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	config := &BuildConfig{}

	trb := NewTypeRefBuilder(registry, resolver, config)

	// Object without properties becomes a map
	schema := &jsonschema.Schema{
		Type: []string{"object"},
	}

	ref := trb.BuildTypeRef(schema, "")

	assert.Equal(t, KindMap, ref.Kind)
	assert.NotNil(t, ref.ValueType)
}

func TestTypeRefBuilder_Primitive(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	config := &BuildConfig{}

	trb := NewTypeRefBuilder(registry, resolver, config)

	schema := &jsonschema.Schema{
		Type: []string{"string"},
	}

	ref := trb.BuildTypeRef(schema, "")

	assert.Equal(t, KindPrimitive, ref.Kind)
	assert.Equal(t, "string", ref.GoType)
}

func TestTypeRefBuilder_PrimitiveWithFormat(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	config := &BuildConfig{}

	trb := NewTypeRefBuilder(registry, resolver, config)

	format := "email"
	schema := &jsonschema.Schema{
		Type:   []string{"string"},
		Format: &format,
	}

	ref := trb.BuildTypeRef(schema, "")

	assert.Equal(t, KindPrimitive, ref.Kind)
	assert.Equal(t, "string", ref.GoType)
	assert.Equal(t, "email", ref.Format)
}

func TestTypeRefBuilder_MapPrimitiveType(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	config := &BuildConfig{}

	trb := NewTypeRefBuilder(registry, resolver, config)

	tests := []struct {
		schemaType []string
		format     *string
		expected   string
	}{
		{[]string{"string"}, nil, "string"},
		{[]string{"integer"}, nil, "int"},
		{[]string{"number"}, nil, "float64"},
		{[]string{"boolean"}, nil, "bool"},
		{[]string{"string"}, strPtr("uuid"), "uuid.UUID"},
		{[]string{"string"}, strPtr("date-time"), "time.Time"},
	}

	for _, tt := range tests {
		schema := &jsonschema.Schema{
			Type:   tt.schemaType,
			Format: tt.format,
		}
		result := trb.MapPrimitiveType(schema)
		assert.Equal(t, tt.expected, result)
	}
}

func TestTypeRefBuilder_ShouldExtractInlineObject(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	config := &BuildConfig{}

	trb := NewTypeRefBuilder(registry, resolver, config)

	// Object with properties should be extracted
	schemaWithProps := &jsonschema.Schema{
		Type: []string{"object"},
		Properties: &jsonschema.SchemaMap{
			"name": &jsonschema.Schema{Type: []string{"string"}},
		},
	}
	assert.True(t, trb.ShouldExtractInlineObject(schemaWithProps))

	// Object without properties should not be extracted
	schemaNoProps := &jsonschema.Schema{
		Type: []string{"object"},
	}
	assert.False(t, trb.ShouldExtractInlineObject(schemaNoProps))

	// $ref should not be extracted
	schemaRef := &jsonschema.Schema{
		Ref: "#/$defs/SomeType",
	}
	assert.False(t, trb.ShouldExtractInlineObject(schemaRef))
}

func TestTypeRefBuilder_Nullable(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	config := &BuildConfig{}

	trb := NewTypeRefBuilder(registry, resolver, config)

	schema := &jsonschema.Schema{
		Type: []string{"string", "null"},
	}

	ref := trb.BuildTypeRef(schema, "")

	assert.True(t, ref.Nullable)
	assert.Equal(t, KindPrimitive, ref.Kind)
}

func strPtr(s string) *string {
	return &s
}
