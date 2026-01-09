package typegraph

import (
	"testing"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTypeBuilder implements TypeBuilder interface for testing
type mockTypeBuilder struct {
	buildStructCalled bool
	buildEnumCalled   bool
	buildUnionCalled  bool
	lastType          *Type
	lastSchema        *jsonschema.Schema
}

func (m *mockTypeBuilder) BuildStruct(typ *Type, s *jsonschema.Schema) error {
	m.buildStructCalled = true
	m.lastType = typ
	m.lastSchema = s
	typ.Kind = KindStruct
	return nil
}

func (m *mockTypeBuilder) BuildEnum(typ *Type, s *jsonschema.Schema) error {
	m.buildEnumCalled = true
	m.lastType = typ
	m.lastSchema = s
	typ.Kind = KindEnum
	return nil
}

func (m *mockTypeBuilder) BuildUnion(typ *Type, s *jsonschema.Schema) error {
	m.buildUnionCalled = true
	m.lastType = typ
	m.lastSchema = s
	typ.Kind = KindUnion
	return nil
}

func (m *mockTypeBuilder) MapPrimitiveType(s *jsonschema.Schema) string {
	return "string"
}

func TestSchemaWalker_Process_Object(t *testing.T) {
	registry := NewTypeRegistry()
	compiler := jsonschema.NewCompiler()
	resolver := NewRefResolver(compiler)
	mock := &mockTypeBuilder{}

	walker := NewSchemaWalker(registry, resolver, mock, nil)

	compiled := &jsonschema.Schema{
		Type: []string{"object"},
		Properties: &jsonschema.SchemaMap{
			"name": &jsonschema.Schema{Type: []string{"string"}},
		},
	}

	s := &schema.Schema{
		Name:     "TestObject",
		Compiled: compiled,
	}

	err := walker.Process(s)

	require.NoError(t, err)
	assert.True(t, mock.buildStructCalled)
	assert.Len(t, registry.All(), 1)
	assert.Equal(t, "TestObject", registry.All()[0].Name)
}

func TestSchemaWalker_Process_Enum(t *testing.T) {
	registry := NewTypeRegistry()
	compiler := jsonschema.NewCompiler()
	resolver := NewRefResolver(compiler)
	mock := &mockTypeBuilder{}

	walker := NewSchemaWalker(registry, resolver, mock, nil)

	compiled := &jsonschema.Schema{
		Enum: []any{"active", "inactive", "pending"},
	}

	s := &schema.Schema{
		Name:     "Status",
		Compiled: compiled,
	}

	err := walker.Process(s)

	require.NoError(t, err)
	assert.True(t, mock.buildEnumCalled)
	assert.Len(t, registry.All(), 1)
	assert.Equal(t, "Status", registry.All()[0].Name)
}

func TestSchemaWalker_Process_Union(t *testing.T) {
	registry := NewTypeRegistry()
	compiler := jsonschema.NewCompiler()
	resolver := NewRefResolver(compiler)
	mock := &mockTypeBuilder{}

	walker := NewSchemaWalker(registry, resolver, mock, nil)

	compiled := &jsonschema.Schema{
		AnyOf: []*jsonschema.Schema{
			{Type: []string{"string"}},
			{Type: []string{"number"}},
		},
	}

	s := &schema.Schema{
		Name:     "StringOrNumber",
		Compiled: compiled,
	}

	err := walker.Process(s)

	require.NoError(t, err)
	assert.True(t, mock.buildUnionCalled)
	assert.Len(t, registry.All(), 1)
	assert.Equal(t, "StringOrNumber", registry.All()[0].Name)
}

func TestSchemaWalker_Process_Primitive(t *testing.T) {
	registry := NewTypeRegistry()
	compiler := jsonschema.NewCompiler()
	resolver := NewRefResolver(compiler)
	mock := &mockTypeBuilder{}

	walker := NewSchemaWalker(registry, resolver, mock, nil)

	compiled := &jsonschema.Schema{
		Type: []string{"string"},
	}

	s := &schema.Schema{
		Name:     "MyString",
		Compiled: compiled,
	}

	err := walker.Process(s)

	require.NoError(t, err)
	assert.False(t, mock.buildStructCalled)
	assert.False(t, mock.buildEnumCalled)
	assert.False(t, mock.buildUnionCalled)
	assert.Len(t, registry.All(), 1)
	assert.Equal(t, "MyString", registry.All()[0].Name)
	assert.Equal(t, KindPrimitive, registry.All()[0].Kind)
}

func TestSchemaWalker_ExtractDefs(t *testing.T) {
	registry := NewTypeRegistry()
	compiler := jsonschema.NewCompiler()
	resolver := NewRefResolver(compiler)
	mock := &mockTypeBuilder{}

	walker := NewSchemaWalker(registry, resolver, mock, nil)

	compiled := &jsonschema.Schema{
		Type: []string{"object"},
		Properties: &jsonschema.SchemaMap{
			"id": &jsonschema.Schema{Type: []string{"string"}},
		},
		Defs: map[string]*jsonschema.Schema{
			"SubType": {
				Type: []string{"object"},
				Properties: &jsonschema.SchemaMap{
					"value": &jsonschema.Schema{Type: []string{"string"}},
				},
			},
		},
	}

	s := &schema.Schema{
		Name:     "Root",
		Compiled: compiled,
	}

	err := walker.Process(s)

	require.NoError(t, err)
	// Should have SubType + Root
	assert.Len(t, registry.All(), 2)

	// First should be SubType (processed from $defs)
	assert.Equal(t, "SubType", registry.All()[0].Name)
	// Second should be Root
	assert.Equal(t, "Root", registry.All()[1].Name)
}
