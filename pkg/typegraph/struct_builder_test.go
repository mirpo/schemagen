package typegraph

import (
	"testing"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/stretchr/testify/assert"
)

// mockFieldBuilder implements FieldBuilder for testing
type mockFieldBuilder struct {
	buildTypeRefCalls int
	lastFieldName     string
	returnTypeRef     *TypeRef
	buildFieldsCalls  int
	returnFields      []*Field
	mapPrimitiveCalls int
	returnPrimitive   string
}

func (m *mockFieldBuilder) BuildTypeRef(schema *jsonschema.Schema, fieldName string) *TypeRef {
	m.buildTypeRefCalls++
	m.lastFieldName = fieldName
	if m.returnTypeRef != nil {
		return m.returnTypeRef
	}
	return &TypeRef{Kind: KindPrimitive, GoType: "string"}
}

func (m *mockFieldBuilder) BuildFieldsFromProperties(schema *jsonschema.Schema, orderPath string) []*Field {
	m.buildFieldsCalls++
	return m.returnFields
}

func (m *mockFieldBuilder) MapPrimitiveType(schema *jsonschema.Schema) string {
	m.mapPrimitiveCalls++
	if m.returnPrimitive != "" {
		return m.returnPrimitive
	}
	return "string"
}

func TestStructBuilder_Build_SimpleObject(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	mock := &mockFieldBuilder{}

	sb := NewStructBuilder(registry, resolver)
	sb.SetFieldBuilder(mock)

	schema := &jsonschema.Schema{
		Type: []string{"object"},
		Properties: &jsonschema.SchemaMap{
			"name": &jsonschema.Schema{Type: []string{"string"}},
			"age":  &jsonschema.Schema{Type: []string{"integer"}},
		},
		Required: []string{"name"},
	}

	typ := &Type{ID: "1", Name: "Person"}
	err := sb.Build(typ, schema)

	assert.NoError(t, err)
	assert.Equal(t, KindStruct, typ.Kind)
	assert.Len(t, typ.Fields, 2)
	assert.Equal(t, 2, mock.buildTypeRefCalls)

	// Check required field
	var nameField *Field
	for _, f := range typ.Fields {
		if f.JSONName == "name" {
			nameField = f
			break
		}
	}
	assert.NotNil(t, nameField)
	assert.True(t, nameField.Required)
}

func TestStructBuilder_Build_AllOf_Ref(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	registry := NewTypeRegistry()
	resolver := NewRefResolver(compiler)
	mock := &mockFieldBuilder{}

	sb := NewStructBuilder(registry, resolver)
	sb.SetFieldBuilder(mock)

	// Schema with allOf containing a $ref
	schema := &jsonschema.Schema{
		AllOf: []*jsonschema.Schema{
			{Ref: "#/$defs/BaseType"},
			{
				Properties: &jsonschema.SchemaMap{
					"extra": &jsonschema.Schema{Type: []string{"string"}},
				},
			},
		},
	}

	typ := &Type{ID: "1", Name: "ExtendedType"}
	err := sb.Build(typ, schema)

	assert.NoError(t, err)
	assert.Equal(t, KindStruct, typ.Kind)
	assert.Contains(t, typ.Extends, "BaseType")
	assert.Len(t, typ.Fields, 1) // Only the "extra" field, not inherited ones
}

func TestStructBuilder_DeduplicateFields(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	sb := NewStructBuilder(registry, resolver)

	fields := []*Field{
		{JSONName: "id", Type: &TypeRef{Kind: KindInterface, GoType: "interface{}"}},
		{JSONName: "id", Type: &TypeRef{Kind: KindPrimitive, GoType: "string"}},
		{JSONName: "name", Type: &TypeRef{Kind: KindPrimitive, GoType: "string"}},
	}

	result := sb.DeduplicateFields(fields)

	assert.Len(t, result, 2)
	// Should keep the more specific type
	var idField *Field
	for _, f := range result {
		if f.JSONName == "id" {
			idField = f
			break
		}
	}
	assert.NotNil(t, idField)
	assert.Equal(t, KindPrimitive, idField.Type.Kind)
}

func TestStructBuilder_ExtractConstraints_String(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	sb := NewStructBuilder(registry, resolver)

	var minLen float64 = 5
	var maxLen float64 = 100
	pattern := "^[a-z]+$"

	schema := &jsonschema.Schema{
		MinLength: &minLen,
		MaxLength: &maxLen,
		Pattern:   &pattern,
	}

	field := &Field{}
	sb.ExtractConstraints(field, schema)

	assert.NotNil(t, field.MinLength)
	assert.Equal(t, 5, *field.MinLength)
	assert.NotNil(t, field.MaxLength)
	assert.Equal(t, 100, *field.MaxLength)
	assert.NotNil(t, field.Pattern)
	assert.Equal(t, "^[a-z]+$", *field.Pattern)
}

func TestStructBuilder_ExtractConstraints_Number(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	sb := NewStructBuilder(registry, resolver)

	min := jsonschema.NewRat(0)
	max := jsonschema.NewRat(100)

	schema := &jsonschema.Schema{
		Minimum: min,
		Maximum: max,
	}

	field := &Field{}
	sb.ExtractConstraints(field, schema)

	assert.NotNil(t, field.Minimum)
	assert.Equal(t, 0.0, *field.Minimum)
	assert.NotNil(t, field.Maximum)
	assert.Equal(t, 100.0, *field.Maximum)
}

func TestStructBuilder_ExtractAdditionalProperties_Boolean(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	mock := &mockFieldBuilder{}

	sb := NewStructBuilder(registry, resolver)
	sb.SetFieldBuilder(mock)

	boolTrue := true
	schema := &jsonschema.Schema{
		AdditionalProperties: &jsonschema.Schema{
			Boolean: &boolTrue,
		},
	}

	config := sb.ExtractAdditionalProperties(schema)

	assert.NotNil(t, config)
	assert.True(t, config.Allowed)
	assert.Nil(t, config.Type)
}

func TestStructBuilder_ExtractAdditionalProperties_Typed(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	mock := &mockFieldBuilder{
		returnTypeRef: &TypeRef{Kind: KindPrimitive, GoType: "string"},
	}

	sb := NewStructBuilder(registry, resolver)
	sb.SetFieldBuilder(mock)

	schema := &jsonschema.Schema{
		AdditionalProperties: &jsonschema.Schema{
			Type: []string{"string"},
		},
	}

	config := sb.ExtractAdditionalProperties(schema)

	assert.NotNil(t, config)
	assert.True(t, config.Allowed)
	assert.NotNil(t, config.Type)
	assert.Equal(t, KindPrimitive, config.Type.Kind)
}

func TestStructBuilder_GetOrderedPropertyNames_WithOrder(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	sb := NewStructBuilder(registry, resolver)

	// Set up property order using ExtractPropertyOrder
	jsonData := []byte(`{"properties": {"z_first": {}, "a_second": {}, "m_third": {}}}`)
	order, err := schema.ExtractPropertyOrder(jsonData, "test.json")
	assert.NoError(t, err)

	sb.SetCurrentOrder(order)
	sb.SetCurrentPath("test.json")

	properties := &jsonschema.SchemaMap{
		"z_first":  &jsonschema.Schema{},
		"a_second": &jsonschema.Schema{},
		"m_third":  &jsonschema.Schema{},
	}

	names := sb.GetOrderedPropertyNames(properties, "test.json")

	assert.Equal(t, []string{"z_first", "a_second", "m_third"}, names)
}

func TestStructBuilder_GetOrderedPropertyNames_Alphabetical(t *testing.T) {
	registry := NewTypeRegistry()
	resolver := NewRefResolver(nil)
	sb := NewStructBuilder(registry, resolver)

	properties := &jsonschema.SchemaMap{
		"zebra": &jsonschema.Schema{},
		"apple": &jsonschema.Schema{},
		"mango": &jsonschema.Schema{},
	}

	names := sb.GetOrderedPropertyNames(properties, "")

	// Without order info, should be alphabetical
	assert.Equal(t, []string{"apple", "mango", "zebra"}, names)
}
