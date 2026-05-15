package typegraph

import (
	"github.com/kaptinlin/jsonschema"
)

func (b *Builder) BuildStruct(ctx *BuildContext, typ *Type, schema *jsonschema.Schema) error {
	return b.structBuilder.Build(ctx, typ, schema)
}

func (b *Builder) shouldExtractInlineObject(schema *jsonschema.Schema) bool {
	return b.typeRefBuilder.ShouldExtractInlineObject(schema)
}

func (b *Builder) extractInlineObjectType(ctx *BuildContext, baseName string, schema *jsonschema.Schema) *Type {
	return b.typeRefBuilder.ExtractInlineObjectType(ctx, baseName, schema)
}

func (b *Builder) BuildFieldsFromProperties(ctx *BuildContext, schema *jsonschema.Schema, orderPath string) []*Field {
	return b.typeRefBuilder.BuildFieldsFromProperties(ctx, schema, orderPath)
}
