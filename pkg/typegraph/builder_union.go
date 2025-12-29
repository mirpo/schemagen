package typegraph

import (
	"github.com/kaptinlin/jsonschema"
)

// buildUnion builds a union type from a schema with anyOf/oneOf.
func (b *Builder) buildUnion(typ *Type, schema *jsonschema.Schema) error {
	typ.Kind = KindUnion

	// Get union members (anyOf takes precedence over oneOf)
	members := schema.AnyOf
	if len(members) == 0 {
		members = schema.OneOf
	}

	typ.UnionMembers = make([]*TypeRef, 0, len(members))
	for _, memberSchema := range members {
		memberRef := b.buildTypeRef(memberSchema, "")
		typ.UnionMembers = append(typ.UnionMembers, memberRef)
	}

	return nil
}
