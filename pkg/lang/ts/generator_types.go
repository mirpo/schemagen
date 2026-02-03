package ts

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/common"
	"github.com/mirpo/schemagen/pkg/lang/tscommon"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

// anyType returns "any" or "unknown" based on configuration.
func (g *Generator) anyType() string {
	if g.config.UnknownAny {
		return "unknown"
	}
	return "any"
}

// typeRefToTS converts a TypeRef to a TypeScript type string.
func (g *Generator) typeRefToTS(ref *typegraph.TypeRef) string {
	if ref == nil {
		return g.anyType()
	}

	var tsType string

	switch ref.Kind {
	case typegraph.KindRef:
		// Reference to another type
		if ref.TypeName != "" {
			tsType = ref.TypeName
		} else {
			tsType = g.anyType()
		}
	case typegraph.KindEnum:
		// Inline enum - generate literal union type
		if len(ref.EnumValues) > 0 {
			literals := make([]string, 0, len(ref.EnumValues))
			for _, val := range ref.EnumValues {
				switch val.(type) {
				case string, float64, int, int64, bool, nil:
					literals = append(literals, common.TSLiterals.FormatValue(val))
				default:
					// Skip other types (objects, arrays)
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
		// Union type (oneOf/anyOf) - TypeScript has native union support
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
		tsType = g.primitiveToTS(ref.GoType)
	case typegraph.KindArray:
		itemType := g.typeRefToTS(ref.ItemType)
		tsType = itemType + "[]"
	case typegraph.KindMap:
		valueType := g.typeRefToTS(ref.ValueType)
		tsType = fmt.Sprintf("Record<string, %s>", valueType)
	case typegraph.KindInterface:
		// Check if this is an inline object with fields
		if len(ref.ObjectFields) > 0 {
			// Generate anonymous interface
			tsType = g.generateAnonymousInterface(ref.ObjectFields)
		} else {
			// Generic object without structure
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

// primitiveToTS maps Go primitive types to TypeScript types.
func (g *Generator) primitiveToTS(goType string) string {
	// Try centralized mapping first
	if tsType := typegraph.MapGoType(goType, "typescript"); tsType != "" {
		// Special case: interface{} should use config-aware anyType()
		if goType == "interface{}" {
			return g.anyType()
		}
		return tsType
	}

	// Default to any/unknown for unmapped types
	return g.anyType()
}

// generateAnonymousInterface generates an inline anonymous interface from fields.
func (g *Generator) generateAnonymousInterface(fields []*typegraph.Field) string {
	var sb strings.Builder
	sb.WriteString("{\n")

	for _, field := range fields {
		// Generate field comment with format annotation if present
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

		// Quote property name if it's not a valid identifier
		propName := field.JSONName
		if tscommon.NeedsQuoting(propName) {
			propName = fmt.Sprintf("%q", propName)
		}

		sb.WriteString(fmt.Sprintf("    %s%s: %s;\n", propName, optional, tsType))
	}

	sb.WriteString("  }")
	return sb.String()
}
