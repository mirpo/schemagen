package ts

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/common"
	"github.com/mirpo/schemagen/pkg/v2/graph"
)

func (g *Generator) anyType() string {
	if g.config.UnknownAny {
		return "unknown"
	}
	return "any"
}

func (g *Generator) typeRefToTS(ref *graph.TypeRef) string {
	if ref == nil {
		return g.anyType()
	}

	var tsType string

	switch ref.Kind {
	case graph.KindRef:
		if ref.TypeName != "" {
			tsType = ref.TypeName
		} else {
			tsType = g.anyType()
		}
	case graph.KindEnum:
		if len(ref.EnumValues) > 0 {
			literals := make([]string, 0, len(ref.EnumValues))
			for _, val := range ref.EnumValues {
				switch val.Value.(type) {
				case string, float64, int, int64, bool, nil:
					literals = append(literals, common.TSLiterals.FormatValue(val.Value))
				default:
				}
			}
			if len(literals) > 0 {
				tsType = strings.Join(literals, " | ")
			} else {
				tsType = g.anyType()
			}
		} else {
			tsType = g.anyType()
		}
	case graph.KindUnion:
		if len(ref.UnionMembers) > 0 {
			memberTypes := make([]string, len(ref.UnionMembers))
			for i, member := range ref.UnionMembers {
				memberTypes[i] = g.typeRefToTS(member)
			}
			tsType = strings.Join(memberTypes, " | ")
		} else {
			tsType = g.anyType()
		}
	case graph.KindPrimitive:
		tsType = g.primitiveToTS(ref.Primitive)
	case graph.KindArray:
		itemType := g.typeRefToTS(ref.ItemType)
		tsType = itemType + "[]"
	case graph.KindMap:
		valueType := g.typeRefToTS(ref.ValueType)
		tsType = fmt.Sprintf("Record<string, %s>", valueType)
	case graph.KindInterface:
		if len(ref.ObjectFields) > 0 {
			tsType = g.generateAnonymousInterface(ref.ObjectFields)
		} else {
			tsType = g.anyType()
		}
	default:
		tsType = g.anyType()
	}

	if ref.Nullable {
		tsType += " | null"
	}

	return tsType
}

func (g *Generator) primitiveToTS(p graph.PrimitiveKind) string {
	switch p {
	case graph.PrimString, graph.PrimEmail, graph.PrimURI,
		graph.PrimHostname, graph.PrimIPv4, graph.PrimIPv6,
		graph.PrimTime, graph.PrimDateTime, graph.PrimDate,
		graph.PrimUUID:
		return "string"
	case graph.PrimInt, graph.PrimInt32, graph.PrimInt64,
		graph.PrimFloat32, graph.PrimFloat64:
		return "number"
	case graph.PrimBool:
		return "boolean"
	default:
		return g.anyType()
	}
}

func (g *Generator) generateAnonymousInterface(fields []*graph.Field) string {
	var sb strings.Builder
	sb.WriteString("{\n")

	for _, field := range fields {
		g.renderField(&sb, field, "    ", true)
	}

	sb.WriteString("  }")
	return sb.String()
}
