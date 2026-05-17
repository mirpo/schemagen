package zod

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/enumutil"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

func (e *Emitter) typeRefToZod(ref *typegraph.TypeRef, field *typegraph.Field) string {
	if ref == nil {
		return "z.unknown()"
	}

	var zodType string

	switch ref.Kind {
	case typegraph.KindRef:
		zodType = ref.TypeName + "Schema"

	case typegraph.KindPrimitive:
		zodType = e.primitiveToZod(ref.Primitive, field)

	case typegraph.KindArray:
		itemType := e.typeRefToZod(ref.ItemType, nil)
		zodType = fmt.Sprintf("z.array(%s)", itemType)
		if field != nil {
			zodType += arrayConstraints(field)
		}

	case typegraph.KindMap:
		valueType := e.typeRefToZod(ref.ValueType, nil)
		zodType = fmt.Sprintf("z.record(z.string(), %s)", valueType)

	case typegraph.KindEnum:
		zodType = e.enumValuesToZod(ref.EnumValues)

	case typegraph.KindUnion:
		members := make([]string, len(ref.UnionMembers))
		for i, member := range ref.UnionMembers {
			members[i] = e.typeRefToZod(member, nil)
		}
		zodType = fmt.Sprintf("z.union([%s])", strings.Join(members, ", "))

	case typegraph.KindInterface:
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

func (e *Emitter) primitiveToZod(p typegraph.PrimitiveKind, field *typegraph.Field) string {
	switch p {
	case typegraph.PrimEmail:
		return "z.email()" + stringConstraints(field)
	case typegraph.PrimURI:
		return "z.url()" + stringConstraints(field)
	case typegraph.PrimUUID:
		return "z.uuid()"
	case typegraph.PrimIPv4:
		return "z.ipv4()"
	case typegraph.PrimIPv6:
		return "z.ipv6()"
	case typegraph.PrimDateTime:
		if e.config.CoerceDates {
			return "z.coerce.date()"
		}
		return "z.iso.datetime()"
	case typegraph.PrimDate:
		return "z.iso.date()"
	case typegraph.PrimTime:
		return "z.iso.time()"
	case typegraph.PrimString, typegraph.PrimHostname:
		return "z.string()" + stringConstraints(field)
	case typegraph.PrimInt, typegraph.PrimInt32, typegraph.PrimInt64:
		return "z.int()" + numberConstraints(field)
	case typegraph.PrimFloat32, typegraph.PrimFloat64:
		return "z.number()" + numberConstraints(field)
	case typegraph.PrimBool:
		return "z.boolean()"
	default:
		return "z.unknown()"
	}
}

func (e *Emitter) enumValuesToZod(values []interface{}) string {
	if len(values) == 0 {
		return "z.never()"
	}

	category := enumutil.AnalyzeRawValues(values)

	if category.AllStrings {
		strValues := make([]string, len(values))
		for i, v := range values {
			strValues[i] = fmt.Sprintf("%q", v.(string))
		}
		return fmt.Sprintf("z.enum([%s])", strings.Join(strValues, ", "))
	}

	literals := make([]string, len(values))
	for i, v := range values {
		literals[i] = formatZodLiteral(v)
	}
	return fmt.Sprintf("z.union([%s])", strings.Join(literals, ", "))
}

func (e *Emitter) generateInlineObject(fields []*typegraph.Field) string {
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
