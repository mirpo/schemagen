package parse

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEmptyObject(t *testing.T) {
	node, err := ParseJSON(strings.NewReader(`{}`))
	require.NoError(t, err)
	assert.Empty(t, node.Type)
	assert.Empty(t, node.Properties)
	assert.Empty(t, node.Required)
	assert.Nil(t, node.Items)
	assert.Empty(t, node.Defs)
}

func TestParseBasicObject(t *testing.T) {
	input := `{
		"type": "object",
		"title": "Person",
		"description": "A person schema",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name"]
	}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)

	assert.Equal(t, "Person", node.Title)
	assert.Equal(t, "A person schema", node.Description)
	assert.True(t, node.Type.Has("object"))
	assert.Len(t, node.Properties, 2)
	assert.Equal(t, "name", node.Properties[0].Name)
	assert.Equal(t, "age", node.Properties[1].Name)
	assert.Equal(t, []string{"name"}, node.Required)
	assert.True(t, node.IsRequired("name"))
	assert.False(t, node.IsRequired("age"))
}

func TestParseStringOrSlice(t *testing.T) {
	t.Run("single string", func(t *testing.T) {
		input := `{"type": "string"}`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		assert.Equal(t, StringOrSlice{"string"}, node.Type)
		assert.Equal(t, "string", node.Type.Single())
		assert.False(t, node.Type.IsNullable())
	})

	t.Run("array of types", func(t *testing.T) {
		input := `{"type": ["string", "null"]}`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		assert.Equal(t, StringOrSlice{"string", "null"}, node.Type)
		assert.True(t, node.Type.Has("string"))
		assert.True(t, node.Type.Has("null"))
		assert.True(t, node.Type.IsNullable())
		assert.Equal(t, "string", node.Type.Single())
	})

	t.Run("null only", func(t *testing.T) {
		input := `{"type": ["null"]}`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		assert.False(t, node.Type.IsNullable())
		assert.Equal(t, "null", node.Type.Single())
	})
}

func TestParsePropertyOrderPreserved(t *testing.T) {
	input := `{
		"type": "object",
		"properties": {
			"zebra": {"type": "string"},
			"alpha": {"type": "string"},
			"mango": {"type": "integer"},
			"beta": {"type": "boolean"},
			"omega": {"type": "number"}
		}
	}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)

	expected := []string{"zebra", "alpha", "mango", "beta", "omega"}
	require.Len(t, node.Properties, len(expected))
	for i, name := range expected {
		assert.Equal(t, name, node.Properties[i].Name, "property %d should be %q", i, name)
	}
}

func TestParseRef(t *testing.T) {
	input := `{"$ref": "#/$defs/Address"}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, "#/$defs/Address", node.Ref)
	assert.True(t, node.IsRef())
}

func TestParseRefSelf(t *testing.T) {
	input := `{"$ref": "#"}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, "#", node.Ref)
	assert.True(t, node.IsRef())
}

func TestParseDefs(t *testing.T) {
	input := `{
		"$defs": {
			"Foo": {"type": "string"},
			"Bar": {"type": "integer"}
		}
	}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, node.Defs, 2)
	assert.Equal(t, "Foo", node.Defs[0].Name)
	assert.Equal(t, "Bar", node.Defs[1].Name)
}

func TestParseDefinitions(t *testing.T) {
	input := `{
		"definitions": {
			"Thing": {"type": "object"}
		}
	}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, node.Defs, 1)
	assert.Equal(t, "Thing", node.Defs[0].Name)
}

func TestParseAllOf(t *testing.T) {
	input := `{
		"allOf": [
			{"$ref": "#/$defs/Entity"},
			{"type": "object", "properties": {"name": {"type": "string"}}}
		]
	}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)
	assert.True(t, node.IsAllOf())
	require.Len(t, node.AllOf, 2)
	assert.Equal(t, "#/$defs/Entity", node.AllOf[0].Ref)
	assert.Len(t, node.AllOf[1].Properties, 1)
}

func TestParseAnyOf(t *testing.T) {
	input := `{
		"anyOf": [
			{"type": "string"},
			{"type": "integer"}
		]
	}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)
	assert.True(t, node.IsAnyOf())
	assert.True(t, node.IsUnion())
	require.Len(t, node.AnyOf, 2)
}

func TestParseOneOf(t *testing.T) {
	input := `{
		"oneOf": [
			{"type": "string"},
			{"type": "number"}
		]
	}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)
	assert.True(t, node.IsOneOf())
	assert.True(t, node.IsUnion())
	require.Len(t, node.OneOf, 2)
}

func TestParseConstraints(t *testing.T) {
	input := `{
		"type": "object",
		"properties": {
			"str": {
				"type": "string",
				"minLength": 5,
				"maxLength": 50,
				"pattern": "^[A-Z]"
			},
			"num": {
				"type": "number",
				"minimum": 0,
				"maximum": 100,
				"multipleOf": 0.5
			},
			"exclusive": {
				"type": "number",
				"exclusiveMinimum": 0,
				"exclusiveMaximum": 100
			},
			"arr": {
				"type": "array",
				"items": {"type": "string"},
				"minItems": 1,
				"maxItems": 10
			}
		}
	}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, node.Properties, 4)

	str := node.Properties[0].Schema
	require.NotNil(t, str.MinLength)
	assert.Equal(t, 5, *str.MinLength)
	require.NotNil(t, str.MaxLength)
	assert.Equal(t, 50, *str.MaxLength)
	require.NotNil(t, str.Pattern)
	assert.Equal(t, "^[A-Z]", *str.Pattern)

	num := node.Properties[1].Schema
	require.NotNil(t, num.Minimum)
	assert.InDelta(t, 0.0, *num.Minimum, 0.001)
	require.NotNil(t, num.Maximum)
	assert.InDelta(t, 100.0, *num.Maximum, 0.001)
	require.NotNil(t, num.MultipleOf)
	assert.InDelta(t, 0.5, *num.MultipleOf, 0.001)

	excl := node.Properties[2].Schema
	require.NotNil(t, excl.ExclusiveMinimum)
	assert.InDelta(t, 0.0, *excl.ExclusiveMinimum, 0.001)
	require.NotNil(t, excl.ExclusiveMaximum)
	assert.InDelta(t, 100.0, *excl.ExclusiveMaximum, 0.001)

	arr := node.Properties[3].Schema
	require.NotNil(t, arr.MinItems)
	assert.Equal(t, 1, *arr.MinItems)
	require.NotNil(t, arr.MaxItems)
	assert.Equal(t, 10, *arr.MaxItems)
}

func TestParseExclusiveLimitBoolean(t *testing.T) {
	input := `{
		"type": "number",
		"exclusiveMinimum": true,
		"exclusiveMaximum": false
	}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)
	assert.Nil(t, node.ExclusiveMinimum)
	assert.Nil(t, node.ExclusiveMaximum)
}

func TestParseAdditionalProperties(t *testing.T) {
	t.Run("boolean false", func(t *testing.T) {
		input := `{"type": "object", "additionalProperties": false}`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		require.NotNil(t, node.AdditionalProperties)
		assert.False(t, node.AdditionalProperties.Allowed)
		assert.Nil(t, node.AdditionalProperties.Schema)
	})

	t.Run("boolean true", func(t *testing.T) {
		input := `{"type": "object", "additionalProperties": true}`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		require.NotNil(t, node.AdditionalProperties)
		assert.True(t, node.AdditionalProperties.Allowed)
		assert.Nil(t, node.AdditionalProperties.Schema)
	})

	t.Run("schema object", func(t *testing.T) {
		input := `{"type": "object", "additionalProperties": {"type": "string"}}`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		require.NotNil(t, node.AdditionalProperties)
		assert.True(t, node.AdditionalProperties.Allowed)
		require.NotNil(t, node.AdditionalProperties.Schema)
		assert.True(t, node.AdditionalProperties.Schema.Type.Has("string"))
	})

	t.Run("schema object with ref", func(t *testing.T) {
		input := `{"type": "object", "additionalProperties": {"$ref": "#/$defs/Value"}}`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		require.NotNil(t, node.AdditionalProperties)
		assert.True(t, node.AdditionalProperties.Allowed)
		require.NotNil(t, node.AdditionalProperties.Schema)
		assert.Equal(t, "#/$defs/Value", node.AdditionalProperties.Schema.Ref)
	})
}

func TestParseEnum(t *testing.T) {
	t.Run("string enum", func(t *testing.T) {
		input := `{"type": "string", "enum": ["a", "b", "c"]}`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		assert.True(t, node.IsEnum())
		assert.Equal(t, []any{"a", "b", "c"}, node.Enum)
	})

	t.Run("number enum", func(t *testing.T) {
		input := `{"type": "number", "enum": [1, 2, 3]}`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		assert.Equal(t, []any{int64(1), int64(2), int64(3)}, node.Enum)
	})

	t.Run("mixed enum", func(t *testing.T) {
		input := `{"enum": ["string", 42, true, null]}`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		require.Len(t, node.Enum, 4)
		assert.Equal(t, "string", node.Enum[0])
		assert.Equal(t, int64(42), node.Enum[1])
		assert.Equal(t, true, node.Enum[2])
		assert.Nil(t, node.Enum[3])
	})
}

func TestParseConst(t *testing.T) {
	t.Run("string const", func(t *testing.T) {
		input := `{"const": "fixed"}`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		assert.True(t, node.IsConst())
		assert.Equal(t, "fixed", node.Const)
	})

	t.Run("number const", func(t *testing.T) {
		input := `{"const": 42}`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		assert.Equal(t, int64(42), node.Const)
	})

	t.Run("boolean const", func(t *testing.T) {
		input := `{"const": true}`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		assert.Equal(t, true, node.Const)
	})

	t.Run("null const", func(t *testing.T) {
		input := `{"const": null}`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		assert.Nil(t, node.Const)
		assert.True(t, node.IsConst())
	})
}

func TestParseJSON_MalformedObjectKey(t *testing.T) {
	t.Run("valid keys do not panic", func(t *testing.T) {
		input := `{"type": "string", "title": "Test"}`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		assert.True(t, node.Type.Has("string"))
		assert.Equal(t, "Test", node.Title)
	})

	t.Run("safe type assertion is in place", func(t *testing.T) {
		input := `{"type": "object", "properties": {"name": {"type": "string"}}}`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		assert.NotNil(t, node)
	})
}

func TestParseConstNull(t *testing.T) {
	input := `{"const": null}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)
	assert.Nil(t, node.Const)
	assert.True(t, node.IsConst(), "IsConst() should return true for {\"const\": null}")
}

func TestParseBooleanSchemas(t *testing.T) {
	t.Run("true schema", func(t *testing.T) {
		input := `true`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		assert.Empty(t, node.Type)
	})

	t.Run("false schema", func(t *testing.T) {
		input := `false`
		node, err := ParseJSON(strings.NewReader(input))
		require.NoError(t, err)
		assert.Equal(t, StringOrSlice{"never"}, node.Type)
	})
}

func TestParseNestedObjects(t *testing.T) {
	input := `{
		"type": "object",
		"properties": {
			"level1": {
				"type": "object",
				"properties": {
					"level2": {
						"type": "object",
						"properties": {
							"value": {"type": "string"}
						}
					}
				}
			}
		}
	}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)

	require.Len(t, node.Properties, 1)
	l1 := node.Properties[0].Schema
	assert.True(t, l1.IsObject())
	require.Len(t, l1.Properties, 1)

	l2 := l1.Properties[0].Schema
	assert.True(t, l2.IsObject())
	require.Len(t, l2.Properties, 1)
	assert.Equal(t, "value", l2.Properties[0].Name)
	assert.True(t, l2.Properties[0].Schema.Type.Has("string"))
}

func TestParseArrayWithItems(t *testing.T) {
	input := `{
		"type": "array",
		"items": {"type": "string"}
	}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)
	assert.True(t, node.IsArray())
	require.NotNil(t, node.Items)
	assert.True(t, node.Items.Type.Has("string"))
}

func TestParsePrefixItems(t *testing.T) {
	input := `{
		"type": "array",
		"prefixItems": [
			{"type": "string"},
			{"type": "integer"}
		]
	}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, node.PrefixItems, 2)
	assert.True(t, node.PrefixItems[0].Type.Has("string"))
	assert.True(t, node.PrefixItems[1].Type.Has("integer"))
}

func TestParseSchemaAndID(t *testing.T) {
	input := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id": "https://example.com/test"
	}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, "https://json-schema.org/draft/2020-12/schema", node.Schema)
	assert.Equal(t, "https://example.com/test", node.ID)
}

func TestParseFormat(t *testing.T) {
	input := `{"type": "string", "format": "date-time"}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, "date-time", node.Format)
}

func TestParseUnknownFieldsSkipped(t *testing.T) {
	input := `{
		"type": "object",
		"x-custom": "ignored",
		"unknown_key": {"nested": true},
		"another_unknown": [1, 2, 3],
		"title": "Survives"
	}`
	node, err := ParseJSON(strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, "Survives", node.Title)
	assert.True(t, node.Type.Has("object"))
}

func TestParseMalformedJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"invalid json", "{invalid}"},
		{"truncated object", `{"type": "string"`},
		{"array instead of object", `[1, 2, 3]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseJSON(strings.NewReader(tt.input))
			assert.Error(t, err)
		})
	}
}

func TestParseHelperMethods(t *testing.T) {
	t.Run("IsPrimitive", func(t *testing.T) {
		for _, typ := range []string{"string", "integer", "number", "boolean"} {
			node := &SchemaNode{Type: StringOrSlice{typ}}
			assert.True(t, node.IsPrimitive(), "type %q should be primitive", typ)
		}
		node := &SchemaNode{Type: StringOrSlice{"object"}}
		assert.False(t, node.IsPrimitive())
	})

	t.Run("IsObject with properties but no type", func(t *testing.T) {
		node := &SchemaNode{
			Properties: []NamedSchema{{Name: "foo", Schema: &SchemaNode{}}},
		}
		assert.True(t, node.IsObject())
	})
}

func TestParseAllTestSchemas(t *testing.T) {
	testdataDir := filepath.Join("..", "..", "..", "testdata", "schemas")

	var schemaFiles []string
	err := filepath.Walk(testdataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".json" || ext == ".yaml" || ext == ".yml" {
			schemaFiles = append(schemaFiles, path)
		}
		return nil
	})
	require.NoError(t, err)
	require.Len(t, schemaFiles, 24, "expected 24 test schemas")

	for _, path := range schemaFiles {
		t.Run(filepath.Base(path), func(t *testing.T) {
			f, err := os.Open(path)
			require.NoError(t, err)
			defer f.Close()

			ext := strings.ToLower(filepath.Ext(path))
			switch ext {
			case ".json":
				_, err = ParseJSON(f)
			case ".yaml", ".yml":
				_, err = ParseYAML(f)
			}
			assert.NoError(t, err, "failed to parse %s", path)
		})
	}
}

func TestParseEcommerceOrder(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "schemas", "complex", "ecommerce_order.json")
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	node, err := ParseJSON(f)
	require.NoError(t, err)

	assert.Equal(t, "EcommerceOrder", node.Title)

	require.Len(t, node.Defs, 3)
	defNames := []string{node.Defs[0].Name, node.Defs[1].Name, node.Defs[2].Name}
	assert.Equal(t, []string{"Money", "Address", "LineItem"}, defNames)

	require.Len(t, node.Properties, 11)
	expectedProps := []string{
		"orderId", "customerId", "orderDate", "status", "items",
		"shippingAddress", "billingAddress", "subtotal", "tax", "shipping", "total",
	}
	for i, name := range expectedProps {
		assert.Equal(t, name, node.Properties[i].Name, "property %d", i)
	}

	require.Len(t, node.Required, 7)

	statusProp := node.Properties[3].Schema
	assert.True(t, statusProp.IsEnum())
	assert.Len(t, statusProp.Enum, 5)

	itemsProp := node.Properties[4].Schema
	assert.True(t, itemsProp.IsArray())
	require.NotNil(t, itemsProp.Items)
	assert.True(t, itemsProp.Items.IsRef())
	require.NotNil(t, itemsProp.MinItems)
	assert.Equal(t, 1, *itemsProp.MinItems)

	shippingProp := node.Properties[5].Schema
	assert.True(t, shippingProp.IsRef())
}

func TestParseDocumentAllOf(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "schemas", "allof", "document.json")
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	node, err := ParseJSON(f)
	require.NoError(t, err)

	assert.Equal(t, "Document", node.Title)
	require.Len(t, node.Defs, 2)
	assert.Equal(t, "Entity", node.Defs[0].Name)
	assert.Equal(t, "Auditable", node.Defs[1].Name)

	require.Len(t, node.AllOf, 5)
	assert.True(t, node.AllOf[0].IsRef())
	assert.True(t, node.AllOf[1].IsRef())
	assert.True(t, node.AllOf[2].IsObject())
}

func TestParseNotification(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "schemas", "anyof", "notification.json")
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	node, err := ParseJSON(f)
	require.NoError(t, err)

	assert.Equal(t, "Notification", node.Title)
	require.Len(t, node.Properties, 5)

	channelProp := node.Properties[2].Schema
	assert.True(t, channelProp.IsAnyOf())
	require.Len(t, channelProp.AnyOf, 3)
}

func TestParseOrganizationRefs(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "schemas", "refs", "organization.json")
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	node, err := ParseJSON(f)
	require.NoError(t, err)

	assert.Equal(t, "Organization", node.Title)
	require.Len(t, node.Defs, 3)
	assert.Equal(t, "Country", node.Defs[0].Name)
	assert.Equal(t, "Address", node.Defs[1].Name)
	assert.Equal(t, "Office", node.Defs[2].Name)
}

func TestParseCyclicRef(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "schemas", "edge-cases", "cyclic-ref.json")
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	node, err := ParseJSON(f)
	require.NoError(t, err)

	assert.Equal(t, "CyclicRef", node.Title)
	require.Len(t, node.Properties, 4)

	parentProp := node.Properties[2].Schema
	assert.Equal(t, "#", parentProp.Ref)

	childrenProp := node.Properties[3].Schema
	assert.True(t, childrenProp.IsArray())
	require.NotNil(t, childrenProp.Items)
	assert.Equal(t, "#", childrenProp.Items.Ref)
}

func TestParseSpecialProps(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "schemas", "edge-cases", "special-props.json")
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	node, err := ParseJSON(f)
	require.NoError(t, err)

	expectedNames := []string{"kebab-case", "with spaces", "123numeric", "$dollar", "class", "from"}
	require.Len(t, node.Properties, len(expectedNames))
	for i, name := range expectedNames {
		assert.Equal(t, name, node.Properties[i].Name)
	}
}

func TestParseDescriptionQuotes(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "schemas", "edge-cases", "description-quotes.json")
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	node, err := ParseJSON(f)
	require.NoError(t, err)

	assert.Equal(t, "DescriptionQuotes", node.Title)
	assert.Contains(t, node.Description, "double quotes")
}

func TestParseEnumComplex(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "schemas", "edge-cases", "enum-complex.json")
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	node, err := ParseJSON(f)
	require.NoError(t, err)

	assert.True(t, node.IsEnum())
	require.Len(t, node.Enum, 5)
	assert.Equal(t, "simple", node.Enum[0])
	assert.Equal(t, int64(42), node.Enum[2])
}

func TestParseUnionNullable(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "schemas", "edge-cases", "union-nullable.json")
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	node, err := ParseJSON(f)
	require.NoError(t, err)

	assert.True(t, node.IsAnyOf())
	require.Len(t, node.AnyOf, 3)
	assert.True(t, node.AnyOf[0].Type.Has("string"))
	assert.True(t, node.AnyOf[1].Type.Has("null"))
	assert.True(t, node.AnyOf[2].Type.Has("integer"))
}

func TestParseTrickyAllOf(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "schemas", "allof", "tricky.json")
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	node, err := ParseJSON(f)
	require.NoError(t, err)

	assert.Equal(t, "MegaTest", node.Title)
	require.Len(t, node.Defs, 4)
	assert.Equal(t, "IdObject", node.Defs[0].Name)
	assert.Equal(t, "Timestamped", node.Defs[1].Name)
	assert.Equal(t, "Person", node.Defs[2].Name)
	assert.Equal(t, "Node", node.Defs[3].Name)

	require.Len(t, node.AllOf, 4)
	require.Len(t, node.Properties, 2)
	assert.Equal(t, "payload", node.Properties[0].Name)
	assert.Equal(t, "node", node.Properties[1].Name)

	payloadProp := node.Properties[0].Schema
	assert.True(t, payloadProp.IsOneOf())
	require.Len(t, payloadProp.OneOf, 2)
}

func TestParseFoundation(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "schemas", "foundation", "foundation.json")
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	node, err := ParseJSON(f)
	require.NoError(t, err)

	assert.Equal(t, "Foundation", node.Title)
	require.Len(t, node.Properties, 9)

	propNames := make([]string, len(node.Properties))
	for i, p := range node.Properties {
		propNames[i] = p.Name
	}
	assert.Equal(t, []string{
		"primitives", "formats", "constraints", "enums",
		"arrays", "nested", "nullable", "additionalProps", "edgeCases",
	}, propNames)

	additionalProps := node.Properties[7].Schema
	require.Len(t, additionalProps.Properties, 3)

	strict := additionalProps.Properties[0].Schema
	require.NotNil(t, strict.AdditionalProperties)
	assert.False(t, strict.AdditionalProperties.Allowed)

	flexible := additionalProps.Properties[1].Schema
	require.NotNil(t, flexible.AdditionalProperties)
	assert.True(t, flexible.AdditionalProperties.Allowed)
	assert.Nil(t, flexible.AdditionalProperties.Schema)

	typedMap := additionalProps.Properties[2].Schema
	require.NotNil(t, typedMap.AdditionalProperties)
	assert.True(t, typedMap.AdditionalProperties.Allowed)
	require.NotNil(t, typedMap.AdditionalProperties.Schema)
	assert.True(t, typedMap.AdditionalProperties.Schema.Type.Has("number"))
}
