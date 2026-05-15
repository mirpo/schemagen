package typegraph

import (
	"testing"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTypeBuilder struct {
	buildStructCalled bool
	buildEnumCalled   bool
	buildUnionCalled  bool
}

func (m *mockTypeBuilder) BuildStruct(_ *BuildContext, typ *Type, s *jsonschema.Schema) error {
	m.buildStructCalled = true
	typ.Kind = KindStruct
	return nil
}

func (m *mockTypeBuilder) BuildEnum(typ *Type, s *jsonschema.Schema) error {
	m.buildEnumCalled = true
	typ.Kind = KindEnum
	return nil
}

func (m *mockTypeBuilder) BuildUnion(_ *BuildContext, typ *Type, s *jsonschema.Schema) error {
	m.buildUnionCalled = true
	typ.Kind = KindUnion
	return nil
}

func TestSchemaWalker_Process(t *testing.T) {
	tests := []struct {
		name       string
		compiled   *jsonschema.Schema
		wantStruct bool
		wantEnum   bool
		wantUnion  bool
		wantKind   TypeKind
	}{
		{
			name: "object",
			compiled: &jsonschema.Schema{
				Type:       []string{"object"},
				Properties: &jsonschema.SchemaMap{"name": &jsonschema.Schema{Type: []string{"string"}}},
			},
			wantStruct: true,
			wantKind:   KindStruct,
		},
		{
			name:     "enum",
			compiled: &jsonschema.Schema{Enum: []any{"a", "b", "c"}},
			wantEnum: true,
			wantKind: KindEnum,
		},
		{
			name: "union",
			compiled: &jsonschema.Schema{
				AnyOf: []*jsonschema.Schema{{Type: []string{"string"}}, {Type: []string{"number"}}},
			},
			wantUnion: true,
			wantKind:  KindUnion,
		},
		{
			name:     "primitive",
			compiled: &jsonschema.Schema{Type: []string{"string"}},
			wantKind: KindPrimitive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewTypeRegistry()
			mock := &mockTypeBuilder{}
			walker := NewSchemaWalker(registry, NewRefResolver(jsonschema.NewCompiler()), mock, nil)

			err := walker.Process(&schema.Schema{Name: "Test", Compiled: tt.compiled})

			require.NoError(t, err)
			assert.Equal(t, tt.wantStruct, mock.buildStructCalled)
			assert.Equal(t, tt.wantEnum, mock.buildEnumCalled)
			assert.Equal(t, tt.wantUnion, mock.buildUnionCalled)
			assert.Len(t, registry.All(), 1)
			assert.Equal(t, tt.wantKind, registry.All()[0].Kind)
		})
	}
}

func TestSchemaWalker_ExtractDefs(t *testing.T) {
	registry := NewTypeRegistry()
	walker := NewSchemaWalker(registry, NewRefResolver(jsonschema.NewCompiler()), &mockTypeBuilder{}, nil)

	err := walker.Process(&schema.Schema{
		Name: "Root",
		Compiled: &jsonschema.Schema{
			Type:       []string{"object"},
			Properties: &jsonschema.SchemaMap{"id": &jsonschema.Schema{Type: []string{"string"}}},
			Defs: map[string]*jsonschema.Schema{
				"SubType": {
					Type:       []string{"object"},
					Properties: &jsonschema.SchemaMap{"value": &jsonschema.Schema{Type: []string{"string"}}},
				},
			},
		},
	})

	require.NoError(t, err)
	assert.Len(t, registry.All(), 2)
	assert.Equal(t, "SubType", registry.All()[0].Name)
	assert.Equal(t, "Root", registry.All()[1].Name)
}

func TestSchemaWalker_ExtractDefs_Union(t *testing.T) {
	registry := NewTypeRegistry()
	mock := &mockTypeBuilder{}
	walker := NewSchemaWalker(registry, NewRefResolver(jsonschema.NewCompiler()), mock, nil)

	err := walker.Process(&schema.Schema{
		Name: "Root",
		Compiled: &jsonschema.Schema{
			Type:       []string{"object"},
			Properties: &jsonschema.SchemaMap{"id": &jsonschema.Schema{Type: []string{"string"}}},
			Defs: map[string]*jsonschema.Schema{
				"Payload": {
					OneOf: []*jsonschema.Schema{
						{Type: []string{"string"}},
						{Type: []string{"integer"}},
					},
				},
			},
		},
	})

	require.NoError(t, err)
	assert.Len(t, registry.All(), 2)

	payload := registry.All()[0]
	assert.Equal(t, "Payload", payload.Name)
	assert.Equal(t, KindUnion, payload.Kind, "union $def should be KindUnion, not KindPrimitive")
	assert.True(t, mock.buildUnionCalled, "BuildUnion should be called for union $defs")
}
