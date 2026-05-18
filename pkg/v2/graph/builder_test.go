package graph

import (
	"strings"
	"testing"

	"github.com/mirpo/schemagen/pkg/v2/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustParse(t *testing.T, json string) *parse.SchemaNode {
	t.Helper()
	node, err := parse.ParseJSON(strings.NewReader(json))
	require.NoError(t, err)
	return node
}

func mustParseNamed(t *testing.T, name, json string) *parse.NamedSchema {
	t.Helper()
	return &parse.NamedSchema{
		Name:   name,
		Schema: mustParse(t, json),
	}
}

func buildOne(t *testing.T, name, json string, cfg BuildConfig) *Graph {
	t.Helper()
	ns := mustParseNamed(t, name, json)
	g, err := Build([]*parse.NamedSchema{ns}, cfg)
	require.NoError(t, err)
	return g
}

func typeMap(g *Graph) map[string]*Type {
	m := make(map[string]*Type)
	for _, t := range g.Types {
		m[t.Name] = t
	}
	return m
}

func fieldMap(t *Type) map[string]*Field {
	m := make(map[string]*Field)
	for _, f := range t.Fields {
		m[f.JSONName] = f
	}
	return m
}

// ==================== Primitive Mapping ====================

func TestMapPrimitive(t *testing.T) {
	tests := []struct {
		name     string
		typ      string
		format   string
		expected PrimitiveKind
	}{
		{"string", "string", "", PrimString},
		{"integer", "integer", "", PrimInt},
		{"number", "number", "", PrimFloat64},
		{"boolean", "boolean", "", PrimBool},
		{"string+email", "string", "email", PrimEmail},
		{"string+uuid", "string", "uuid", PrimUUID},
		{"string+date-time", "string", "date-time", PrimDateTime},
		{"string+date", "string", "date", PrimDate},
		{"string+time", "string", "time", PrimTime},
		{"string+uri", "string", "uri", PrimURI},
		{"string+url", "string", "url", PrimURI},
		{"string+hostname", "string", "hostname", PrimHostname},
		{"string+ipv4", "string", "ipv4", PrimIPv4},
		{"string+ipv6", "string", "ipv6", PrimIPv6},
		{"number+int32", "number", "int32", PrimInt32},
		{"number+int64", "number", "int64", PrimInt64},
		{"number+float", "number", "float", PrimFloat32},
		{"number+double", "number", "double", PrimFloat64},
		{"unknown type", "unknown", "", PrimUnknown},
		{"empty type", "", "", PrimUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, mapPrimitive(tt.typ, tt.format))
		})
	}
}

// ==================== Enum Helpers ====================

func TestInferEnumType(t *testing.T) {
	tests := []struct {
		name     string
		values   []any
		expected string
	}{
		{"all strings", []any{"a", "b", "c"}, "string"},
		{"all ints", []any{int64(1), int64(2), int64(3)}, "int"},
		{"all floats", []any{1.5, 2.5}, "int"},
		{"mixed string and number", []any{"a", int64(1)}, "mixed"},
		{"mixed with bool", []any{"a", true}, "mixed"},
		{"mixed with nil", []any{"a", nil}, "mixed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, inferEnumType(tt.values))
		})
	}
}

func TestEnumValueName(t *testing.T) {
	assert.Equal(t, "hello", enumValueName("hello"))
	assert.Equal(t, "42", enumValueName(42))
	assert.Equal(t, "true", enumValueName(true))
}

// ==================== Simple Struct ====================

func TestBuild_SimpleStruct(t *testing.T) {
	g := buildOne(t, "User", `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		}
	}`, BuildConfig{})

	require.Len(t, g.Types, 1)
	typ := g.Types[0]
	assert.Equal(t, "User", typ.Name)
	assert.Equal(t, KindStruct, typ.Kind)
	assert.Len(t, typ.Fields, 2)
}

// ==================== Field Order ====================

func TestBuild_FieldOrder(t *testing.T) {
	g := buildOne(t, "Order", `{
		"type": "object",
		"properties": {
			"alpha": {"type": "string"},
			"beta": {"type": "integer"},
			"gamma": {"type": "boolean"}
		}
	}`, BuildConfig{})

	require.Len(t, g.Types, 1)
	typ := g.Types[0]
	require.Len(t, typ.Fields, 3)
	assert.Equal(t, "alpha", typ.Fields[0].JSONName)
	assert.Equal(t, "beta", typ.Fields[1].JSONName)
	assert.Equal(t, "gamma", typ.Fields[2].JSONName)
}

// ==================== Required Fields ====================

func TestBuild_RequiredFields(t *testing.T) {
	g := buildOne(t, "User", `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"},
			"email": {"type": "string"}
		},
		"required": ["name", "email"]
	}`, BuildConfig{})

	require.Len(t, g.Types, 1)
	fm := fieldMap(g.Types[0])
	assert.True(t, fm["name"].Required)
	assert.False(t, fm["age"].Required)
	assert.True(t, fm["email"].Required)
}

// ==================== Constraints ====================

func TestBuild_Constraints(t *testing.T) {
	g := buildOne(t, "Product", `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"minLength": 3,
				"maxLength": 100,
				"pattern": "^[A-Za-z]+$"
			},
			"price": {
				"type": "number",
				"minimum": 0.01,
				"maximum": 9999.99
			},
			"count": {
				"type": "integer",
				"exclusiveMinimum": 0,
				"exclusiveMaximum": 1000
			},
			"tags": {
				"type": "array",
				"items": {"type": "string"},
				"minItems": 1,
				"maxItems": 10
			}
		}
	}`, BuildConfig{})

	fm := fieldMap(g.Types[0])

	nameF := fm["name"]
	require.NotNil(t, nameF.MinLength)
	assert.Equal(t, 3, *nameF.MinLength)
	require.NotNil(t, nameF.MaxLength)
	assert.Equal(t, 100, *nameF.MaxLength)
	require.NotNil(t, nameF.Pattern)
	assert.Equal(t, "^[A-Za-z]+$", *nameF.Pattern)

	priceF := fm["price"]
	require.NotNil(t, priceF.Minimum)
	assert.InDelta(t, 0.01, *priceF.Minimum, 0.001)
	require.NotNil(t, priceF.Maximum)
	assert.InDelta(t, 9999.99, *priceF.Maximum, 0.001)

	countF := fm["count"]
	require.NotNil(t, countF.ExclusiveMinimum)
	assert.InDelta(t, 0.0, *countF.ExclusiveMinimum, 0.001)
	require.NotNil(t, countF.ExclusiveMaximum)
	assert.InDelta(t, 1000.0, *countF.ExclusiveMaximum, 0.001)

	tagsF := fm["tags"]
	require.NotNil(t, tagsF.MinItems)
	assert.Equal(t, 1, *tagsF.MinItems)
	require.NotNil(t, tagsF.MaxItems)
	assert.Equal(t, 10, *tagsF.MaxItems)
}

func TestField_HasConstraints(t *testing.T) {
	minLen := 1
	assert.False(t, (&Field{}).HasConstraints())
	assert.True(t, (&Field{MinLength: &minLen}).HasConstraints())
}

// ==================== Enum ====================

func TestBuild_StringEnum(t *testing.T) {
	g := buildOne(t, "Status", `{
		"type": "string",
		"enum": ["active", "inactive", "pending"]
	}`, BuildConfig{})

	require.Len(t, g.Types, 1)
	typ := g.Types[0]
	assert.Equal(t, KindEnum, typ.Kind)
	assert.Equal(t, "string", typ.EnumType)
	require.Len(t, typ.EnumValues, 3)
	assert.Equal(t, "ACTIVE", typ.EnumValues[0].Name)
	assert.Equal(t, "active", typ.EnumValues[0].Value)
}

func TestBuild_NumberEnum(t *testing.T) {
	g := buildOne(t, "Priority", `{
		"type": "number",
		"enum": [1, 2, 3]
	}`, BuildConfig{})

	require.Len(t, g.Types, 1)
	typ := g.Types[0]
	assert.Equal(t, KindEnum, typ.Kind)
	assert.Equal(t, "int", typ.EnumType)
	require.Len(t, typ.EnumValues, 3)
}

func TestBuild_MixedEnum(t *testing.T) {
	g := buildOne(t, "Mixed", `{
		"enum": ["active", 1, true, null]
	}`, BuildConfig{})

	require.Len(t, g.Types, 1)
	typ := g.Types[0]
	assert.Equal(t, KindEnum, typ.Kind)
	assert.Equal(t, "mixed", typ.EnumType)
	assert.Len(t, typ.EnumValues, 4)
}

// ==================== Union (anyOf) ====================

func TestBuild_AnyOfPrimitives(t *testing.T) {
	g := buildOne(t, "FlexValue", `{
		"anyOf": [
			{"type": "string"},
			{"type": "integer"}
		]
	}`, BuildConfig{})

	require.Len(t, g.Types, 1)
	typ := g.Types[0]
	assert.Equal(t, KindUnion, typ.Kind)
	require.Len(t, typ.UnionMembers, 2)
	assert.Equal(t, KindPrimitive, typ.UnionMembers[0].Kind)
	assert.Equal(t, PrimString, typ.UnionMembers[0].Primitive)
	assert.Equal(t, KindPrimitive, typ.UnionMembers[1].Kind)
	assert.Equal(t, PrimInt, typ.UnionMembers[1].Primitive)
}

func TestBuild_AnyOfObjects(t *testing.T) {
	g := buildOne(t, "Response", `{
		"anyOf": [
			{
				"type": "object",
				"title": "SuccessResponse",
				"properties": {
					"data": {"type": "string"}
				}
			},
			{
				"type": "object",
				"title": "ErrorResponse",
				"properties": {
					"error": {"type": "string"}
				}
			}
		]
	}`, BuildConfig{})

	tm := typeMap(g)
	assert.Contains(t, tm, "Response")
	assert.Equal(t, KindUnion, tm["Response"].Kind)

	require.Len(t, tm["Response"].UnionMembers, 2)
	member0 := tm["Response"].UnionMembers[0]
	member1 := tm["Response"].UnionMembers[1]
	assert.Equal(t, KindInterface, member0.Kind)
	assert.Equal(t, KindInterface, member1.Kind)
	require.NotEmpty(t, member0.ObjectFields)
	require.NotEmpty(t, member1.ObjectFields)
}

// ==================== Union (oneOf) ====================

func TestBuild_OneOfPrimitives(t *testing.T) {
	g := buildOne(t, "Variant", `{
		"oneOf": [
			{"type": "string"},
			{"type": "number"}
		]
	}`, BuildConfig{})

	require.Len(t, g.Types, 1)
	typ := g.Types[0]
	assert.Equal(t, KindUnion, typ.Kind)
	assert.Len(t, typ.UnionMembers, 2)
}

func TestBuild_OneOfObjects(t *testing.T) {
	g := buildOne(t, "Shape", `{
		"oneOf": [
			{
				"type": "object",
				"title": "Circle",
				"properties": {"radius": {"type": "number"}}
			},
			{
				"type": "object",
				"title": "Square",
				"properties": {"side": {"type": "number"}}
			}
		]
	}`, BuildConfig{})

	tm := typeMap(g)
	assert.Contains(t, tm, "Shape")
	assert.Equal(t, KindUnion, tm["Shape"].Kind)
	require.Len(t, tm["Shape"].UnionMembers, 2)
	assert.Equal(t, KindInterface, tm["Shape"].UnionMembers[0].Kind)
	assert.Equal(t, KindInterface, tm["Shape"].UnionMembers[1].Kind)
}

// ==================== AllOf Composition ====================

func TestBuild_AllOfRefs(t *testing.T) {
	g := buildOne(t, "Document", `{
		"$defs": {
			"Entity": {
				"type": "object",
				"properties": {
					"id": {"type": "string"}
				}
			},
			"Auditable": {
				"type": "object",
				"properties": {
					"createdBy": {"type": "string"}
				}
			}
		},
		"allOf": [
			{"$ref": "#/$defs/Entity"},
			{"$ref": "#/$defs/Auditable"},
			{
				"type": "object",
				"properties": {
					"title": {"type": "string"}
				},
				"required": ["title"]
			}
		]
	}`, BuildConfig{})

	tm := typeMap(g)
	require.Contains(t, tm, "Document")
	doc := tm["Document"]
	assert.Equal(t, KindStruct, doc.Kind)
	assert.Contains(t, doc.Extends, "Entity")
	assert.Contains(t, doc.Extends, "Auditable")
	require.Len(t, doc.Fields, 1)
	assert.Equal(t, "title", doc.Fields[0].JSONName)
	assert.True(t, doc.Fields[0].Required)
}

func TestBuild_AllOfInlineOnly(t *testing.T) {
	g := buildOne(t, "Vehicle", `{
		"allOf": [
			{
				"type": "object",
				"properties": {
					"make": {"type": "string"},
					"model": {"type": "string"}
				},
				"required": ["make"]
			},
			{
				"type": "object",
				"properties": {
					"year": {"type": "integer"}
				}
			}
		]
	}`, BuildConfig{})

	require.Len(t, g.Types, 1)
	typ := g.Types[0]
	assert.Equal(t, KindStruct, typ.Kind)
	assert.Len(t, typ.Fields, 3)
	assert.Empty(t, typ.Extends)
}

func TestBuild_AllOfFieldDedup(t *testing.T) {
	g := buildOne(t, "Merged", `{
		"allOf": [
			{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "first name def"}
				}
			},
			{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "override name def"},
					"extra": {"type": "integer"}
				}
			}
		]
	}`, BuildConfig{})

	require.Len(t, g.Types, 1)
	typ := g.Types[0]
	assert.Len(t, typ.Fields, 2)

	fm := fieldMap(typ)
	assert.Equal(t, "first name def", fm["name"].Description)
	assert.NotNil(t, fm["extra"])
}

// ==================== $ref Resolution ====================

func TestBuild_RefResolution(t *testing.T) {
	g := buildOne(t, "User", `{
		"type": "object",
		"properties": {
			"address": {"$ref": "#/$defs/Address"}
		},
		"$defs": {
			"Address": {
				"type": "object",
				"properties": {
					"street": {"type": "string"},
					"city": {"type": "string"}
				}
			}
		}
	}`, BuildConfig{})

	tm := typeMap(g)
	require.Contains(t, tm, "User")
	require.Contains(t, tm, "Address")

	addrField := fieldMap(tm["User"])["address"]
	require.NotNil(t, addrField)
	assert.Equal(t, KindRef, addrField.Type.Kind)
	assert.Equal(t, "Address", addrField.Type.TypeName)
}

// ==================== Self-Reference ====================

func TestBuild_SelfReference(t *testing.T) {
	g := buildOne(t, "CyclicRef", `{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"parent": {"$ref": "#"},
			"children": {
				"type": "array",
				"items": {"$ref": "#"}
			}
		}
	}`, BuildConfig{})

	require.Len(t, g.Types, 1)
	typ := g.Types[0]
	assert.Equal(t, "CyclicRef", typ.Name)

	fm := fieldMap(typ)
	parentF := fm["parent"]
	require.NotNil(t, parentF)
	assert.Equal(t, KindRef, parentF.Type.Kind)
	assert.Equal(t, "CyclicRef", parentF.Type.TypeName)

	childrenF := fm["children"]
	require.NotNil(t, childrenF)
	assert.Equal(t, KindArray, childrenF.Type.Kind)
	assert.Equal(t, KindRef, childrenF.Type.ItemType.Kind)
	assert.Equal(t, "CyclicRef", childrenF.Type.ItemType.TypeName)
}

// ==================== Inline Extraction (enums) ====================

func TestBuild_InlineExtraction_Enum(t *testing.T) {
	g := buildOne(t, "BlogPost", `{
		"type": "object",
		"properties": {
			"status": {
				"type": "string",
				"enum": ["draft", "published"]
			}
		}
	}`, BuildConfig{ExtractInlined: true})

	tm := typeMap(g)
	require.Contains(t, tm, "BlogPost")
	require.Contains(t, tm, "Status")
	assert.Equal(t, KindEnum, tm["Status"].Kind)

	statusField := fieldMap(tm["BlogPost"])["status"]
	assert.Equal(t, KindRef, statusField.Type.Kind)
	assert.Equal(t, "Status", statusField.Type.TypeName)
}

func TestBuild_InlineExtraction_Disabled(t *testing.T) {
	g := buildOne(t, "BlogPost", `{
		"type": "object",
		"properties": {
			"status": {
				"type": "string",
				"enum": ["draft", "published"]
			}
		}
	}`, BuildConfig{ExtractInlined: false})

	require.Len(t, g.Types, 1)
	statusField := fieldMap(g.Types[0])["status"]
	assert.Equal(t, KindEnum, statusField.Type.Kind)
	assert.Len(t, statusField.Type.EnumValues, 2)
}

// ==================== Inline Extraction (objects) ====================

func TestBuild_InlineExtraction_Object(t *testing.T) {
	g := buildOne(t, "BlogPost", `{
		"type": "object",
		"properties": {
			"author": {
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"email": {"type": "string"}
				}
			}
		}
	}`, BuildConfig{ExtractInlined: true})

	tm := typeMap(g)
	require.Contains(t, tm, "BlogPost")
	require.Contains(t, tm, "Author")
	assert.Equal(t, KindStruct, tm["Author"].Kind)
	assert.Len(t, tm["Author"].Fields, 2)

	authorField := fieldMap(tm["BlogPost"])["author"]
	assert.Equal(t, KindRef, authorField.Type.Kind)
	assert.Equal(t, "Author", authorField.Type.TypeName)
}

func TestBuild_InlineExtraction_ObjectDisabled(t *testing.T) {
	g := buildOne(t, "BlogPost", `{
		"type": "object",
		"properties": {
			"author": {
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"email": {"type": "string"}
				}
			}
		}
	}`, BuildConfig{ExtractInlined: false})

	require.Len(t, g.Types, 1)
	authorField := fieldMap(g.Types[0])["author"]
	assert.Equal(t, KindInterface, authorField.Type.Kind)
	assert.Len(t, authorField.Type.ObjectFields, 2)
}

// ==================== AdditionalProperties ====================

func TestBuild_AdditionalProperties_BooleanFalse(t *testing.T) {
	g := buildOne(t, "Strict", `{
		"type": "object",
		"properties": {
			"id": {"type": "string"}
		},
		"additionalProperties": false
	}`, BuildConfig{})

	typ := g.Types[0]
	require.NotNil(t, typ.AdditionalProps)
	assert.False(t, typ.AdditionalProps.Allowed)
	assert.Nil(t, typ.AdditionalProps.Type)
}

func TestBuild_AdditionalProperties_BooleanTrue(t *testing.T) {
	g := buildOne(t, "Flexible", `{
		"type": "object",
		"properties": {
			"id": {"type": "string"}
		},
		"additionalProperties": true
	}`, BuildConfig{})

	typ := g.Types[0]
	require.NotNil(t, typ.AdditionalProps)
	assert.True(t, typ.AdditionalProps.Allowed)
	assert.Nil(t, typ.AdditionalProps.Type)
}

func TestBuild_AdditionalProperties_Schema(t *testing.T) {
	g := buildOne(t, "TypedMap", `{
		"type": "object",
		"properties": {
			"id": {"type": "string"}
		},
		"additionalProperties": {"type": "number"}
	}`, BuildConfig{})

	typ := g.Types[0]
	require.NotNil(t, typ.AdditionalProps)
	assert.True(t, typ.AdditionalProps.Allowed)
	require.NotNil(t, typ.AdditionalProps.Type)
	assert.Equal(t, KindPrimitive, typ.AdditionalProps.Type.Kind)
	assert.Equal(t, PrimFloat64, typ.AdditionalProps.Type.Primitive)
}

// ==================== Array Type ====================

func TestBuild_ArrayType(t *testing.T) {
	g := buildOne(t, "List", `{
		"type": "object",
		"properties": {
			"items": {
				"type": "array",
				"items": {"type": "string"}
			}
		}
	}`, BuildConfig{})

	itemsField := fieldMap(g.Types[0])["items"]
	require.NotNil(t, itemsField)
	assert.Equal(t, KindArray, itemsField.Type.Kind)
	assert.NotNil(t, itemsField.Type.ItemType)
	assert.Equal(t, KindPrimitive, itemsField.Type.ItemType.Kind)
	assert.Equal(t, PrimString, itemsField.Type.ItemType.Primitive)
}

func TestBuild_ArrayOfRefs(t *testing.T) {
	g := buildOne(t, "Container", `{
		"type": "object",
		"properties": {
			"items": {
				"type": "array",
				"items": {"$ref": "#/$defs/Item"}
			}
		},
		"$defs": {
			"Item": {
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				}
			}
		}
	}`, BuildConfig{})

	fm := fieldMap(typeMap(g)["Container"])
	assert.Equal(t, KindArray, fm["items"].Type.Kind)
	assert.Equal(t, KindRef, fm["items"].Type.ItemType.Kind)
	assert.Equal(t, "Item", fm["items"].Type.ItemType.TypeName)
}

// ==================== Map Type ====================

func TestBuild_MapType(t *testing.T) {
	g := buildOne(t, "Config", `{
		"type": "object",
		"properties": {
			"metadata": {
				"type": "object",
				"additionalProperties": {"type": "string"}
			}
		}
	}`, BuildConfig{})

	metaField := fieldMap(g.Types[0])["metadata"]
	assert.Equal(t, KindMap, metaField.Type.Kind)
	require.NotNil(t, metaField.Type.ValueType)
	assert.Equal(t, KindPrimitive, metaField.Type.ValueType.Kind)
	assert.Equal(t, PrimString, metaField.Type.ValueType.Primitive)
}

// ==================== Type Uniqueness ====================

func TestBuild_TypeUniqueness(t *testing.T) {
	b := &builder{
		graph:          NewGraph(),
		cfg:            BuildConfig{},
		globalDefs:     make(map[string]*parse.SchemaNode),
		processedTypes: make(map[string]bool),
		schemaNames:    make(map[string]string),
	}
	b.graph.AddType(&Type{Name: "Status"})
	b.processedTypes["Status"] = true

	result := b.ensureUniqueName("Status")
	assert.NotEqual(t, "Status", result)
	assert.Equal(t, "Status2", result)
}

func TestBuild_TypeUniqueness_Multiple(t *testing.T) {
	b := &builder{
		graph:          NewGraph(),
		cfg:            BuildConfig{},
		globalDefs:     make(map[string]*parse.SchemaNode),
		processedTypes: make(map[string]bool),
		schemaNames:    make(map[string]string),
	}
	b.graph.AddType(&Type{Name: "Status"})
	b.processedTypes["Status"] = true
	b.graph.AddType(&Type{Name: "Status1"})
	b.processedTypes["Status1"] = true

	result := b.ensureUniqueName("Status")
	assert.Equal(t, "Status2", result)
}

// ==================== Multiple Schemas ====================

func TestBuild_MultipleSchemas(t *testing.T) {
	ns1 := mustParseNamed(t, "User", `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		}
	}`)
	ns2 := mustParseNamed(t, "Product", `{
		"type": "object",
		"properties": {
			"title": {"type": "string"},
			"price": {"type": "number"}
		}
	}`)

	g, err := Build([]*parse.NamedSchema{ns1, ns2}, BuildConfig{})
	require.NoError(t, err)

	tm := typeMap(g)
	require.Contains(t, tm, "User")
	require.Contains(t, tm, "Product")
	assert.Len(t, tm["User"].Fields, 1)
	assert.Len(t, tm["Product"].Fields, 2)
}

// ==================== Description ====================

func TestBuild_Description(t *testing.T) {
	g := buildOne(t, "Item", `{
		"type": "object",
		"description": "An item",
		"properties": {
			"name": {"type": "string", "description": "The name"}
		}
	}`, BuildConfig{})

	typ := g.Types[0]
	assert.Equal(t, "An item", typ.Description)
	assert.Equal(t, "The name", typ.Fields[0].Description)
}

// ==================== Const ====================

func TestBuild_ConstTypeRef(t *testing.T) {
	g := buildOne(t, "Config", `{
		"type": "object",
		"properties": {
			"version": {"const": "1.0"}
		}
	}`, BuildConfig{})

	versionField := fieldMap(g.Types[0])["version"]
	assert.Equal(t, KindEnum, versionField.Type.Kind)
	require.Len(t, versionField.Type.EnumValues, 1)
	assert.Equal(t, "1.0", versionField.Type.EnumValues[0])
}

// ==================== Object without properties (map) ====================

func TestBuild_EmptyObject(t *testing.T) {
	g := buildOne(t, "Container", `{
		"type": "object",
		"properties": {
			"data": {"type": "object"}
		}
	}`, BuildConfig{})

	dataField := fieldMap(g.Types[0])["data"]
	assert.Equal(t, KindMap, dataField.Type.Kind)
}

// ==================== Nullable ====================

func TestBuild_NullableField(t *testing.T) {
	g := buildOne(t, "Nullable", `{
		"type": "object",
		"properties": {
			"name": {"type": ["string", "null"]}
		}
	}`, BuildConfig{})

	nameField := fieldMap(g.Types[0])["name"]
	assert.True(t, nameField.Type.Nullable)
	assert.Equal(t, KindPrimitive, nameField.Type.Kind)
	assert.Equal(t, PrimString, nameField.Type.Primitive)
}

// ==================== UniqueMemberName ====================

func TestUniqueMemberName(t *testing.T) {
	assert.Equal(t, "Base", uniqueMemberName("Base", 0))
	assert.Equal(t, "Base1", uniqueMemberName("Base", 1))
	assert.Equal(t, "Base2", uniqueMemberName("Base", 2))
}

// ==================== Inline union extraction ====================

func TestBuild_InlineExtraction_Union(t *testing.T) {
	g := buildOne(t, "Container", `{
		"type": "object",
		"properties": {
			"value": {
				"anyOf": [
					{"type": "string"},
					{"type": "integer"}
				]
			}
		}
	}`, BuildConfig{ExtractInlined: true})

	tm := typeMap(g)
	require.Contains(t, tm, "Container")

	valueField := fieldMap(tm["Container"])["value"]
	assert.Equal(t, KindUnion, valueField.Type.Kind)
	require.Len(t, valueField.Type.UnionMembers, 2)
}

func TestBuild_InlineUnion_Disabled(t *testing.T) {
	g := buildOne(t, "Container", `{
		"type": "object",
		"properties": {
			"value": {
				"anyOf": [
					{"type": "string"},
					{"type": "integer"}
				]
			}
		}
	}`, BuildConfig{ExtractInlined: false})

	require.Len(t, g.Types, 1)
	valueField := fieldMap(g.Types[0])["value"]
	assert.Equal(t, KindUnion, valueField.Type.Kind)
	assert.Len(t, valueField.Type.UnionMembers, 2)
}

// ==================== Processed types guard ====================

func TestBuild_ProcessedTypesGuard(t *testing.T) {
	g := buildOne(t, "Root", `{
		"type": "object",
		"properties": {
			"a": {"$ref": "#/$defs/Shared"},
			"b": {"$ref": "#/$defs/Shared"}
		},
		"$defs": {
			"Shared": {
				"type": "object",
				"properties": {
					"id": {"type": "string"}
				}
			}
		}
	}`, BuildConfig{})

	sharedCount := 0
	for _, t := range g.Types {
		if t.Name == "Shared" {
			sharedCount++
		}
	}
	assert.Equal(t, 1, sharedCount)
}

// ==================== TypeRef Walk ====================

func TestTypeRef_Walk(t *testing.T) {
	inner := &TypeRef{Kind: KindPrimitive, Primitive: PrimString}
	ref := &TypeRef{Kind: KindArray, ItemType: inner}

	var visited []TypeKind
	ref.Walk(func(r *TypeRef) {
		visited = append(visited, r.Kind)
	})
	assert.Equal(t, []TypeKind{KindArray, KindPrimitive}, visited)
}

func TestTypeRef_Walk_Nil(t *testing.T) {
	var ref *TypeRef
	ref.Walk(func(r *TypeRef) {
		t.Fatal("should not be called")
	})
}

// ==================== Graph methods ====================

func TestGraph_AddTypeAndGetType(t *testing.T) {
	g := NewGraph()
	g.AddType(&Type{Name: "Foo"})
	g.AddType(&Type{Name: "Bar"})

	assert.Equal(t, "Foo", g.GetType("Foo").Name)
	assert.Equal(t, "Bar", g.GetType("Bar").Name)
	assert.Nil(t, g.GetType("Baz"))
}

func TestGraph_GetType_NilIndex(t *testing.T) {
	g := &Graph{
		Types: []*Type{{Name: "X"}},
	}
	assert.Equal(t, "X", g.GetType("X").Name)
	assert.Nil(t, g.GetType("Y"))
}

// ==================================================================
// Integration tests with real test schemas
// ==================================================================

func TestIntegration_EcommerceOrder(t *testing.T) {
	ns, err := parse.ParseFile("/Users/mirpo/Projects/schemagen/testdata/schemas/complex/ecommerce_order.json")
	require.NoError(t, err)

	g, err := Build([]*parse.NamedSchema{ns}, BuildConfig{})
	require.NoError(t, err)

	tm := typeMap(g)
	require.Contains(t, tm, "Money")
	require.Contains(t, tm, "Address")
	require.Contains(t, tm, "LineItem")
	require.Contains(t, tm, "EcommerceOrder")
	assert.Len(t, g.Types, 4)

	money := tm["Money"]
	assert.Equal(t, KindStruct, money.Kind)
	mfm := fieldMap(money)
	assert.True(t, mfm["amount"].Required)
	assert.True(t, mfm["currency"].Required)
	assert.Equal(t, KindPrimitive, mfm["amount"].Type.Kind)
	assert.Equal(t, PrimFloat64, mfm["amount"].Type.Primitive)
	assert.Equal(t, KindEnum, mfm["currency"].Type.Kind)
	assert.Len(t, mfm["currency"].Type.EnumValues, 4)

	addr := tm["Address"]
	assert.Equal(t, KindStruct, addr.Kind)
	afm := fieldMap(addr)
	assert.True(t, afm["name"].Required)
	assert.True(t, afm["street"].Required)
	assert.True(t, afm["city"].Required)
	assert.True(t, afm["country"].Required)
	assert.False(t, afm["state"].Required)

	lineItem := tm["LineItem"]
	lifm := fieldMap(lineItem)
	assert.Equal(t, KindRef, lifm["unitPrice"].Type.Kind)
	assert.Equal(t, "Money", lifm["unitPrice"].Type.TypeName)
	assert.Equal(t, KindRef, lifm["totalPrice"].Type.Kind)

	order := tm["EcommerceOrder"]
	ofm := fieldMap(order)
	assert.Equal(t, KindPrimitive, ofm["orderDate"].Type.Kind)
	assert.Equal(t, PrimDateTime, ofm["orderDate"].Type.Primitive)
	assert.Equal(t, KindEnum, ofm["status"].Type.Kind)
	assert.Len(t, ofm["status"].Type.EnumValues, 5)
	assert.Equal(t, KindArray, ofm["items"].Type.Kind)
	assert.Equal(t, KindRef, ofm["items"].Type.ItemType.Kind)
	assert.Equal(t, "LineItem", ofm["items"].Type.ItemType.TypeName)
	assert.Equal(t, KindRef, ofm["shippingAddress"].Type.Kind)
	assert.Equal(t, "Address", ofm["shippingAddress"].Type.TypeName)
	require.NotNil(t, ofm["items"].MinItems)
	assert.Equal(t, 1, *ofm["items"].MinItems)
}

func TestIntegration_CyclicRef(t *testing.T) {
	ns, err := parse.ParseFile("/Users/mirpo/Projects/schemagen/testdata/schemas/edge-cases/cyclic-ref.json")
	require.NoError(t, err)

	g, err := Build([]*parse.NamedSchema{ns}, BuildConfig{})
	require.NoError(t, err)

	assert.Len(t, g.Types, 1)
	typ := g.Types[0]
	assert.Equal(t, "CyclicRef", typ.Name)
	assert.Equal(t, KindStruct, typ.Kind)

	fm := fieldMap(typ)
	assert.Equal(t, KindRef, fm["parent"].Type.Kind)
	assert.Equal(t, "CyclicRef", fm["parent"].Type.TypeName)
	assert.Equal(t, KindArray, fm["children"].Type.Kind)
	assert.Equal(t, KindRef, fm["children"].Type.ItemType.Kind)
	assert.Equal(t, "CyclicRef", fm["children"].Type.ItemType.TypeName)
}

func TestIntegration_Document(t *testing.T) {
	ns, err := parse.ParseFile("/Users/mirpo/Projects/schemagen/testdata/schemas/allof/document.json")
	require.NoError(t, err)

	g, err := Build([]*parse.NamedSchema{ns}, BuildConfig{})
	require.NoError(t, err)

	tm := typeMap(g)
	require.Contains(t, tm, "Entity")
	require.Contains(t, tm, "Auditable")
	require.Contains(t, tm, "Document")
	assert.Len(t, g.Types, 3)

	doc := tm["Document"]
	assert.Equal(t, KindStruct, doc.Kind)
	assert.Contains(t, doc.Extends, "Entity")
	assert.Contains(t, doc.Extends, "Auditable")
	assert.Len(t, doc.Extends, 2)

	dfm := fieldMap(doc)
	require.Contains(t, dfm, "title")
	require.Contains(t, dfm, "content")
	require.Contains(t, dfm, "author")
	require.Contains(t, dfm, "tags")

	assert.True(t, dfm["title"].Required)
	assert.True(t, dfm["content"].Required)

	assert.Equal(t, "First title definition", dfm["title"].Description)

	assert.Equal(t, KindArray, dfm["tags"].Type.Kind)
	assert.Equal(t, KindPrimitive, dfm["tags"].Type.ItemType.Kind)
}

func TestIntegration_Notification(t *testing.T) {
	ns, err := parse.ParseFile("/Users/mirpo/Projects/schemagen/testdata/schemas/anyof/notification.json")
	require.NoError(t, err)

	g, err := Build([]*parse.NamedSchema{ns}, BuildConfig{})
	require.NoError(t, err)

	tm := typeMap(g)
	require.Contains(t, tm, "Notification")

	notification := tm["Notification"]
	assert.Equal(t, KindStruct, notification.Kind)

	nfm := fieldMap(notification)
	assert.True(t, nfm["id"].Required)
	assert.True(t, nfm["timestamp"].Required)
	assert.True(t, nfm["channel"].Required)
	assert.True(t, nfm["content"].Required)

	assert.Equal(t, PrimDateTime, nfm["timestamp"].Type.Primitive)

	channelRef := nfm["channel"].Type
	assert.Equal(t, KindUnion, channelRef.Kind)
	assert.Len(t, channelRef.UnionMembers, 3)
	for _, m := range channelRef.UnionMembers {
		assert.Equal(t, KindRef, m.Kind)
	}

	contentRef := nfm["content"].Type
	assert.Equal(t, KindUnion, contentRef.Kind)
	assert.Len(t, contentRef.UnionMembers, 2)

	assert.Equal(t, KindEnum, nfm["priority"].Type.Kind)
	assert.Len(t, nfm["priority"].Type.EnumValues, 4)

	// Verify inline object types were created
	inlineTypes := 0
	for _, typ := range g.Types {
		if typ.Kind == KindStruct && typ.Name != "Notification" {
			inlineTypes++
		}
	}
	assert.Equal(t, 5, inlineTypes)
}

func TestIntegration_Foundation(t *testing.T) {
	ns, err := parse.ParseFile("/Users/mirpo/Projects/schemagen/testdata/schemas/foundation/foundation.json")
	require.NoError(t, err)

	g, err := Build([]*parse.NamedSchema{ns}, BuildConfig{})
	require.NoError(t, err)

	tm := typeMap(g)
	require.Contains(t, tm, "Foundation")

	foundation := tm["Foundation"]
	ffm := fieldMap(foundation)

	primitivesField := ffm["primitives"]
	require.NotNil(t, primitivesField)
	assert.Equal(t, KindInterface, primitivesField.Type.Kind)
	require.Len(t, primitivesField.Type.ObjectFields, 4)

	primFM := make(map[string]*Field)
	for _, f := range primitivesField.Type.ObjectFields {
		primFM[f.JSONName] = f
	}
	assert.Equal(t, PrimString, primFM["stringVal"].Type.Primitive)
	assert.Equal(t, PrimFloat64, primFM["numberVal"].Type.Primitive)
	assert.Equal(t, PrimInt, primFM["integerVal"].Type.Primitive)
	assert.Equal(t, PrimBool, primFM["booleanVal"].Type.Primitive)

	formatsField := ffm["formats"]
	require.NotNil(t, formatsField)
	formatFM := make(map[string]*Field)
	for _, f := range formatsField.Type.ObjectFields {
		formatFM[f.JSONName] = f
	}
	assert.Equal(t, PrimEmail, formatFM["email"].Type.Primitive)
	assert.Equal(t, PrimURI, formatFM["uri"].Type.Primitive)
	assert.Equal(t, PrimUUID, formatFM["uuid"].Type.Primitive)
	assert.Equal(t, PrimDateTime, formatFM["dateTime"].Type.Primitive)
	assert.Equal(t, PrimDate, formatFM["date"].Type.Primitive)
	assert.Equal(t, PrimTime, formatFM["time"].Type.Primitive)
	assert.Equal(t, PrimIPv4, formatFM["ipv4"].Type.Primitive)
	assert.Equal(t, PrimIPv6, formatFM["ipv6"].Type.Primitive)

	addPropsField := ffm["additionalProps"]
	require.NotNil(t, addPropsField)
	apFM := make(map[string]*Field)
	for _, f := range addPropsField.Type.ObjectFields {
		apFM[f.JSONName] = f
	}
	strictField := apFM["strict"]
	require.NotNil(t, strictField)
	assert.Equal(t, KindInterface, strictField.Type.Kind)

	enumsField := ffm["enums"]
	require.NotNil(t, enumsField)
	enumFM := make(map[string]*Field)
	for _, f := range enumsField.Type.ObjectFields {
		enumFM[f.JSONName] = f
	}
	assert.Equal(t, KindEnum, enumFM["stringEnum"].Type.Kind)
	assert.Len(t, enumFM["stringEnum"].Type.EnumValues, 3)
	assert.Equal(t, KindEnum, enumFM["numberEnum"].Type.Kind)
	assert.Len(t, enumFM["numberEnum"].Type.EnumValues, 3)
	assert.Equal(t, KindEnum, enumFM["mixedEnum"].Type.Kind)
	assert.Len(t, enumFM["mixedEnum"].Type.EnumValues, 4)
}

func TestIntegration_Organization(t *testing.T) {
	ns, err := parse.ParseFile("/Users/mirpo/Projects/schemagen/testdata/schemas/refs/organization.json")
	require.NoError(t, err)

	g, err := Build([]*parse.NamedSchema{ns}, BuildConfig{})
	require.NoError(t, err)

	tm := typeMap(g)
	require.Contains(t, tm, "Country")
	require.Contains(t, tm, "Address")
	require.Contains(t, tm, "Office")
	require.Contains(t, tm, "Organization")

	// Verify chained refs: Organization -> Office -> Address -> Country
	org := tm["Organization"]
	ofm := fieldMap(org)
	assert.Equal(t, KindRef, ofm["headquarters"].Type.Kind)
	assert.Equal(t, "Office", ofm["headquarters"].Type.TypeName)
	assert.Equal(t, KindArray, ofm["branches"].Type.Kind)
	assert.Equal(t, KindRef, ofm["branches"].Type.ItemType.Kind)
	assert.Equal(t, "Office", ofm["branches"].Type.ItemType.TypeName)

	office := tm["Office"]
	oFM := fieldMap(office)
	assert.Equal(t, KindRef, oFM["address"].Type.Kind)
	assert.Equal(t, "Address", oFM["address"].Type.TypeName)

	addr := tm["Address"]
	aFM := fieldMap(addr)
	assert.Equal(t, KindRef, aFM["country"].Type.Kind)
	assert.Equal(t, "Country", aFM["country"].Type.TypeName)

	country := tm["Country"]
	cFM := fieldMap(country)
	assert.Equal(t, KindEnum, cFM["code"].Type.Kind)
	assert.Len(t, cFM["code"].Type.EnumValues, 6)
}

// ==================== Object with enum field is still struct ====================

func TestBuild_ObjectWithEnumFieldIsStruct(t *testing.T) {
	g := buildOne(t, "Item", `{
		"type": "object",
		"properties": {
			"status": {
				"type": "string",
				"enum": ["on", "off"]
			},
			"name": {"type": "string"}
		}
	}`, BuildConfig{})

	require.Len(t, g.Types, 1)
	assert.Equal(t, KindStruct, g.Types[0].Kind)
}

// ==================== Enum at root with object type skipped ====================

func TestBuild_RootEnumOnly(t *testing.T) {
	g := buildOne(t, "Color", `{
		"type": "string",
		"enum": ["red", "green", "blue"]
	}`, BuildConfig{})

	require.Len(t, g.Types, 1)
	assert.Equal(t, KindEnum, g.Types[0].Kind)
}

// ==================== Root-level array schema ====================

func TestBuild_RootArraySchema(t *testing.T) {
	g := buildOne(t, "StringList", `{
		"title": "StringList",
		"type": "array",
		"items": {"type": "string"}
	}`, BuildConfig{})

	require.Len(t, g.Types, 1)
	typ := g.Types[0]
	assert.Equal(t, "StringList", typ.Name)
}

// ==================== Primitive at root ====================

func TestBuild_PrimitiveAtRoot(t *testing.T) {
	g := buildOne(t, "CustomString", `{
		"type": "string",
		"format": "uuid"
	}`, BuildConfig{})

	require.Len(t, g.Types, 1)
	assert.Equal(t, KindPrimitive, g.Types[0].Kind)
	assert.Equal(t, PrimUUID, g.Types[0].Primitive)
}

// ==================== Ref path resolution ====================

func TestResolveRefName_ForwardSlashPaths(t *testing.T) {
	b := &builder{
		graph:          NewGraph(),
		cfg:            BuildConfig{},
		globalDefs:     make(map[string]*parse.SchemaNode),
		processedTypes: make(map[string]bool),
		schemaNames:    make(map[string]string),
	}
	b.schemaNames["schemas/address.json"] = "Address"
	b.schemaNames["./schemas/address.json"] = "Address"

	assert.Equal(t, "Address", b.resolveRefName("./schemas/address.json", "Root"))
	assert.Equal(t, "Address", b.resolveRefName("schemas/address.json", "Root"))

	name := b.resolveRefName("./other/person.json", "Root")
	assert.Equal(t, "Person", name)
}

// ==================== AllOf root required merge ====================

func TestBuild_AllOfRootRequired(t *testing.T) {
	g := buildOne(t, "Merged", `{
		"required": ["fieldFromBranch"],
		"allOf": [
			{
				"type": "object",
				"properties": {
					"fieldFromBranch": {"type": "string"}
				}
			}
		]
	}`, BuildConfig{})

	require.Len(t, g.Types, 1)
	typ := g.Types[0]
	fm := fieldMap(typ)
	require.Contains(t, fm, "fieldFromBranch")
	assert.True(t, fm["fieldFromBranch"].Required, "fieldFromBranch should be required via root-level required")
}
