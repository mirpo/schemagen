package typegraph

import (
	"github.com/kaptinlin/jsonschema"
)

func (b *Builder) BuildStruct(ctx *BuildContext, typ *Type, schema *jsonschema.Schema) error {
	return b.structBuilder.Build(ctx, typ, schema)
}
