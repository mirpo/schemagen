package ts

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/common"
	"github.com/mirpo/schemagen/pkg/lang/tscommon"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

func (g *Generator) anyType() string {
	if g.config.UnknownAny {
		return "unknown"
	}
	return "any"
}

func (g *Generator) typeRefToTS(ref *typegraph.TypeRef) string {
	if ref == nil {
		return g.anyType()
	}

	var tsType string

	switch ref.Kind {
	case typegraph.KindRef:
		if ref.TypeName != "" {
			tsType = ref.TypeName
		} else {
			tsType = g.anyType()
		}
	case typegraph.KindEnum:
		if len(ref.EnumValues) > 0 {
			literals := make([]string, 0, len(ref.EnumValues))
			for _, val := range ref.EnumValues {
				switch val.(type) {
				case string, float64, int, int64, bool, nil:
					literals = append(literals, common.TSLiterals.FormatValue(val))
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
	case typegraph.KindUnion:
		if len(ref.UnionMembers) > 0 {
			memberTypes := make([]string, len(ref.UnionMembers))
			for i, member := range ref.UnionMembers {
				memberTypes[i] = g.typeRefToTS(member)
			}
			tsType = strings.Join(memberTypes, " | ")
		} else {
			tsType = g.anyType()
		}
	case typegraph.KindPrimitive:
		tsType = g.primitiveToTS(ref.Primitive)
	case typegraph.KindArray:
		itemType := g.typeRefToTS(ref.ItemType)
		tsType = itemType + "[]"
	case typegraph.KindMap:
		valueType := g.typeRefToTS(ref.ValueType)
		tsType = fmt.Sprintf("Record<string, %s>", valueType)
	case typegraph.KindInterface:
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

func (g *Generator) primitiveToTS(p typegraph.PrimitiveKind) string {
	switch p {
	case typegraph.PrimString, typegraph.PrimEmail, typegraph.PrimURI,
		typegraph.PrimHostname, typegraph.PrimIPv4, typegraph.PrimIPv6,
		typegraph.PrimTime, typegraph.PrimDateTime, typegraph.PrimDate,
		typegraph.PrimUUID:
		return "string"
	case typegraph.PrimInt, typegraph.PrimInt32, typegraph.PrimInt64,
		typegraph.PrimFloat32, typegraph.PrimFloat64:
		return "number"
	case typegraph.PrimBool:
		return "boolean"
	default:
		return g.anyType()
	}
}

func (g *Generator) generateAnonymousInterface(fields []*typegraph.Field) string {
	var sb strings.Builder
	sb.WriteString("{\n")

	for _, field := range fields {
		format := ""
		if field.Type != nil {
			format = field.Type.Format
		}
		tscommon.WriteJSDocWithFormat(&sb, "    ", field.Description, format)

		tsType := g.typeRefToTS(field.Type)
		optional := ""
		if !field.Required {
			optional = "?"
		}

		propName := field.JSONName
		if tscommon.NeedsQuoting(propName) {
			propName = fmt.Sprintf("%q", propName)
		}

		fmt.Fprintf(&sb, "    %s%s: %s;\n", propName, optional, tsType)
	}

	sb.WriteString("  }")
	return sb.String()
}
