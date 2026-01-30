package typegraph

import (
	"testing"

	"github.com/kaptinlin/jsonschema"
	"github.com/stretchr/testify/assert"
)

func TestTypeRefBuilder_BuildTypeRef(t *testing.T) {
	t.Run("ref", func(t *testing.T) {
		trb := NewTypeRefBuilder(NewTypeRegistry(), NewRefResolver(nil), &BuildConfig{})
		ref := trb.BuildTypeRef(&jsonschema.Schema{Ref: "#/$defs/MyType"}, "")
		assert.Equal(t, KindRef, ref.Kind)
		assert.Equal(t, "MyType", ref.TypeName)
	})

	t.Run("enum inline", func(t *testing.T) {
		trb := NewTypeRefBuilder(NewTypeRegistry(), NewRefResolver(nil), &BuildConfig{ExtractInlined: false})
		ref := trb.BuildTypeRef(&jsonschema.Schema{Enum: []interface{}{"a", "b", "c"}}, "status")
		assert.Equal(t, KindEnum, ref.Kind)
		assert.Len(t, ref.EnumValues, 3)
	})

	t.Run("enum extracted", func(t *testing.T) {
		registry := NewTypeRegistry()
		trb := NewTypeRefBuilder(registry, NewRefResolver(nil), &BuildConfig{ExtractInlined: true})
		ref := trb.BuildTypeRef(&jsonschema.Schema{Enum: []interface{}{"a", "b", "c"}}, "status")
		assert.Equal(t, KindRef, ref.Kind)
		assert.Equal(t, "Status", ref.TypeName)
		assert.Len(t, registry.All(), 1)
	})

	t.Run("union oneOf", func(t *testing.T) {
		trb := NewTypeRefBuilder(NewTypeRegistry(), NewRefResolver(nil), &BuildConfig{})
		ref := trb.BuildTypeRef(&jsonschema.Schema{
			OneOf: []*jsonschema.Schema{{Type: []string{"string"}}, {Type: []string{"integer"}}},
		}, "")
		assert.Equal(t, KindUnion, ref.Kind)
		assert.Len(t, ref.UnionMembers, 2)
	})

	t.Run("array", func(t *testing.T) {
		trb := NewTypeRefBuilder(NewTypeRegistry(), NewRefResolver(nil), &BuildConfig{})
		ref := trb.BuildTypeRef(&jsonschema.Schema{
			Type:  []string{"array"},
			Items: &jsonschema.Schema{Type: []string{"string"}},
		}, "tags")
		assert.Equal(t, KindArray, ref.Kind)
		assert.Equal(t, KindPrimitive, ref.ItemType.Kind)
	})

	t.Run("object as map", func(t *testing.T) {
		trb := NewTypeRefBuilder(NewTypeRegistry(), NewRefResolver(nil), &BuildConfig{})
		ref := trb.BuildTypeRef(&jsonschema.Schema{Type: []string{"object"}}, "")
		assert.Equal(t, KindMap, ref.Kind)
	})

	t.Run("primitive", func(t *testing.T) {
		trb := NewTypeRefBuilder(NewTypeRegistry(), NewRefResolver(nil), &BuildConfig{})
		ref := trb.BuildTypeRef(&jsonschema.Schema{Type: []string{"string"}}, "")
		assert.Equal(t, KindPrimitive, ref.Kind)
		assert.Equal(t, "string", ref.GoType)
	})

	t.Run("primitive with format", func(t *testing.T) {
		trb := NewTypeRefBuilder(NewTypeRegistry(), NewRefResolver(nil), &BuildConfig{})
		format := "email"
		ref := trb.BuildTypeRef(&jsonschema.Schema{Type: []string{"string"}, Format: &format}, "")
		assert.Equal(t, "email", ref.Format)
	})

	t.Run("nullable", func(t *testing.T) {
		trb := NewTypeRefBuilder(NewTypeRegistry(), NewRefResolver(nil), &BuildConfig{})
		ref := trb.BuildTypeRef(&jsonschema.Schema{Type: []string{"string", "null"}}, "")
		assert.True(t, ref.Nullable)
		assert.Equal(t, KindPrimitive, ref.Kind)
	})
}

func TestTypeRefBuilder_MapPrimitiveType(t *testing.T) {
	trb := NewTypeRefBuilder(NewTypeRegistry(), NewRefResolver(nil), &BuildConfig{})
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
		schema := &jsonschema.Schema{Type: tt.schemaType, Format: tt.format}
		assert.Equal(t, tt.expected, trb.MapPrimitiveType(schema))
	}
}

func TestTypeRefBuilder_ShouldExtractInlineObject(t *testing.T) {
	trb := NewTypeRefBuilder(NewTypeRegistry(), NewRefResolver(nil), &BuildConfig{})

	assert.True(t, trb.ShouldExtractInlineObject(&jsonschema.Schema{
		Type:       []string{"object"},
		Properties: &jsonschema.SchemaMap{"name": &jsonschema.Schema{Type: []string{"string"}}},
	}))
	assert.False(t, trb.ShouldExtractInlineObject(&jsonschema.Schema{Type: []string{"object"}}))
	assert.False(t, trb.ShouldExtractInlineObject(&jsonschema.Schema{Ref: "#/$defs/SomeType"}))
}

func strPtr(s string) *string { return &s }
