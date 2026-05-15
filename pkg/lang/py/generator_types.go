package py

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/common"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

func (g *Generator) checkTypeRefForImports(ref *typegraph.TypeRef) {
	if ref == nil {
		g.needsAny = true
		return
	}

	if imp := importsForPrimitive(ref.Primitive); imp != "" {
		g.imports[imp] = true
	}

	if ref.Kind == typegraph.KindEnum && len(ref.EnumValues) > 0 {
		g.imports["typing_literal"] = true
	}

	switch ref.Kind {
	case typegraph.KindInterface:
		g.needsAny = true
	case typegraph.KindPrimitive:
		if ref.Primitive == typegraph.PrimUnknown {
			g.needsAny = true
		}
	case typegraph.KindMap:
		g.checkTypeRefForImports(ref.ValueType)
	case typegraph.KindUnion:
		for _, member := range ref.UnionMembers {
			g.checkTypeRefForImports(member)
		}
	case typegraph.KindArray:
		g.checkTypeRefForImports(ref.ItemType)
	}
}

func (g *Generator) typeRefToPython(ref *typegraph.TypeRef, optional bool) string {
	if ref == nil {
		return "Any"
	}

	var pyType string

	switch ref.Kind {
	case typegraph.KindRef:
		if ref.TypeName != "" {
			pyType = ref.TypeName
		} else {
			pyType = "Any"
		}
	case typegraph.KindEnum:
		if len(ref.EnumValues) > 0 {
			g.imports["typing_literal"] = true
			literals := make([]string, 0, len(ref.EnumValues))
			for _, val := range ref.EnumValues {
				switch val.(type) {
				case string, float64, int, int64, bool, nil:
					literals = append(literals, common.PyLiterals.FormatValue(val))
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
	case typegraph.KindUnion:
		if len(ref.UnionMembers) > 0 {
			memberTypes := make([]string, len(ref.UnionMembers))
			for i, member := range ref.UnionMembers {
				memberTypes[i] = g.typeRefToPython(member, false)
			}
			pyType = strings.Join(memberTypes, " | ")
		} else {
			pyType = "Any"
		}
	case typegraph.KindPrimitive:
		pyType = g.primitiveToPython(ref.Primitive)
	case typegraph.KindArray:
		itemType := g.typeRefToPython(ref.ItemType, false)
		pyType = fmt.Sprintf("list[%s]", itemType)
	case typegraph.KindMap:
		valueType := g.typeRefToPython(ref.ValueType, false)
		pyType = fmt.Sprintf("dict[str, %s]", valueType)
	case typegraph.KindInterface:
		if len(ref.ObjectFields) > 0 {
			pyType = "dict[str, Any]"
			g.needsAny = true
		} else {
			pyType = "dict[str, Any]"
			g.needsAny = true
		}
	default:
		pyType = "Any"
	}

	if optional && !strings.Contains(pyType, " | None") {
		pyType += " | None"
	}

	return pyType
}

func (g *Generator) primitiveToPython(p typegraph.PrimitiveKind) string {
	if imp := importsForPrimitive(p); imp != "" {
		g.imports[imp] = true
	}

	switch p {
	case typegraph.PrimString, typegraph.PrimHostname, typegraph.PrimIPv4, typegraph.PrimIPv6, typegraph.PrimTime:
		return "str"
	case typegraph.PrimInt, typegraph.PrimInt32, typegraph.PrimInt64:
		return "int"
	case typegraph.PrimFloat32, typegraph.PrimFloat64:
		return "float"
	case typegraph.PrimBool:
		return "bool"
	case typegraph.PrimEmail:
		return "EmailStr"
	case typegraph.PrimURI:
		return "AnyUrl"
	case typegraph.PrimUUID:
		return "UUID"
	case typegraph.PrimDateTime, typegraph.PrimDate:
		return "datetime"
	default:
		return "Any"
	}
}

func importsForPrimitive(p typegraph.PrimitiveKind) string {
	switch p {
	case typegraph.PrimEmail:
		return "pydantic_email"
	case typegraph.PrimURI:
		return "pydantic_url"
	case typegraph.PrimUUID:
		return "uuid"
	case typegraph.PrimDateTime, typegraph.PrimDate:
		return "datetime"
	default:
		return ""
	}
}
