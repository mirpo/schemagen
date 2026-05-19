package zod

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/v2/graph"
)

func (e *Emitter) typeRefToZod(ref *graph.TypeRef, field *graph.Field) string {
	if ref == nil {
		return "z.unknown()"
	}

	var zodType string

	switch ref.Kind {
	case graph.KindRef:
		zodType = ref.TypeName + "Schema"

	case graph.KindPrimitive:
		zodType = e.primitiveToZod(ref.Primitive, field)

	case graph.KindArray:
		itemType := e.typeRefToZod(ref.ItemType, nil)
		zodType = fmt.Sprintf("z.array(%s)", itemType)
		if field != nil {
			zodType += arrayConstraints(field)
		}

	case graph.KindMap:
		valueType := e.typeRefToZod(ref.ValueType, nil)
		zodType = fmt.Sprintf("z.record(z.string(), %s)", valueType)

	case graph.KindEnum:
		zodType = e.enumValuesToZod(ref.EnumValues)

	case graph.KindUnion:
		members := make([]string, len(ref.UnionMembers))
		for i, member := range ref.UnionMembers {
			members[i] = e.typeRefToZod(member, nil)
		}
		zodType = fmt.Sprintf("z.union([%s])", strings.Join(members, ", "))

	case graph.KindInterface:
		if len(ref.ObjectFields) > 0 {
			zodType = e.generateInlineObject(ref.ObjectFields)
		} else {
			zodType = "z.record(z.string(), z.unknown())"
		}

	default:
		zodType = "z.unknown()"
	}

	if ref.Nullable {
		zodType += ".nullable()"
	}

	return zodType
}

func (e *Emitter) primitiveToZod(p graph.PrimitiveKind, field *graph.Field) string {
	switch p {
	case graph.PrimEmail:
		return "z.email()" + stringConstraints(field)
	case graph.PrimURI:
		return "z.url()" + stringConstraints(field)
	case graph.PrimUUID:
		return "z.uuid()"
	case graph.PrimIPv4:
		return "z.ipv4()"
	case graph.PrimIPv6:
		return "z.ipv6()"
	case graph.PrimDateTime:
		if e.config.CoerceDates {
			return "z.coerce.date()"
		}
		return "z.iso.datetime()"
	case graph.PrimDate:
		return "z.iso.date()"
	case graph.PrimTime:
		return "z.iso.time()"
	case graph.PrimString, graph.PrimHostname:
		return "z.string()" + stringConstraints(field)
	case graph.PrimInt, graph.PrimInt32, graph.PrimInt64:
		return "z.int()" + numberConstraints(field)
	case graph.PrimFloat32, graph.PrimFloat64:
		return "z.number()" + numberConstraints(field)
	case graph.PrimBool:
		return "z.boolean()"
	default:
		return "z.unknown()"
	}
}

func (e *Emitter) enumValuesToZod(values []graph.EnumValue) string {
	if len(values) == 0 {
		return "z.never()"
	}

	category := graph.AnalyzeEnumValues(values)

	if category.AllStrings {
		strValues := make([]string, len(values))
		for i, v := range values {
			strValues[i] = fmt.Sprintf("%q", v.Value.(string))
		}
		return fmt.Sprintf("z.enum([%s])", strings.Join(strValues, ", "))
	}

	literals := make([]string, len(values))
	for i, v := range values {
		literals[i] = formatZodLiteral(v.Value)
	}
	return fmt.Sprintf("z.union([%s])", strings.Join(literals, ", "))
}

func (e *Emitter) generateInlineObject(fields []*graph.Field) string {
	var sb strings.Builder

	objectFunc := "z.object"
	if e.config.Strict {
		objectFunc = "z.strictObject"
	}

	sb.WriteString(objectFunc + "({\n")

	for _, field := range fields {
		fmt.Fprintf(&sb, "    %s,\n", e.generateField(field))
	}

	sb.WriteString("  })")

	return sb.String()
}
