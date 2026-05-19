package py

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/graph"
	"github.com/mirpo/schemagen/pkg/render"
)

func (g *Generator) checkTypeRefForImports(ref *graph.TypeRef) {
	if ref == nil {
		g.needsAny = true
		return
	}

	ref.Walk(func(r *graph.TypeRef) {
		if imp := importsForPrimitive(r.Primitive); imp != "" {
			g.imports[imp] = true
		}
		if r.Kind == graph.KindEnum && len(r.EnumValues) > 0 {
			g.imports["typing_literal"] = true
		}
		if r.Kind == graph.KindInterface || (r.Kind == graph.KindPrimitive && r.Primitive == graph.PrimUnknown) {
			g.needsAny = true
		}
	})
}

func (g *Generator) typeRefToPython(ref *graph.TypeRef, optional bool) string {
	if ref == nil {
		return "Any"
	}

	var pyType string

	switch ref.Kind {
	case graph.KindRef:
		if ref.TypeName != "" {
			pyType = ref.TypeName
		} else {
			pyType = "Any"
		}
	case graph.KindEnum:
		if len(ref.EnumValues) > 0 {
			literals := make([]string, 0, len(ref.EnumValues))
			for _, val := range ref.EnumValues {
				switch val.Value.(type) {
				case string, float64, int, int64, bool, nil:
					literals = append(literals, render.PyLiterals.FormatValue(val.Value))
				default:
				}
			}
			if len(literals) > 0 {
				pyType = fmt.Sprintf("Literal[%s]", strings.Join(literals, ", "))
			} else {
				pyType = "Any"
			}
		} else {
			pyType = "Any"
		}
	case graph.KindUnion:
		if len(ref.UnionMembers) > 0 {
			memberTypes := make([]string, len(ref.UnionMembers))
			for i, member := range ref.UnionMembers {
				memberTypes[i] = g.typeRefToPython(member, false)
			}
			pyType = strings.Join(memberTypes, " | ")
		} else {
			pyType = "Any"
		}
	case graph.KindPrimitive:
		pyType = g.primitiveToPython(ref.Primitive)
	case graph.KindArray:
		itemType := g.typeRefToPython(ref.ItemType, false)
		pyType = fmt.Sprintf("list[%s]", itemType)
	case graph.KindMap:
		valueType := g.typeRefToPython(ref.ValueType, false)
		pyType = fmt.Sprintf("dict[str, %s]", valueType)
	case graph.KindInterface:
		pyType = "dict[str, Any]"
	default:
		pyType = "Any"
	}

	if optional && !strings.Contains(pyType, " | None") {
		pyType += " | None"
	}

	return pyType
}

func (g *Generator) primitiveToPython(p graph.PrimitiveKind) string {
	switch p {
	case graph.PrimString, graph.PrimHostname, graph.PrimIPv4, graph.PrimIPv6, graph.PrimTime:
		return "str"
	case graph.PrimInt, graph.PrimInt32, graph.PrimInt64:
		return "int"
	case graph.PrimFloat32, graph.PrimFloat64:
		return "float"
	case graph.PrimBool:
		return "bool"
	case graph.PrimEmail:
		return "EmailStr"
	case graph.PrimURI:
		return "AnyUrl"
	case graph.PrimUUID:
		return "UUID"
	case graph.PrimDateTime, graph.PrimDate:
		return "datetime"
	default:
		return "Any"
	}
}

func importsForPrimitive(p graph.PrimitiveKind) string {
	switch p {
	case graph.PrimEmail:
		return "pydantic_email"
	case graph.PrimURI:
		return "pydantic_url"
	case graph.PrimUUID:
		return "uuid"
	case graph.PrimDateTime, graph.PrimDate:
		return "datetime"
	default:
		return ""
	}
}
