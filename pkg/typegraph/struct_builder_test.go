package typegraph

import (
	"testing"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFieldBuilder struct {
	buildTypeRefCalls int
	returnTypeRef     *TypeRef
	returnFields      []*Field
}

func (m *mockFieldBuilder) BuildTypeRef(schema *jsonschema.Schema, fieldName string) *TypeRef {
	m.buildTypeRefCalls++
	if m.returnTypeRef != nil {
		return m.returnTypeRef
	}
	return &TypeRef{Kind: KindPrimitive, GoType: "string"}
}

func (m *mockFieldBuilder) BuildFieldsFromProperties(schema *jsonschema.Schema, orderPath string) []*Field {
	return m.returnFields
}

func (m *mockFieldBuilder) MapPrimitiveType(schema *jsonschema.Schema) string { return "string" }

func TestStructBuilder_Build(t *testing.T) {
	t.Run("simple object", func(t *testing.T) {
		registry := NewTypeRegistry()
		sb := NewStructBuilder(registry, NewRefResolver(nil))
		mock := &mockFieldBuilder{}
		sb.SetFieldBuilder(mock)

		typ := &Type{ID: "1", Name: "Person"}
		err := sb.Build(typ, &jsonschema.Schema{
			Type: []string{"object"},
			Properties: &jsonschema.SchemaMap{
				"name": &jsonschema.Schema{Type: []string{"string"}},
				"age":  &jsonschema.Schema{Type: []string{"integer"}},
			},
			Required: []string{"name"},
		})

		require.NoError(t, err)
		assert.Equal(t, KindStruct, typ.Kind)
		assert.Len(t, typ.Fields, 2)
		assert.Equal(t, 2, mock.buildTypeRefCalls)
	})

	t.Run("allOf with ref", func(t *testing.T) {
		sb := NewStructBuilder(NewTypeRegistry(), NewRefResolver(jsonschema.NewCompiler()))
		sb.SetFieldBuilder(&mockFieldBuilder{})

		typ := &Type{ID: "1", Name: "ExtendedType"}
		err := sb.Build(typ, &jsonschema.Schema{
			AllOf: []*jsonschema.Schema{
				{Ref: "#/$defs/BaseType"},
				{Properties: &jsonschema.SchemaMap{"extra": &jsonschema.Schema{Type: []string{"string"}}}},
			},
		})

		require.NoError(t, err)
		assert.Contains(t, typ.Extends, "BaseType")
		assert.Len(t, typ.Fields, 1)
	})
}

func TestStructBuilder_DeduplicateFields(t *testing.T) {
	sb := NewStructBuilder(NewTypeRegistry(), NewRefResolver(nil))

	result := sb.DeduplicateFields([]*Field{
		{JSONName: "id", Type: &TypeRef{Kind: KindInterface, GoType: "interface{}"}},
		{JSONName: "id", Type: &TypeRef{Kind: KindPrimitive, GoType: "string"}},
		{JSONName: "name", Type: &TypeRef{Kind: KindPrimitive, GoType: "string"}},
	})

	assert.Len(t, result, 2)
	var idField *Field
	for _, f := range result {
		if f.JSONName == "id" {
			idField = f
			break
		}
	}
	assert.Equal(t, KindPrimitive, idField.Type.Kind)
}

func TestStructBuilder_ExtractConstraints(t *testing.T) {
	sb := NewStructBuilder(NewTypeRegistry(), NewRefResolver(nil))

	t.Run("string", func(t *testing.T) {
		minLen, maxLen, pattern := float64(5), float64(100), "^[a-z]+$"
		field := &Field{}
		sb.ExtractConstraints(field, &jsonschema.Schema{MinLength: &minLen, MaxLength: &maxLen, Pattern: &pattern})

		assert.Equal(t, 5, *field.MinLength)
		assert.Equal(t, 100, *field.MaxLength)
		assert.Equal(t, "^[a-z]+$", *field.Pattern)
	})

	t.Run("number", func(t *testing.T) {
		field := &Field{}
		sb.ExtractConstraints(field, &jsonschema.Schema{
			Minimum: jsonschema.NewRat(0),
			Maximum: jsonschema.NewRat(100),
		})

		assert.InDelta(t, 0.0, *field.Minimum, 0.0001)
		assert.InDelta(t, 100.0, *field.Maximum, 0.0001)
	})
}

func TestStructBuilder_ExtractAdditionalProperties(t *testing.T) {
	t.Run("boolean", func(t *testing.T) {
		sb := NewStructBuilder(NewTypeRegistry(), NewRefResolver(nil))
		sb.SetFieldBuilder(&mockFieldBuilder{})

		boolTrue := true
		config := sb.ExtractAdditionalProperties(&jsonschema.Schema{
			AdditionalProperties: &jsonschema.Schema{Boolean: &boolTrue},
		})

		assert.True(t, config.Allowed)
		assert.Nil(t, config.Type)
	})

	t.Run("typed", func(t *testing.T) {
		sb := NewStructBuilder(NewTypeRegistry(), NewRefResolver(nil))
		sb.SetFieldBuilder(&mockFieldBuilder{returnTypeRef: &TypeRef{Kind: KindPrimitive, GoType: "string"}})

		config := sb.ExtractAdditionalProperties(&jsonschema.Schema{
			AdditionalProperties: &jsonschema.Schema{Type: []string{"string"}},
		})

		assert.True(t, config.Allowed)
		assert.Equal(t, KindPrimitive, config.Type.Kind)
	})
}

func TestStructBuilder_GetOrderedPropertyNames(t *testing.T) {
	t.Run("with order", func(t *testing.T) {
		sb := NewStructBuilder(NewTypeRegistry(), NewRefResolver(nil))
		order, _ := schema.ExtractPropertyOrder([]byte(`{"properties": {"z": {}, "a": {}, "m": {}}}`), "test.json")
		sb.SetCurrentOrder(order)
		sb.SetCurrentPath("test.json")

		names := sb.GetOrderedPropertyNames(&jsonschema.SchemaMap{
			"z": &jsonschema.Schema{}, "a": &jsonschema.Schema{}, "m": &jsonschema.Schema{},
		}, "test.json")

		assert.Equal(t, []string{"z", "a", "m"}, names)
	})

	t.Run("alphabetical", func(t *testing.T) {
		sb := NewStructBuilder(NewTypeRegistry(), NewRefResolver(nil))
		names := sb.GetOrderedPropertyNames(&jsonschema.SchemaMap{
			"zebra": &jsonschema.Schema{}, "apple": &jsonschema.Schema{}, "mango": &jsonschema.Schema{},
		}, "")

		assert.Equal(t, []string{"apple", "mango", "zebra"}, names)
	})
}
