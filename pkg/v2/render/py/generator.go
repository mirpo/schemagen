package py

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mirpo/schemagen/pkg/common"
	"github.com/mirpo/schemagen/pkg/naming"
	"github.com/mirpo/schemagen/pkg/v2/graph"
)

type Config struct {
	DisableHeaders   bool
	DisableTimestamp bool
	SnakeCaseField   bool // Convert field names to snake_case with JSON alias
	AllowExtraFields bool // Generate model_config = ConfigDict(extra='allow')
}

type Generator struct {
	config   *Config
	imports  map[string]bool
	needsAny bool // Track if typing.Any is needed
}

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

func (g *Generator) GenerateFile(types []*graph.Type, fileImports []graph.ImportSpec) (string, error) {
	var sb strings.Builder

	sb.WriteString(common.GenerateHeader(common.HeaderConfig{
		CommentPrefix:    common.CommentPrefixHash,
		DisableHeaders:   g.config.DisableHeaders,
		DisableTimestamp: g.config.DisableTimestamp,
	}))
	sb.WriteString("from __future__ import annotations\n\n")

	g.scanTypesForImports(types)
	sb.WriteString(g.generateImports())

	if len(fileImports) > 0 {
		sb.WriteString("\n")
		for _, imp := range fileImports {
			if len(imp.TypeNames) == 0 {
				continue
			}

			names := slices.Sorted(slices.Values(imp.TypeNames))

			fmt.Fprintf(&sb, "from %s import %s\n",
				imp.ImportPath,
				strings.Join(names, ", "))
		}
	}

	// PEP 8: Two blank lines after imports before type definitions
	sb.WriteString("\n\n")

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

func (g *Generator) scanTypesForImports(types []*graph.Type) {
	g.imports = make(map[string]bool)
	g.needsAny = false

	hasStructTypes := false
	for _, typ := range types {
		switch typ.Kind {
		case graph.KindStruct:
			hasStructTypes = true
			for _, field := range typ.Fields {
				g.checkTypeRefForImports(field.Type)
				if needsField(field, g.fieldNeedsAlias(field)) {
					g.imports["pydantic_field"] = true
				}
			}
		case graph.KindUnion:
			g.needsAny = true
		case graph.KindPrimitive:
			if typ.Primitive == graph.PrimUnknown {
				g.needsAny = true
			}
			if imp := importsForPrimitive(typ.Primitive); imp != "" {
				g.imports[imp] = true
			}
		case graph.KindEnum:
			category := graph.AnalyzeEnumValues(typ.EnumValues)
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
		if typ.Kind == graph.KindStruct && typ.AdditionalProps != nil && typ.AdditionalProps.Allowed {
			g.imports["pydantic_config"] = true
			return
		}
	}
}

func (g *Generator) generateImports() string {
	var sb strings.Builder

	if g.imports["datetime"] {
		sb.WriteString("from datetime import datetime\n")
	}

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

	if g.imports["datetime"] || g.imports["uuid"] || g.imports["enum"] || g.imports["enum_int"] {
		sb.WriteString("\n")
	}

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

func (g *Generator) generateType(typ *graph.Type) (string, error) {
	switch typ.Kind {
	case graph.KindStruct:
		return g.generateClass(typ)
	case graph.KindEnum:
		return g.generateEnum(typ)
	case graph.KindPrimitive:
		return g.generatePrimitiveAlias(typ)
	case graph.KindUnion:
		return g.generateUnionAlias(typ)
	default:
		return "", fmt.Errorf("unsupported type kind: %s", typ.Kind)
	}
}

func (g *Generator) fieldNeedsAlias(field *graph.Field) bool {
	name := field.JSONName
	if sanitizePythonIdentifier(name) != name {
		return true
	}
	if g.config.SnakeCaseField {
		return naming.ToSnakeCase(name) != name
	}
	return false
}

var pythonKeywords = map[string]bool{
	"and": true, "as": true, "assert": true, "async": true, "await": true,
	"break": true, "class": true, "continue": true, "def": true, "del": true,
	"elif": true, "else": true, "except": true, "finally": true, "for": true,
	"from": true, "global": true, "if": true, "import": true, "in": true,
	"is": true, "lambda": true, "nonlocal": true, "not": true, "or": true,
	"pass": true, "raise": true, "return": true, "try": true, "while": true,
	"with": true, "yield": true,
}

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
	return fmt.Sprintf("%q", s)
}

func writeDescription(description, indent, suffix string) string {
	if description == "" {
		return ""
	}
	return fmt.Sprintf("%s\"\"\"%s\"\"\"%s", indent, escapeDocstring(description), suffix)
}

func (g *Generator) generateClass(typ *graph.Type) (string, error) {
	var sb strings.Builder

	// Class declaration with inheritance
	baseClasses := "BaseModel"
	if len(typ.Extends) > 0 {
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

func (g *Generator) renderField(sb *strings.Builder, field *graph.Field) {
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

func (g *Generator) generateEnum(typ *graph.Type) (string, error) {
	var sb strings.Builder

	// Categorize enum type using shared analyzer
	category := graph.AnalyzeEnumValues(typ.EnumValues)

	// For mixed-type enums or enums with bool/null, use Literal type
	if category.HasMixed {
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

func (g *Generator) generatePrimitiveAlias(typ *graph.Type) (string, error) {
	var sb strings.Builder

	pyType := g.primitiveToPython(typ.Primitive)
	fmt.Fprintf(&sb, "%s = %s\n", typ.Name, pyType)
	sb.WriteString(writeDescription(typ.Description, "", "\n"))

	return sb.String(), nil
}

func (g *Generator) generateUnionAlias(typ *graph.Type) (string, error) {
	var sb strings.Builder

	fmt.Fprintf(&sb, "%s = Any\n", typ.Name)
	sb.WriteString(writeDescription(typ.Description, "", "\n"))

	return sb.String(), nil
}
