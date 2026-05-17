package py

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mirpo/schemagen/pkg/common"
	"github.com/mirpo/schemagen/pkg/enumutil"
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
	config   *Config
	imports  map[string]bool
	needsAny bool // Track if typing.Any is needed
}

// NewGeneratorWithConfig creates a Python generator with custom config.
func NewGeneratorWithConfig(cfg *Config) *Generator {
	if cfg == nil {
		cfg = &Config{}
	}
	return &Generator{
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

	g.scanTypesForImports(types)
	sb.WriteString(g.generateImports())

	// Write relative imports from other files
	if len(fileImports) > 0 {
		sb.WriteString("\n")
		for _, imp := range fileImports {
			if len(imp.TypeNames) == 0 {
				continue
			}

			sort.Strings(imp.TypeNames)

			fmt.Fprintf(&sb, "from %s import %s\n",
				imp.ImportPath,
				strings.Join(imp.TypeNames, ", "))
		}
	}

	// PEP 8: Two blank lines after imports before type definitions
	sb.WriteString("\n\n")

	// Generate types
	for i, typ := range types {
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

func (g *Generator) scanTypesForImports(types []*typegraph.Type) {
	g.imports = make(map[string]bool)
	g.needsAny = false

	hasStructTypes := false
	for _, typ := range types {
		switch typ.Kind {
		case typegraph.KindStruct:
			hasStructTypes = true
			for _, field := range typ.Fields {
				g.checkTypeRefForImports(field.Type)
				if needsField(field, g.fieldNeedsAlias(field)) {
					g.imports["pydantic_field"] = true
				}
			}
		case typegraph.KindUnion:
			g.needsAny = true
		case typegraph.KindPrimitive:
			if typ.Primitive == typegraph.PrimUnknown {
				g.needsAny = true
			}
		case typegraph.KindEnum:
			category := enumutil.AnalyzeEnumValues(typ.EnumValues)
			if category.HasMixed {
				g.imports["typing_literal"] = true
			} else if category.AllNumbers {
				g.imports["enum_int"] = true
			} else {
				g.imports["enum"] = true
			}
		}
	}

	if !hasStructTypes {
		return
	}

	g.imports["pydantic"] = true

	if !g.config.AllowExtraFields {
		return
	}

	for _, typ := range types {
		if typ.Kind == typegraph.KindStruct && typ.AdditionalProps != nil && typ.AdditionalProps.Allowed {
			g.imports["pydantic_config"] = true
			return
		}
	}
}

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
		fmt.Fprintf(&sb, "from enum import %s\n", strings.Join(enumImports, ", "))
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
		fmt.Fprintf(&sb, "from typing import %s\n", strings.Join(typingImports, ", "))
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
		fmt.Fprintf(&sb, "from pydantic import %s\n", strings.Join(pydanticImports, ", "))
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
	default:
		return "", fmt.Errorf("unsupported type kind: %s", typ.Kind)
	}
}

func (g *Generator) fieldNeedsAlias(field *typegraph.Field) bool {
	name := field.JSONName
	if sanitizePythonIdentifier(name) != name {
		return true
	}
	if g.config.SnakeCaseField {
		return naming.ToSnakeCase(name) != name
	}
	return false
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
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' {
				result.WriteRune(r)
			} else if r >= '0' && r <= '9' {
				result.WriteString("field_")
				result.WriteRune(r)
			} else {
				result.WriteString("field__")
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

func escapeDocstring(s string) string {
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
	fmt.Fprintf(&sb, "class %s(%s):\n", typ.Name, baseClasses)
	sb.WriteString(writeDescription(typ.Description, "    ", "\n\n"))

	// Add model_config if flag enabled and type has additionalProperties
	if g.config.AllowExtraFields && typ.AdditionalProps != nil && typ.AdditionalProps.Allowed {
		sb.WriteString("    model_config = ConfigDict(extra='allow')\n\n")
	}

	if len(typ.Fields) == 0 {
		sb.WriteString("    pass\n")
	} else {
		for _, field := range typ.Fields {
			g.renderField(&sb, field)
		}
	}

	return sb.String(), nil
}

func (g *Generator) renderField(sb *strings.Builder, field *typegraph.Field) {
	pyType := g.typeRefToPython(field.Type, !field.Required)

	fieldName := field.JSONName
	needsAlias := false

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

	fieldParams := g.buildFieldParams(field, field.Required, needsAlias, field.JSONName)

	if len(fieldParams) > 0 {
		fmt.Fprintf(sb, "    %s: %s = Field(%s)\n",
			fieldName, pyType, strings.Join(fieldParams, ", "))
	} else if field.Required {
		fmt.Fprintf(sb, "    %s: %s\n", fieldName, pyType)
	} else {
		fmt.Fprintf(sb, "    %s: %s = None\n", fieldName, pyType)
	}
}

// generateEnum generates a Python Enum class or Literal type alias.
func (g *Generator) generateEnum(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// Categorize enum type using shared analyzer
	category := enumutil.AnalyzeEnumValues(typ.EnumValues)

	// For mixed-type enums or enums with bool/null, use Literal type
	if category.HasMixed {
		g.imports["typing_literal"] = true

		// Add description as comment if present
		if typ.Description != "" {
			fmt.Fprintf(&sb, "# %s\n", typ.Description)
		}

		literals := make([]string, 0, len(typ.EnumValues))
		for _, val := range typ.EnumValues {
			switch val.Value.(type) {
			case string, float64, int, int64, int32, bool, nil:
				literals = append(literals, common.PyLiterals.FormatValue(val.Value))
			}
		}
		fmt.Fprintf(&sb, "%s = Literal[%s]\n", typ.Name, strings.Join(literals, ", "))
		return sb.String(), nil
	}

	// For number-only enums, use IntEnum
	if category.AllNumbers {
		g.imports["enum_int"] = true

		// Class declaration
		fmt.Fprintf(&sb, "class %s(IntEnum):\n", typ.Name)
		sb.WriteString(writeDescription(typ.Description, "    ", "\n\n"))

		// Enum values
		for _, val := range typ.EnumValues {
			memberName := naming.SanitizeEnumMember(val.Name)
			switch v := val.Value.(type) {
			case float64:
				fmt.Fprintf(&sb, "    %s = %d\n", memberName, int(v))
			case int, int64:
				fmt.Fprintf(&sb, "    %s = %v\n", memberName, v)
			}
		}

		return sb.String(), nil
	}

	// For string-only enums, use traditional str, Enum class
	g.imports["enum"] = true

	// Class declaration
	fmt.Fprintf(&sb, "class %s(str, Enum):\n", typ.Name)
	sb.WriteString(writeDescription(typ.Description, "    ", "\n\n"))

	// Enum values
	for _, val := range typ.EnumValues {
		if strVal, ok := val.Value.(string); ok {
			memberName := naming.SanitizeEnumMember(val.Name)
			fmt.Fprintf(&sb, "    %s = %q\n", memberName, strVal)
		}
	}

	return sb.String(), nil
}

// generatePrimitiveAlias generates a type alias for a primitive type.
func (g *Generator) generatePrimitiveAlias(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// Python type alias using TypeAlias (Python 3.10+) or simple assignment
	pyType := g.primitiveToPython(typ.Primitive)
	fmt.Fprintf(&sb, "%s = %s\n", typ.Name, pyType)
	sb.WriteString(writeDescription(typ.Description, "", "\n"))

	return sb.String(), nil
}

func (g *Generator) generateUnionAlias(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	g.needsAny = true
	fmt.Fprintf(&sb, "%s = Any\n", typ.Name)
	sb.WriteString(writeDescription(typ.Description, "", "\n"))

	return sb.String(), nil
}
