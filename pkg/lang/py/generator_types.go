package py

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/common"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

// checkTypeRefForImports recursively checks a TypeRef for import requirements.
func (g *Generator) checkTypeRefForImports(ref *typegraph.TypeRef) {
	if ref == nil {
		g.needsAny = true
		return
	}

	// Check format for special Pydantic types
	if ref.Format != "" {
		switch ref.Format {
		case "email":
			g.imports["pydantic_email"] = true
		case "uri", "url":
			g.imports["pydantic_url"] = true
		case "uuid":
			g.imports["uuid"] = true
		case "date-time":
			g.imports["datetime"] = true
		}
	}

	// Check Go type for imports
	switch ref.GoType {
	case "uuid.UUID":
		g.imports["uuid"] = true
	case "time.Time":
		g.imports["datetime"] = true
	}

	// Check for inline enum (needs Literal)
	if ref.Kind == typegraph.KindEnum && len(ref.EnumValues) > 0 {
		g.imports["typing_literal"] = true
	}

	// Check for types that need Any
	switch ref.Kind {
	case typegraph.KindInterface:
		g.needsAny = true
	case typegraph.KindPrimitive:
		// Check if it's interface{} which becomes Any
		if ref.GoType == "interface{}" {
			g.needsAny = true
		}
	case typegraph.KindMap:
		// Check if map value type needs Any
		g.checkTypeRefForImports(ref.ValueType)
	case typegraph.KindUnion:
		// Check all union members
		for _, member := range ref.UnionMembers {
			g.checkTypeRefForImports(member)
		}
	case typegraph.KindArray:
		g.checkTypeRefForImports(ref.ItemType)
	}
}

// typeRefToPython converts a TypeRef to a Python type annotation.
func (g *Generator) typeRefToPython(ref *typegraph.TypeRef, optional bool) string {
	if ref == nil {
		return "Any"
	}

	var pyType string

	switch ref.Kind {
	case typegraph.KindRef:
		// Reference to another type
		if ref.TypeName != "" {
			pyType = ref.TypeName
		} else {
			pyType = "Any"
		}
	case typegraph.KindEnum:
		// Inline enum - generate Literal type
		if len(ref.EnumValues) > 0 {
			g.imports["typing_literal"] = true
			literals := make([]string, 0, len(ref.EnumValues))
			for _, val := range ref.EnumValues {
				switch val.(type) {
				case string, float64, int, int64, bool, nil:
					literals = append(literals, common.PyLiterals.FormatValue(val))
				default:
					// Skip other types (objects, arrays)
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
		// Union type (oneOf/anyOf) - Python 3.10+ has native union support
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
		pyType = g.primitiveToPython(ref.GoType, ref.Format)
	case typegraph.KindArray:
		itemType := g.typeRefToPython(ref.ItemType, false)
		pyType = fmt.Sprintf("list[%s]", itemType)
	case typegraph.KindMap:
		valueType := g.typeRefToPython(ref.ValueType, false)
		pyType = fmt.Sprintf("dict[str, %s]", valueType)
	case typegraph.KindInterface:
		// Check if this is an inline object with fields
		if len(ref.ObjectFields) > 0 {
			// For Python, we use dict[str, Any] but it's better than just dict
			// TODO: Extract as separate BaseModel class for full type safety
			pyType = "dict[str, Any]"
			g.needsAny = true
		} else {
			// Generic dict without structure
			pyType = "dict[str, Any]"
			g.needsAny = true
		}
	default:
		pyType = "Any"
	}

	// Add Optional wrapper if field is optional (unless already nullable)
	if optional && !strings.Contains(pyType, " | None") {
		pyType += " | None"
	}

	return pyType
}

// primitiveToPython maps Go primitive types to Python types.
func (g *Generator) primitiveToPython(goType string, format string) string {
	// Check for format-specific types first
	if format != "" {
		switch format {
		case "email":
			g.imports["pydantic_email"] = true
			return "EmailStr"
		case "uri", "url":
			g.imports["pydantic_url"] = true
			return "AnyUrl"
		case "uuid":
			g.imports["uuid"] = true
			return "UUID"
		case "date-time":
			g.imports["datetime"] = true
			return "datetime"
		}
	}

	// Fall back to centralized Go type mapping
	if pyType := typegraph.MapGoType(goType, "python"); pyType != "" {
		// Handle special imports for mapped types
		switch goType {
		case "uuid.UUID":
			g.imports["uuid"] = true
		case "time.Time":
			g.imports["datetime"] = true
		}
		return pyType
	}

	// Default to Any for unmapped types
	return "Any"
}
