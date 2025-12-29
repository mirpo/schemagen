package typegraph

import (
	"testing"

	"github.com/kaptinlin/jsonschema"
	"github.com/stretchr/testify/assert"
)

func TestRefResolver_ExtractTypeName_InternalDefs(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	resolver := NewRefResolver(compiler)

	tests := []struct {
		ref      string
		expected string
	}{
		{"#/$defs/IdObject", "IdObject"},
		{"#/$defs/person", "Person"},
		{"#/$defs/my-type", "MyType"},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			result := resolver.ExtractTypeName(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRefResolver_ExtractTypeName_RootSelfReference(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	resolver := NewRefResolver(compiler)

	// Without currentSchema set, should return "Schema"
	result := resolver.ExtractTypeName("#")
	assert.Equal(t, "Schema", result)

	// With currentSchema set with a title
	title := "MyRootType"
	resolver.SetCurrentSchema(&jsonschema.Schema{Title: &title})
	result = resolver.ExtractTypeName("#")
	assert.Equal(t, "MyRootType", result)
}

func TestRefResolver_ExtractTypeName_FilenameOnly(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	resolver := NewRefResolver(compiler)

	tests := []struct {
		ref      string
		expected string
	}{
		{"./settings.yaml", "Settings"},
		{"header.json", "Header"},
		{"./user-settings.yaml", "UserSettings"},
		{"../events/config.json", "Config"},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			result := resolver.ExtractTypeName(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRefResolver_ExtractTypeName_ExternalWithCompiler(t *testing.T) {
	compiler := jsonschema.NewCompiler()

	// Register a schema with a title
	title := "EventHeader"
	compiler.SetSchema("header.json", &jsonschema.Schema{Title: &title})

	resolver := NewRefResolver(compiler)

	result := resolver.ExtractTypeName("header.json")
	assert.Equal(t, "EventHeader", result)
}

func TestRefResolver_DeriveTypeName_FromTitle(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	resolver := NewRefResolver(compiler)

	title := "UserProfile"
	schema := &jsonschema.Schema{Title: &title}

	result := resolver.DeriveTypeName(schema, "")
	assert.Equal(t, "UserProfile", result)
}

func TestRefResolver_DeriveTypeName_TitleNeedsPascalCase(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	resolver := NewRefResolver(compiler)

	title := "user-profile"
	schema := &jsonschema.Schema{Title: &title}

	result := resolver.DeriveTypeName(schema, "")
	assert.Equal(t, "UserProfile", result)
}

func TestRefResolver_DeriveTypeName_FromURI(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	resolver := NewRefResolver(compiler)

	schema := &jsonschema.Schema{}

	result := resolver.DeriveTypeName(schema, "payloads/subscribe.json")
	assert.Equal(t, "Subscribe", result)
}

func TestRefResolver_DeriveTypeName_NoTitleNoURI(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	resolver := NewRefResolver(compiler)

	schema := &jsonschema.Schema{}

	result := resolver.DeriveTypeName(schema, "")
	assert.Equal(t, "Unknown", result)
}
