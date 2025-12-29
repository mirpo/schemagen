package typegraph

import (
	"github.com/kaptinlin/jsonschema"
)

// BuildUnion builds a union type from a schema with anyOf/oneOf (implements TypeBuilder).
func (b *Builder) BuildUnion(typ *Type, schema *jsonschema.Schema) error {
	typ.Kind = KindUnion

	// Get union members (anyOf takes precedence over oneOf)
	members := schema.AnyOf
	if len(members) == 0 {
		members = schema.OneOf
	}

	typ.UnionMembers = make([]*TypeRef, 0, len(members))
	for _, memberSchema := range members {
		memberRef := b.BuildTypeRef(memberSchema, "")
		typ.UnionMembers = append(typ.UnionMembers, memberRef)
	}

	return nil
}
