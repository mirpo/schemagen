package golang

import (
	"github.com/mirpo/schemagen/pkg/v2/graph"
)

func (g *Generator) fieldGoType(field *graph.Field) string {
	typeStr := g.typeRefToGoType(field.Type)

	if !field.Required && g.config.UsePointers {
		return "*" + typeStr
	}

	return typeStr
}

func (g *Generator) typeRefToGoType(ref *graph.TypeRef) string {
	if ref == nil {
		return "any"
	}

	if ref.TypeName != "" {
		return ref.TypeName
	}

	switch ref.Kind {
	case graph.KindPrimitive, graph.KindEnum:
		return primitiveToGo(ref.Primitive)
	case graph.KindUnion:
		return "any"
	case graph.KindArray:
		return "[]" + g.typeRefToGoType(ref.ItemType)
	case graph.KindMap:
		return "map[string]" + g.typeRefToGoType(ref.ValueType)
	case graph.KindInterface:
		return "any"
	default:
		return "any"
	}
}

func primitiveToGo(p graph.PrimitiveKind) string {
	switch p {
	case graph.PrimString, graph.PrimEmail, graph.PrimURI,
		graph.PrimHostname, graph.PrimIPv4, graph.PrimIPv6, graph.PrimTime:
		return "string"
	case graph.PrimInt:
		return "int"
	case graph.PrimInt32:
		return "int32"
	case graph.PrimInt64:
		return "int64"
	case graph.PrimFloat32:
		return "float32"
	case graph.PrimFloat64:
		return "float64"
	case graph.PrimBool:
		return "bool"
	case graph.PrimDateTime, graph.PrimDate:
		return "time.Time"
	case graph.PrimUUID:
		return "uuid.UUID"
	default:
		return "any"
	}
}

func importsForPrimitive(p graph.PrimitiveKind) string {
	switch p {
	case graph.PrimDateTime, graph.PrimDate:
		return "time"
	case graph.PrimUUID:
		return "github.com/google/uuid"
	default:
		return ""
	}
}

func fieldHasValidation(field *graph.Field) bool {
	if field.Required || field.HasConstraints() {
		return true
	}
	if field.Type != nil {
		switch field.Type.Format {
		case graph.FormatEmail, graph.FormatURI, graph.FormatURL, graph.FormatUUID:
			return true
		}
		if len(field.Type.EnumValues) > 0 {
			return true
		}
	}
	return false
}

func (g *Generator) scanTypeForImports(typ *graph.Type) {
	switch typ.Kind {
	case graph.KindStruct:
		for _, field := range typ.Fields {
			g.scanTypeRefForImports(field.Type)
			if fieldHasValidation(field) {
				g.imports[validatorImport] = "_"
			}
		}
	case graph.KindPrimitive:
		if imp := importsForPrimitive(typ.Primitive); imp != "" {
			g.imports[imp] = ""
		}
	}
}

func (g *Generator) scanTypeRefForImports(ref *graph.TypeRef) {
	ref.Walk(func(r *graph.TypeRef) {
		if imp := importsForPrimitive(r.Primitive); imp != "" {
			g.imports[imp] = ""
		}
	})
}

func (g *Generator) resetImports() {
	g.imports = make(map[string]string)
}
