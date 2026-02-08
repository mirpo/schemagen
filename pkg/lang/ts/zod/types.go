package zod

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/typegraph"
)

// typeRefToZod converts a TypeRef to Zod schema code.
func (e *Emitter) typeRefToZod(ref *typegraph.TypeRef, field *typegraph.Field) string {
	if ref == nil {
		return "z.unknown()"
	}

	var zodType string

	switch ref.Kind {
	case typegraph.KindRef:
		zodType = ref.TypeName + "Schema"

	case typegraph.KindPrimitive:
		zodType = e.primitiveToZod(ref.GoType, ref.Format, field)

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

// primitiveToZod converts primitive types using Zod v4 top-level schemas.
func (e *Emitter) primitiveToZod(goType, format string, field *typegraph.Field) string {
	// Handle format-specific types first (Zod v4 top-level)
	switch format {
	case "email":
		return "z.email()" + stringConstraints(field)
	case "uri", "url":
		return "z.url()" + stringConstraints(field)
	case "uuid":
		return "z.uuid()"
	case "ipv4":
		return "z.ipv4()"
	case "ipv6":
		return "z.ipv6()"
	case "date-time":
		if e.config.CoerceDates {
			return "z.coerce.date()"
		}
		return "z.iso.datetime()"
	case "date":
		return "z.iso.date()"
	case "time":
		return "z.iso.time()"
	}

	// Handle base types
	switch goType {
	case "string":
		return "z.string()" + stringConstraints(field)
	case "int", "int32", "int64":
		return "z.int()" + numberConstraints(field)
	case "float64", "float32":
		return "z.number()" + numberConstraints(field)
	case "bool":
		return "z.boolean()"
	case "interface{}":
		return "z.unknown()"
	default:
		return "z.unknown()"
	}
}

// enumValuesToZod converts enum values to a Zod schema.
func (e *Emitter) enumValuesToZod(values []interface{}) string {
	if len(values) == 0 {
		return "z.never()"
	}

	// Check if all values are strings
	allStrings := true
	for _, v := range values {
		if _, ok := v.(string); !ok {
			allStrings = false
			break
		}
	}

	if allStrings {
		// Use z.enum for string enums
		strValues := make([]string, len(values))
		for i, v := range values {
			strValues[i] = fmt.Sprintf("%q", v.(string))
		}
		return fmt.Sprintf("z.enum([%s])", strings.Join(strValues, ", "))
	}

	// Use union of literals for mixed types (including objects/arrays)
	literals := make([]string, len(values))
	for i, v := range values {
		literals[i] = formatZodLiteral(v)
	}
	return fmt.Sprintf("z.union([%s])", strings.Join(literals, ", "))
}

// generateInlineObject generates a Zod object schema for inline object fields.
// Note: Inline objects don't have schema-level additionalProperties info,
// so we only use the --ts-zod-strict flag as a fallback default.
func (e *Emitter) generateInlineObject(fields []*typegraph.Field) string {
	var sb strings.Builder

	// Use strictObject when --ts-zod-strict flag is set
	objectFunc := "z.object"
	if e.config.Strict {
		objectFunc = "z.strictObject"
	}

	sb.WriteString(objectFunc + "({\n")

	for _, field := range fields {
		sb.WriteString(fmt.Sprintf("    %s,\n", e.generateField(field)))
	}

	sb.WriteString("  })")

	return sb.String()
}
