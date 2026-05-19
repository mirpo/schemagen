package normalize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mirpo/schemagen/pkg/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseJSON(t *testing.T, jsonStr string) *parse.SchemaNode {
	t.Helper()
	node, err := parse.ParseJSON(strings.NewReader(jsonStr))
	require.NoError(t, err)
	return node
}

func TestDetectDraft(t *testing.T) {
	tests := []struct {
		name     string
		schema   string
		expected Draft
	}{
		{
			name:     "draft-04",
			schema:   `{"$schema": "http://json-schema.org/draft-04/schema#"}`,
			expected: Draft04,
		},
		{
			name:     "draft-07",
			schema:   `{"$schema": "http://json-schema.org/draft-07/schema#"}`,
			expected: Draft07,
		},
		{
			name:     "2019-09",
			schema:   `{"$schema": "https://json-schema.org/draft/2019-09/schema"}`,
			expected: Draft201909,
		},
		{
			name:     "2020-12",
			schema:   `{"$schema": "https://json-schema.org/draft/2020-12/schema"}`,
			expected: Draft202012,
		},
		{
			name:     "empty schema string",
			schema:   `{"$schema": ""}`,
			expected: DraftUnknown,
		},
		{
			name:     "no schema field",
			schema:   `{"type": "object"}`,
			expected: DraftUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := parseJSON(t, tt.schema)
			got := DetectDraft(node)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestNormalize_PassthroughModernDrafts(t *testing.T) {
	t.Run("2020-12 schema normalizes without changes", func(t *testing.T) {
		node := parseJSON(t, `{
			"$schema": "https://json-schema.org/draft/2020-12/schema",
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "integer", "minimum": 0}
			},
			"required": ["name"]
		}`)

		Normalize(node)

		assert.Equal(t, "https://json-schema.org/draft/2020-12/schema", node.Schema)
		assert.True(t, node.Type.Has("object"))
		require.Len(t, node.Properties, 2)
		assert.Equal(t, "name", node.Properties[0].Name)
		assert.True(t, node.Properties[0].Schema.Type.Has("string"))
		assert.Equal(t, "age", node.Properties[1].Name)
		assert.True(t, node.Properties[1].Schema.Type.Has("integer"))
		assert.Equal(t, []string{"name"}, node.Required)
	})

	t.Run("draft-07 schema normalizes without changes", func(t *testing.T) {
		node := parseJSON(t, `{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"properties": {
				"email": {"type": "string", "format": "email"},
				"active": {"type": "boolean"}
			},
			"required": ["email", "active"]
		}`)

		Normalize(node)

		assert.Equal(t, "http://json-schema.org/draft-07/schema#", node.Schema)
		assert.True(t, node.Type.Has("object"))
		require.Len(t, node.Properties, 2)
		assert.Equal(t, "email", node.Properties[0].Name)
		assert.Equal(t, "email", node.Properties[0].Schema.Format)
		assert.Equal(t, "active", node.Properties[1].Name)
		assert.True(t, node.Properties[1].Schema.Type.Has("boolean"))
		assert.Equal(t, []string{"email", "active"}, node.Required)
	})
}

func TestNormalize_RecursiveVisitation(t *testing.T) {
	t.Run("properties children are visited", func(t *testing.T) {
		node := parseJSON(t, `{
			"type": "object",
			"properties": {
				"inner": {
					"type": "object",
					"properties": {
						"value": {"type": "string"}
					}
				}
			}
		}`)

		Normalize(node)

		require.Len(t, node.Properties, 1)
		inner := node.Properties[0].Schema
		require.Len(t, inner.Properties, 1)
		assert.Equal(t, "value", inner.Properties[0].Name)
		assert.True(t, inner.Properties[0].Schema.Type.Has("string"))
	})

	t.Run("defs children are visited", func(t *testing.T) {
		node := parseJSON(t, `{
			"$defs": {
				"Foo": {
					"type": "object",
					"properties": {
						"bar": {"type": "integer"}
					}
				}
			}
		}`)

		Normalize(node)

		require.Len(t, node.Defs, 1)
		assert.Equal(t, "Foo", node.Defs[0].Name)
		foo := node.Defs[0].Schema
		require.Len(t, foo.Properties, 1)
		assert.Equal(t, "bar", foo.Properties[0].Name)
		assert.True(t, foo.Properties[0].Schema.Type.Has("integer"))
	})

	t.Run("allOf children are visited", func(t *testing.T) {
		node := parseJSON(t, `{
			"allOf": [
				{
					"type": "object",
					"properties": {
						"a": {"type": "string"}
					}
				},
				{
					"type": "object",
					"properties": {
						"b": {"type": "number"}
					}
				}
			]
		}`)

		Normalize(node)

		require.Len(t, node.AllOf, 2)
		require.Len(t, node.AllOf[0].Properties, 1)
		assert.Equal(t, "a", node.AllOf[0].Properties[0].Name)
		require.Len(t, node.AllOf[1].Properties, 1)
		assert.Equal(t, "b", node.AllOf[1].Properties[0].Name)
	})

	t.Run("anyOf children are visited", func(t *testing.T) {
		node := parseJSON(t, `{
			"anyOf": [
				{
					"type": "object",
					"properties": {
						"x": {"type": "string"}
					}
				},
				{
					"type": "object",
					"properties": {
						"y": {"type": "integer"}
					}
				}
			]
		}`)

		Normalize(node)

		require.Len(t, node.AnyOf, 2)
		require.Len(t, node.AnyOf[0].Properties, 1)
		assert.Equal(t, "x", node.AnyOf[0].Properties[0].Name)
		require.Len(t, node.AnyOf[1].Properties, 1)
		assert.Equal(t, "y", node.AnyOf[1].Properties[0].Name)
	})

	t.Run("oneOf children are visited", func(t *testing.T) {
		node := parseJSON(t, `{
			"oneOf": [
				{"type": "string"},
				{"type": "integer"}
			]
		}`)

		Normalize(node)

		require.Len(t, node.OneOf, 2)
		assert.True(t, node.OneOf[0].Type.Has("string"))
		assert.True(t, node.OneOf[1].Type.Has("integer"))
	})

	t.Run("nested items are visited", func(t *testing.T) {
		node := parseJSON(t, `{
			"type": "array",
			"items": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"deep": {"type": "boolean"}
					}
				}
			}
		}`)

		Normalize(node)

		require.NotNil(t, node.Items)
		require.NotNil(t, node.Items.Items)
		inner := node.Items.Items
		require.Len(t, inner.Properties, 1)
		assert.Equal(t, "deep", inner.Properties[0].Name)
		assert.True(t, inner.Properties[0].Schema.Type.Has("boolean"))
	})

	t.Run("nil node does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Normalize(nil)
		})
	})
}

func TestNormalize_Integration(t *testing.T) {
	testdataDir := filepath.Join("..", "..", "..", "testdata", "schemas")

	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		t.Skipf("testdata directory not found at %s", testdataDir)
	}

	t.Run("ecommerce order schema", func(t *testing.T) {
		ns, err := parse.ParseFile(filepath.Join(testdataDir, "complex", "ecommerce_order.json"))
		require.NoError(t, err)

		node := ns.Schema
		draft := DetectDraft(node)
		assert.Equal(t, Draft07, draft)

		Normalize(node)

		assert.True(t, node.Type.Has("object"))
		require.NotEmpty(t, node.Properties)

		propNames := make(map[string]bool)
		for _, p := range node.Properties {
			propNames[p.Name] = true
		}
		assert.True(t, propNames["orderId"])
		assert.True(t, propNames["customerId"])
		assert.True(t, propNames["status"])
		assert.True(t, propNames["items"])

		require.NotEmpty(t, node.Defs)
		defNames := make(map[string]bool)
		for _, d := range node.Defs {
			defNames[d.Name] = true
		}
		assert.True(t, defNames["Money"])
		assert.True(t, defNames["Address"])
		assert.True(t, defNames["LineItem"])

		assert.Contains(t, node.Required, "orderId")
		assert.Contains(t, node.Required, "total")
	})

	t.Run("organization schema with refs", func(t *testing.T) {
		ns, err := parse.ParseFile(filepath.Join(testdataDir, "refs", "organization.json"))
		require.NoError(t, err)

		node := ns.Schema
		Normalize(node)

		assert.True(t, node.Type.Has("object"))

		propNames := make(map[string]bool)
		for _, p := range node.Properties {
			propNames[p.Name] = true
		}
		assert.True(t, propNames["id"])
		assert.True(t, propNames["name"])
		assert.True(t, propNames["headquarters"])
		assert.True(t, propNames["branches"])

		defNames := make(map[string]bool)
		for _, d := range node.Defs {
			defNames[d.Name] = true
		}
		assert.True(t, defNames["Country"])
		assert.True(t, defNames["Address"])
		assert.True(t, defNames["Office"])

		var officeDef *parse.SchemaNode
		for _, d := range node.Defs {
			if d.Name == "Office" {
				officeDef = d.Schema
				break
			}
		}
		require.NotNil(t, officeDef)
		assert.True(t, officeDef.Type.Has("object"))
		require.NotEmpty(t, officeDef.Properties)
	})

	t.Run("notification schema with anyOf", func(t *testing.T) {
		ns, err := parse.ParseFile(filepath.Join(testdataDir, "anyof", "notification.json"))
		require.NoError(t, err)

		node := ns.Schema
		Normalize(node)

		assert.True(t, node.Type.Has("object"))

		var channelProp *parse.SchemaNode
		for _, p := range node.Properties {
			if p.Name == "channel" {
				channelProp = p.Schema
				break
			}
		}
		require.NotNil(t, channelProp)
		require.Len(t, channelProp.AnyOf, 3)

		for _, variant := range channelProp.AnyOf {
			assert.True(t, variant.Type.Has("object"))
			assert.NotEmpty(t, variant.Properties)
		}
	})

	t.Run("document schema with allOf", func(t *testing.T) {
		ns, err := parse.ParseFile(filepath.Join(testdataDir, "allof", "document.json"))
		require.NoError(t, err)

		node := ns.Schema
		Normalize(node)

		require.Len(t, node.AllOf, 5)

		defNames := make(map[string]bool)
		for _, d := range node.Defs {
			defNames[d.Name] = true
		}
		assert.True(t, defNames["Entity"])
		assert.True(t, defNames["Auditable"])
	})

	t.Run("2020-12 schema preserves structure", func(t *testing.T) {
		ns, err := parse.ParseFile(filepath.Join(testdataDir, "basic", "any_type.json"))
		require.NoError(t, err)

		node := ns.Schema
		draft := DetectDraft(node)
		assert.Equal(t, Draft202012, draft)

		Normalize(node)

		assert.True(t, node.Type.Has("object"))
		require.Len(t, node.Properties, 2)

		propNames := make(map[string]bool)
		for _, p := range node.Properties {
			propNames[p.Name] = true
		}
		assert.True(t, propNames["name"])
		assert.True(t, propNames["data"])
		assert.Equal(t, []string{"name", "data"}, node.Required)
	})

	t.Run("all testdata schemas parse and normalize without panic", func(t *testing.T) {
		schemas, err := parse.ParseDir(testdataDir)
		require.NoError(t, err)
		require.NotEmpty(t, schemas)

		for _, ns := range schemas {
			t.Run(ns.Path, func(t *testing.T) {
				assert.NotPanics(t, func() {
					Normalize(ns.Schema)
				})
			})
		}
	})
}
