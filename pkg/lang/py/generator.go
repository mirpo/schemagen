package py

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mirpo/schemagen/pkg/common"
	"github.com/mirpo/schemagen/pkg/naming"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

// Config contains Python generation configuration.
type Config struct {
	DisableHeaders   bool
	DisableTimestamp bool
	SnakeCaseField   bool // Convert field names to snake_case with JSON alias
	AllowExtraFields bool // Generate model_config = ConfigDict(extra='allow')
}

// Generator generates Python (Pydantic) code from a type graph.
type Generator struct {
	graph    *typegraph.Graph
	config   *Config
	imports  map[string]bool
	needsAny bool // Track if typing.Any is needed
}

// NewGenerator creates a new Python generator.
func NewGenerator(graph *typegraph.Graph) *Generator {
	return NewGeneratorWithConfig(graph, &Config{})
}

// NewGeneratorWithConfig creates a Python generator with custom config.
func NewGeneratorWithConfig(graph *typegraph.Graph, cfg *Config) *Generator {
	if cfg == nil {
		cfg = &Config{}
	}
	return &Generator{
		graph:    graph,
		config:   cfg,
		imports:  make(map[string]bool),
		needsAny: false,
	}
}

// GenerateFile produces Python source code for a specific set of types with imports.
func (g *Generator) GenerateFile(types []*typegraph.Type, fileImports []typegraph.ImportSpec) (string, error) {
	var sb strings.Builder

	// Header
	sb.WriteString(common.GenerateHeader(common.HeaderConfig{
		CommentPrefix:    common.CommentPrefixHash,
		DisableHeaders:   g.config.DisableHeaders,
		DisableTimestamp: g.config.DisableTimestamp,
	}))
	sb.WriteString("from __future__ import annotations\n\n")

	// Reset and scan for imports from these specific types
	g.imports = make(map[string]bool)
	g.needsAny = false

	// Check if we need BaseModel (any struct or class type)
	hasStructTypes := false
	for _, typ := range types {
		switch typ.Kind {
		case typegraph.KindStruct:
			hasStructTypes = true
			for _, field := range typ.Fields {
				g.checkTypeRefForImports(field.Type)
				// Check if Field() will be needed for this field
				if field.Description != "" || field.MinLength != nil || field.MaxLength != nil ||
					field.Pattern != nil || field.Minimum != nil || field.Maximum != nil ||
					field.ExclusiveMinimum != nil || field.ExclusiveMaximum != nil ||
					field.MinItems != nil || field.MaxItems != nil {
					g.imports["pydantic_field"] = true
				}
			}
		case typegraph.KindUnion:
			// Check if union will fall back to Any
			if typ.TargetType == nil || typ.TargetType.Kind != typegraph.KindUnion {
				g.needsAny = true
			} else {
				g.checkTypeRefForImports(typ.TargetType)
			}
		case typegraph.KindAlias:
			// Check if alias will fall back to Any
			if typ.TargetType == nil {
				g.needsAny = true
			} else {
				g.checkTypeRefForImports(typ.TargetType)
			}
		case typegraph.KindPrimitive:
			// Check if primitive type will use Any
			if typ.GoType == "interface{}" || typ.GoType == "" {
				g.needsAny = true
			}
		case typegraph.KindEnum:
			// Categorize enum type for import determination
			hasString := false
			hasNumber := false
			hasOther := false

			for _, val := range typ.EnumValues {
				switch val.Value.(type) {
				case string:
					hasString = true
				case float64, int, int64:
					hasNumber = true
				default:
					hasOther = true
				}
			}

			// Determine which imports are needed
			if (hasString && hasNumber) || (hasString && hasOther) || (hasNumber && hasOther) || hasOther {
				g.imports["typing_literal"] = true
			} else if hasNumber && !hasString {
				g.imports["enum_int"] = true
			} else {
				g.imports["enum"] = true
			}
		}
	}

	// Always import BaseModel if we have struct types
	if hasStructTypes {
		g.imports["pydantic"] = true

		// Check if we need ConfigDict for additionalProperties
		if g.config.AllowExtraFields {
			for _, typ := range types {
				if typ.Kind == typegraph.KindStruct && typ.AdditionalProps != nil && typ.AdditionalProps.Allowed {
					g.imports["pydantic_config"] = true
					break
				}
			}
		}
	}

	// Write standard library and typing imports
	sb.WriteString(g.generateImports())

	// Write relative imports from other files
	if len(fileImports) > 0 {
		sb.WriteString("\n")
		for _, imp := range fileImports {
			if len(imp.TypeNames) == 0 {
				continue
			}

			sort.Strings(imp.TypeNames)

			sb.WriteString(fmt.Sprintf("from %s import %s\n",
				imp.ImportPath,
				strings.Join(imp.TypeNames, ", ")))
		}
	}

	// PEP 8: Two blank lines after imports before type definitions
	sb.WriteString("\n\n")

	// Sort types in topological order (dependencies first)
	sortedTypes := make([]*typegraph.Type, len(types))
	copy(sortedTypes, types)
	sortedTypes = topologicalSort(sortedTypes)

	// Generate types
	for i, typ := range sortedTypes {
		if i > 0 {
			sb.WriteString("\n\n")
		}

		code, err := g.generateType(typ)
		if err != nil {
			return "", err
		}
		sb.WriteString(code)
	}

	// No trailing newline - Black doesn't want it
	return sb.String(), nil
}

// checkTypeRefForImports checks a TypeRef and adds necessary imports.

// generateImports generates the import statements.
func (g *Generator) generateImports() string {
	var sb strings.Builder

	// Standard library imports
	if g.imports["datetime"] {
		sb.WriteString("from datetime import datetime\n")
	}

	// Enum imports - handle both Enum and IntEnum
	enumImports := []string{}
	if g.imports["enum"] {
		enumImports = append(enumImports, "Enum")
	}
	if g.imports["enum_int"] {
		enumImports = append(enumImports, "IntEnum")
	}
	if len(enumImports) > 0 {
		sb.WriteString(fmt.Sprintf("from enum import %s\n", strings.Join(enumImports, ", ")))
	}

	if g.imports["uuid"] {
		sb.WriteString("from uuid import UUID\n")
	}

	// Add newline if we had stdlib imports
	if g.imports["datetime"] || g.imports["uuid"] || g.imports["enum"] || g.imports["enum_int"] {
		sb.WriteString("\n")
	}

	// typing imports
	typingImports := []string{}
	if g.needsAny {
		typingImports = append(typingImports, "Any")
	}
	if g.imports["typing_literal"] {
		typingImports = append(typingImports, "Literal")
	}
	if len(typingImports) > 0 {
		sb.WriteString(fmt.Sprintf("from typing import %s\n", strings.Join(typingImports, ", ")))
	}

	// Pydantic imports
	if g.imports["pydantic"] || g.imports["pydantic_field"] || g.imports["pydantic_email"] || g.imports["pydantic_url"] || g.imports["pydantic_config"] {
		pydanticImports := []string{"BaseModel"}
		if g.imports["pydantic_field"] {
			pydanticImports = append(pydanticImports, "Field")
		}
		if g.imports["pydantic_email"] {
			pydanticImports = append(pydanticImports, "EmailStr")
		}
		if g.imports["pydantic_url"] {
			pydanticImports = append(pydanticImports, "AnyUrl")
		}
		if g.imports["pydantic_config"] {
			pydanticImports = append(pydanticImports, "ConfigDict")
		}
		sb.WriteString(fmt.Sprintf("from pydantic import %s\n", strings.Join(pydanticImports, ", ")))
	}

	return sb.String()
}

// generateType generates code for a single type.
func (g *Generator) generateType(typ *typegraph.Type) (string, error) {
	switch typ.Kind {
	case typegraph.KindStruct:
		return g.generateClass(typ)
	case typegraph.KindEnum:
		return g.generateEnum(typ)
	case typegraph.KindPrimitive:
		return g.generatePrimitiveAlias(typ)
	case typegraph.KindUnion:
		return g.generateUnionAlias(typ)
	case typegraph.KindAlias:
		return g.generateTypeAlias(typ)
	default:
		return "", fmt.Errorf("unsupported type kind: %s", typ.Kind)
	}
}

// Python keywords that need to be escaped
var pythonKeywords = map[string]bool{
	"and": true, "as": true, "assert": true, "async": true, "await": true,
	"break": true, "class": true, "continue": true, "def": true, "del": true,
	"elif": true, "else": true, "except": true, "finally": true, "for": true,
	"from": true, "global": true, "if": true, "import": true, "in": true,
	"is": true, "lambda": true, "nonlocal": true, "not": true, "or": true,
	"pass": true, "raise": true, "return": true, "try": true, "while": true,
	"with": true, "yield": true,
}

// sanitizePythonIdentifier converts a string to a valid Python identifier.
// Handles cases like: "123abc" -> "field_123abc", "$var" -> "field_var", "@special" -> "field_special", "as" -> "as_"
func sanitizePythonIdentifier(s string) string {
	if s == "" {
		return "field"
	}

	// Replace invalid characters with underscores
	var result strings.Builder
	for i, r := range s {
		if i == 0 {
			// First character must be letter or underscore
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' {
				result.WriteRune(r)
			} else if r >= '0' && r <= '9' {
				// Starts with number - prefix with "field_"
				result.WriteString("field_")
				result.WriteRune(r)
			} else {
				// Starts with special char - prefix with "field_"
				result.WriteString("field_")
				if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
					result.WriteRune(r)
				} else {
					result.WriteRune('_')
				}
			}
		} else {
			// Subsequent characters can be letter, digit, or underscore
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
				result.WriteRune(r)
			} else {
				result.WriteRune('_')
			}
		}
	}

	sanitized := result.String()

	// Check if it's a Python keyword - if so, append underscore
	if pythonKeywords[sanitized] {
		return sanitized + "_"
	}

	return sanitized
}

// escapeDocstring escapes triple quotes in Python docstrings to prevent syntax errors.
// Python docstrings are delimited by """ and any """ inside must be escaped.
func escapeDocstring(s string) string {
	// Escape triple double quotes to prevent premature docstring termination
	return strings.ReplaceAll(s, `"""`, `\"\"\"`)
}

// formatPythonString formats a string for use in Python code, choosing between
// single or double quotes based on content to minimize escaping.
func formatPythonString(s string) string {
	hasDoubleQuote := strings.Contains(s, `"`)
	hasSingleQuote := strings.Contains(s, `'`)

	// If string has double quotes but no single quotes, use single quotes
	if hasDoubleQuote && !hasSingleQuote {
		// Escape backslashes and single quotes
		s = strings.ReplaceAll(s, `\`, `\\`)
		return fmt.Sprintf("'%s'", s)
	}

	// Otherwise use double quotes (default, matches Python convention)
	// Use %q which handles escaping for us
	return fmt.Sprintf("%q", s)
}

// writeDescription writes a Python docstring with proper escaping and formatting.
// indent should be the indentation string (e.g., "    " for class-level, "" for module-level).
// suffix controls trailing newlines (e.g., "\n\n" for classes, "\n" for type aliases).
func writeDescription(description, indent, suffix string) string {
	if description == "" {
		return ""
	}
	return fmt.Sprintf("%s\"\"\"%s\"\"\"%s", indent, escapeDocstring(description), suffix)
}

// generateClass generates a Pydantic BaseModel class.
func (g *Generator) generateClass(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// Class declaration with inheritance
	baseClasses := "BaseModel"
	if len(typ.Extends) > 0 {
		// Multiple inheritance: class Child(Parent1, Parent2):
		// Don't include BaseModel if we're extending other classes (they already inherit from it)
		baseClasses = strings.Join(typ.Extends, ", ")
	}
	sb.WriteString(fmt.Sprintf("class %s(%s):\n", typ.Name, baseClasses))
	sb.WriteString(writeDescription(typ.Description, "    ", "\n\n"))

	// Add model_config if flag enabled and type has additionalProperties
	if g.config.AllowExtraFields && typ.AdditionalProps != nil && typ.AdditionalProps.Allowed {
		sb.WriteString("    model_config = ConfigDict(extra='allow')\n\n")
	}

	// Fields
	if len(typ.Fields) == 0 && len(typ.Extends) > 0 {
		// If we have no fields but extend other types, use pass
		sb.WriteString("    pass\n")
	} else if len(typ.Fields) == 0 {
		sb.WriteString("    pass\n")
	} else {
		for _, field := range typ.Fields {
			pyType := g.typeRefToPython(field.Type, !field.Required)

			// Determine field name (snake_case if flag is enabled)
			fieldName := field.JSONName
			needsAlias := false

			// Sanitize field name to be a valid Python identifier
			sanitized := sanitizePythonIdentifier(field.JSONName)
			if sanitized != field.JSONName {
				fieldName = sanitized
				needsAlias = true
			}

			if g.config.SnakeCaseField {
				snakeName := naming.ToSnakeCase(fieldName)
				if snakeName != fieldName {
					fieldName = snakeName
					needsAlias = true
				}
			}

			// Build Field() parameters (empty if not needed)
			fieldParams := g.buildFieldParams(field, field.Required, needsAlias, field.JSONName)

			if len(fieldParams) > 0 {
				// Use Field() with constraints
				sb.WriteString(fmt.Sprintf("    %s: %s = Field(%s)\n",
					fieldName, pyType, strings.Join(fieldParams, ", ")))
			} else {
				// Simple field without Field()
				if field.Required {
					sb.WriteString(fmt.Sprintf("    %s: %s\n", fieldName, pyType))
				} else {
					sb.WriteString(fmt.Sprintf("    %s: %s = None\n", fieldName, pyType))
				}
			}
		}
	}

	return sb.String(), nil
}

// sanitizeEnumMemberName ensures the enum member name is a valid Python identifier.
// Numeric names like "1", "2" are prefixed with "N_" to become "N_1", "N_2".
func sanitizeEnumMemberName(name string) string {
	if len(name) == 0 {
		return "EMPTY"
	}

	// Check if name starts with a digit
	if name[0] >= '0' && name[0] <= '9' {
		return "N_" + name
	}

	return name
}

// generateEnum generates a Python Enum class or Literal type alias.
func (g *Generator) generateEnum(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// Categorize enum type: all strings, all numbers, or mixed
	hasString := false
	hasNumber := false
	hasOther := false // bool, null, etc.

	for _, val := range typ.EnumValues {
		switch val.Value.(type) {
		case string:
			hasString = true
		case float64, int, int64:
			hasNumber = true
		default:
			hasOther = true
		}
	}

	// For mixed-type enums or enums with bool/null, use Literal type
	if (hasString && hasNumber) || (hasString && hasOther) || (hasNumber && hasOther) || hasOther {
		g.imports["typing_literal"] = true

		// Add description as comment if present
		if typ.Description != "" {
			sb.WriteString(fmt.Sprintf("# %s\n", typ.Description))
		}

		literals := make([]string, 0, len(typ.EnumValues))
		for _, val := range typ.EnumValues {
			switch v := val.Value.(type) {
			case string:
				literals = append(literals, fmt.Sprintf("%q", v))
			case float64, int, int64:
				literals = append(literals, fmt.Sprintf("%v", v))
			case bool:
				if v {
					literals = append(literals, "True")
				} else {
					literals = append(literals, "False")
				}
			case nil:
				literals = append(literals, "None")
			}
		}
		sb.WriteString(fmt.Sprintf("%s = Literal[%s]\n", typ.Name, strings.Join(literals, ", ")))
		return sb.String(), nil
	}

	// For number-only enums, use IntEnum
	if hasNumber && !hasString {
		g.imports["enum_int"] = true

		// Class declaration
		sb.WriteString(fmt.Sprintf("class %s(IntEnum):\n", typ.Name))
		sb.WriteString(writeDescription(typ.Description, "    ", "\n\n"))

		// Enum values
		for _, val := range typ.EnumValues {
			memberName := sanitizeEnumMemberName(val.Name)
			switch v := val.Value.(type) {
			case float64:
				sb.WriteString(fmt.Sprintf("    %s = %d\n", memberName, int(v)))
			case int, int64:
				sb.WriteString(fmt.Sprintf("    %s = %v\n", memberName, v))
			}
		}

		return sb.String(), nil
	}

	// For string-only enums, use traditional str, Enum class
	g.imports["enum"] = true

	// Class declaration
	sb.WriteString(fmt.Sprintf("class %s(str, Enum):\n", typ.Name))
	sb.WriteString(writeDescription(typ.Description, "    ", "\n\n"))

	// Enum values
	for _, val := range typ.EnumValues {
		if strVal, ok := val.Value.(string); ok {
			memberName := sanitizeEnumMemberName(val.Name)
			sb.WriteString(fmt.Sprintf("    %s = %q\n", memberName, strVal))
		}
	}

	return sb.String(), nil
}

// generatePrimitiveAlias generates a type alias for a primitive type.
func (g *Generator) generatePrimitiveAlias(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// Python type alias using TypeAlias (Python 3.10+) or simple assignment
	pyType := g.primitiveToPython(typ.GoType, "")
	sb.WriteString(fmt.Sprintf("%s = %s\n", typ.Name, pyType))
	sb.WriteString(writeDescription(typ.Description, "", "\n"))

	return sb.String(), nil
}

// generateUnionAlias generates a type alias for a union type.
func (g *Generator) generateUnionAlias(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// Generate union type from TargetType if it's a union
	if typ.TargetType != nil && typ.TargetType.Kind == typegraph.KindUnion {
		pyType := g.typeRefToPython(typ.TargetType, false)
		sb.WriteString(fmt.Sprintf("%s = %s\n", typ.Name, pyType))
	} else {
		// Fallback to Any if we don't have proper union information
		g.needsAny = true
		sb.WriteString(fmt.Sprintf("%s = Any\n", typ.Name))
	}
	sb.WriteString(writeDescription(typ.Description, "", "\n"))

	return sb.String(), nil
}

// generateTypeAlias generates a type alias for an alias type.
func (g *Generator) generateTypeAlias(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// Generate type alias
	if typ.TargetType != nil {
		pyType := g.typeRefToPython(typ.TargetType, false)
		sb.WriteString(fmt.Sprintf("%s = %s\n", typ.Name, pyType))
	} else {
		g.needsAny = true
		sb.WriteString(fmt.Sprintf("%s = Any\n", typ.Name))
	}
	sb.WriteString(writeDescription(typ.Description, "", "\n"))

	return sb.String(), nil
}

// typeRefToPython converts a TypeRef to a Python type annotation.

// primitiveToPython maps Go primitive types to Python types, considering format.
