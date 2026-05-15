package typegraph

import (
	"github.com/kaptinlin/jsonschema"
)

func (b *Builder) BuildUnion(ctx *BuildContext, typ *Type, schema *jsonschema.Schema) error {
	typ.Kind = KindUnion

	members := schema.AnyOf
	if len(members) == 0 {
		members = schema.OneOf
	}

	typ.UnionMembers = make([]*TypeRef, 0, len(members))
	for _, memberSchema := range members {
		memberRef := b.BuildTypeRef(ctx, memberSchema, "")
		typ.UnionMembers = append(typ.UnionMembers, memberRef)
	}

	return nil
}
