package typegraph

import (
	"github.com/kaptinlin/jsonschema"
)

// BuildStruct builds a struct/object type from a schema (implements TypeBuilder).
func (b *Builder) BuildStruct(typ *Type, schema *jsonschema.Schema) error {
	// Sync walker's state to structBuilder
	b.structBuilder.SetCurrentOrder(b.walker.CurrentOrder())
	b.structBuilder.SetCurrentPath(b.walker.CurrentPath())

	// Delegate to structBuilder
	return b.structBuilder.Build(typ, schema)
}

// shouldExtractInlineObject checks if a schema represents an inline object that should be extracted.
// Delegates to typeRefBuilder.
func (b *Builder) shouldExtractInlineObject(schema *jsonschema.Schema) bool {
	return b.typeRefBuilder.ShouldExtractInlineObject(schema)
}

// extractInlineObjectType extracts an inline object as a separate Type.
// Delegates to typeRefBuilder.
func (b *Builder) extractInlineObjectType(baseName string, schema *jsonschema.Schema) *Type {
	return b.typeRefBuilder.ExtractInlineObjectType(baseName, schema)
}

// BuildFieldsFromProperties extracts fields from schema properties for inline objects (implements FieldBuilder).
// Delegates to typeRefBuilder.
func (b *Builder) BuildFieldsFromProperties(schema *jsonschema.Schema, orderPath string) []*Field {
	return b.typeRefBuilder.BuildFieldsFromProperties(schema, orderPath)
}
