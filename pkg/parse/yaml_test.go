package parse

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseYAMLSimple(t *testing.T) {
	input := `
type: object
title: Test
properties:
  name:
    type: string
  age:
    type: integer
required:
  - name
`
	node, err := ParseYAML(strings.NewReader(input))
	require.NoError(t, err)

	assert.Equal(t, "Test", node.Title)
	assert.True(t, node.Type.Has("object"))
	require.Len(t, node.Properties, 2)

	propNames := make(map[string]bool)
	for _, p := range node.Properties {
		propNames[p.Name] = true
	}
	assert.True(t, propNames["name"])
	assert.True(t, propNames["age"])
	assert.Equal(t, []string{"name"}, node.Required)
}

func TestParseYAMLWithEnum(t *testing.T) {
	input := `
type: string
enum:
  - debug
  - info
  - warn
  - error
`
	node, err := ParseYAML(strings.NewReader(input))
	require.NoError(t, err)
	assert.True(t, node.IsEnum())
	assert.Len(t, node.Enum, 4)
}

func TestParseYAMLWithRef(t *testing.T) {
	input := `
type: object
properties:
  address:
    $ref: "#/$defs/Address"
`
	node, err := ParseYAML(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, node.Properties, 1)
	assert.Equal(t, "#/$defs/Address", node.Properties[0].Schema.Ref)
}

func TestParseYAMLConfigFile(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "schemas", "yaml", "config.yaml")
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	node, err := ParseYAML(f)
	require.NoError(t, err)

	assert.Equal(t, "Config", node.Title)
	assert.True(t, node.Type.Has("object"))
	require.Len(t, node.Properties, 3)

	propMap := make(map[string]*SchemaNode)
	for _, p := range node.Properties {
		propMap[p.Name] = p.Schema
	}
	assert.Contains(t, propMap, "appName")
	assert.Contains(t, propMap, "version")
	assert.Contains(t, propMap, "settings")
	assert.True(t, propMap["settings"].IsRef())
}

func TestParseYAMLSettingsFile(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "schemas", "yaml", "settings.yaml")
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	node, err := ParseYAML(f)
	require.NoError(t, err)

	assert.Equal(t, "Settings", node.Title)
	require.Len(t, node.Properties, 3)
	assert.Equal(t, "debug", node.Properties[0].Name)
	assert.Equal(t, "logLevel", node.Properties[1].Name)
	assert.Equal(t, "maxConnections", node.Properties[2].Name)

	logLevel := node.Properties[1].Schema
	assert.True(t, logLevel.IsEnum())
	assert.Len(t, logLevel.Enum, 4)
}

func TestParseYAMLSimpleTestdata(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "schemas", "basic", "simple.yaml")
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	node, err := ParseYAML(f)
	require.NoError(t, err)

	assert.Equal(t, "Simple", node.Title)
	assert.Equal(t, "A simple test schema", node.Description)
	require.Len(t, node.Properties, 2)
	assert.Equal(t, "name", node.Properties[0].Name)
	assert.Equal(t, "value", node.Properties[1].Name)
}

func TestParseYAMLKeyOrderNotGuaranteed(t *testing.T) {
	input := `
type: object
properties:
  zebra:
    type: string
  alpha:
    type: string
  mango:
    type: integer
  beta:
    type: boolean
`
	node, err := ParseYAML(strings.NewReader(input))
	require.NoError(t, err)

	expected := map[string]bool{"zebra": true, "alpha": true, "mango": true, "beta": true}
	require.Len(t, node.Properties, len(expected))
	for _, p := range node.Properties {
		assert.True(t, expected[p.Name], "unexpected property %q", p.Name)
	}
}

func TestParseYAMLInvalid(t *testing.T) {
	input := `
invalid: yaml: [
  missing bracket
`
	_, err := ParseYAML(strings.NewReader(input))
	assert.Error(t, err)
}
