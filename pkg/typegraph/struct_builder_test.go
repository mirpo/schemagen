package typegraph

import (
	"testing"

	"github.com/kaptinlin/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFieldBuilder struct {
	buildTypeRefCalls int
	returnTypeRef     *TypeRef
	returnFields      []*Field
}

func (m *mockFieldBuilder) BuildTypeRef(_ *buildContext, schema *jsonschema.Schema, fieldName string) *TypeRef {
	m.buildTypeRefCalls++
	if m.returnTypeRef != nil {
		return m.returnTypeRef
	}
	return &TypeRef{Kind: KindPrimitive, Primitive: PrimString}
}

func (m *mockFieldBuilder) buildFieldsFromProperties(_ *buildContext, schema *jsonschema.Schema, orderPath string) []*Field {
	return m.returnFields
}

func TestStructBuilder_Build(t *testing.T) {
	t.Run("simple object", func(t *testing.T) {
		registry := newTypeRegistry()
		sb := newStructBuilder(registry, newRefResolver(nil))
		mock := &mockFieldBuilder{}
		sb.setFieldBuilder(mock)

		typ := &Type{Name: "Person"}
		err := sb.Build(&buildContext{}, typ, &jsonschema.Schema{
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
		sb := newStructBuilder(newTypeRegistry(), newRefResolver(jsonschema.NewCompiler()))
		sb.setFieldBuilder(&mockFieldBuilder{})

		typ := &Type{Name: "ExtendedType"}
		err := sb.Build(&buildContext{}, typ, &jsonschema.Schema{
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
	sb := newStructBuilder(newTypeRegistry(), newRefResolver(nil))

	result := sb.deduplicateFields([]*Field{
		{JSONName: "id", Type: &TypeRef{Kind: KindInterface, Primitive: PrimUnknown}},
		{JSONName: "id", Type: &TypeRef{Kind: KindPrimitive, Primitive: PrimString}},
		{JSONName: "name", Type: &TypeRef{Kind: KindPrimitive, Primitive: PrimString}},
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
