package golang

import (
	"github.com/mirpo/schemagen/pkg/typegraph"
)

func (g *Generator) fieldGoType(field *typegraph.Field) string {
	typeStr := g.typeRefToGoType(field.Type)

	if !field.Required && g.config.UsePointers {
		return "*" + typeStr
	}

	return typeStr
}

func (g *Generator) typeRefToGoType(ref *typegraph.TypeRef) string {
	if ref == nil {
		return "any"
	}

	if ref.TypeName != "" {
		return ref.TypeName
	}

	switch ref.Kind {
	case typegraph.KindPrimitive, typegraph.KindEnum:
		return primitiveToGo(ref.Primitive)
	case typegraph.KindUnion:
		return "any"
	case typegraph.KindArray:
		return "[]" + g.typeRefToGoType(ref.ItemType)
	case typegraph.KindMap:
		return "map[string]" + g.typeRefToGoType(ref.ValueType)
	case typegraph.KindInterface:
		return "any"
	default:
		return "any"
	}
}

func primitiveToGo(p typegraph.PrimitiveKind) string {
	switch p {
	case typegraph.PrimString, typegraph.PrimEmail, typegraph.PrimURI,
		typegraph.PrimHostname, typegraph.PrimIPv4, typegraph.PrimIPv6, typegraph.PrimTime:
		return "string"
	case typegraph.PrimInt:
		return "int"
	case typegraph.PrimInt32:
		return "int32"
	case typegraph.PrimInt64:
		return "int64"
	case typegraph.PrimFloat32:
		return "float32"
	case typegraph.PrimFloat64:
		return "float64"
	case typegraph.PrimBool:
		return "bool"
	case typegraph.PrimDateTime, typegraph.PrimDate:
		return "time.Time"
	case typegraph.PrimUUID:
		return "uuid.UUID"
	default:
		return "any"
	}
}

func importsForPrimitive(p typegraph.PrimitiveKind) string {
	switch p {
	case typegraph.PrimDateTime, typegraph.PrimDate:
		return "time"
	case typegraph.PrimUUID:
		return "github.com/google/uuid"
	default:
		return ""
	}
}

func fieldHasValidation(field *typegraph.Field) bool {
	if field.Required || field.HasConstraints() {
		return true
	}
	if field.Type != nil {
		switch field.Type.Format {
		case "email", "uri", "url", "uuid":
			return true
		}
		if len(field.Type.EnumValues) > 0 {
			return true
		}
	}
	return false
}

func (g *Generator) scanTypeForImports(typ *typegraph.Type) {
	switch typ.Kind {
	case typegraph.KindStruct:
		for _, field := range typ.Fields {
			g.scanTypeRefForImports(field.Type)
			if fieldHasValidation(field) {
				g.imports[validatorImport] = "_"
			}
		}
	case typegraph.KindPrimitive:
		if imp := importsForPrimitive(typ.Primitive); imp != "" {
			g.imports[imp] = ""
		}
	}
}

func (g *Generator) scanTypeRefForImports(ref *typegraph.TypeRef) {
	ref.Walk(func(r *typegraph.TypeRef) {
		if imp := importsForPrimitive(r.Primitive); imp != "" {
			g.imports[imp] = ""
		}
	})
}

func (g *Generator) resetImports() {
	g.imports = make(map[string]string)
}
