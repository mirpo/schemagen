package golang

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mirpo/schemagen/pkg/common"
	"github.com/mirpo/schemagen/pkg/naming"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

// Generator generates Go code from a type graph.
type Generator struct {
	graph   *typegraph.Graph
	config  *Config
	imports map[string]bool // import path → used
}

// Config contains generation configuration.
type Config struct {
	PackageName      string
	UsePointers      bool   // Use pointers for optional fields
	OmitEmpty        bool   // Add omitempty to optional fields
	DisableComments  bool   // Don't generate comments
	PackagePrefix    string // Package import prefix (e.g., "github.com/org/project")
	DisableHeaders   bool   // Don't generate file headers
	DisableTimestamp bool   // Don't generate timestamp in headers
}

// NewGenerator creates a new Go generator.
func NewGenerator(graph *typegraph.Graph, cfg *Config) *Generator {
	if cfg == nil {
		cfg = &Config{
			PackageName: "models",
			UsePointers: true,
			OmitEmpty:   true,
		}
	}
	return &Generator{
		graph:   graph,
		config:  cfg,
		imports: make(map[string]bool),
	}
}

// GenerateFile produces Go source code for a specific set of types with imports.
func (g *Generator) GenerateFile(types []*typegraph.Type, fileImports []typegraph.ImportSpec) (string, error) {
	var sb strings.Builder

	// Header comment
	sb.WriteString(common.GenerateHeader(common.HeaderConfig{
		CommentPrefix:    common.CommentPrefixGo,
		DisableHeaders:   g.config.DisableHeaders,
		DisableTimestamp: g.config.DisableTimestamp,
	}))

	// Package declaration
	sb.WriteString(fmt.Sprintf("package %s\n\n", g.config.PackageName))

	// Reset imports and scan types
	g.resetImports()

	for _, typ := range types {
		g.scanTypeForImports(typ)
	}

	// Add imports from other packages (file imports)
	for _, fileImport := range fileImports {
		if fileImport.ImportPath != "" {
			g.imports[fileImport.ImportPath] = true
		}
	}

	// Write imports if any
	if len(g.imports) > 0 {
		importList := make([]string, 0, len(g.imports))
		for imp := range g.imports {
			importList = append(importList, imp)
		}
		sort.Strings(importList)

		sb.WriteString("import (\n")
		for _, imp := range importList {
			sb.WriteString(fmt.Sprintf("\t%q\n", imp))
		}
		sb.WriteString(")\n\n")
	}

	// Sort types by name for deterministic output
	sortedTypes := make([]*typegraph.Type, len(types))
	copy(sortedTypes, types)
	sort.Slice(sortedTypes, func(i, j int) bool {
		return sortedTypes[i].Name < sortedTypes[j].Name
	})

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

	return sb.String(), nil
}

// generateType generates code for a single type.
func (g *Generator) generateType(typ *typegraph.Type) (string, error) {
	switch typ.Kind {
	case typegraph.KindStruct:
		return g.generateStruct(typ)
	case typegraph.KindEnum:
		return g.generateEnum(typ)
	case typegraph.KindUnion:
		return g.generateUnion(typ)
	case typegraph.KindPrimitive:
		return g.generateTypeAlias(typ)
	default:
		return "", fmt.Errorf("unsupported type kind: %s", typ.Kind)
	}
}

// generateStruct generates a struct type.
func (g *Generator) generateStruct(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// Type comment
	if !g.config.DisableComments && typ.Description != "" {
		sb.WriteString(fmt.Sprintf("// %s %s\n", typ.Name, typ.Description))
	}

	// Type declaration
	sb.WriteString(fmt.Sprintf("type %s struct {\n", typ.Name))

	// Embedded base types (for allOf composition)
	for _, baseType := range typ.Extends {
		sb.WriteString(fmt.Sprintf("\t%s\n", baseType))
	}

	// Fields
	for _, field := range typ.Fields {
		if !g.config.DisableComments {
			if field.Description != "" {
				sb.WriteString(fmt.Sprintf("\t// %s\n", field.Description))
			}
			// Add union type information if this is a union
			if field.Type.Kind == typegraph.KindUnion && len(field.Type.UnionMembers) > 0 {
				memberTypes := make([]string, 0, len(field.Type.UnionMembers))
				for _, member := range field.Type.UnionMembers {
					memberTypes = append(memberTypes, g.typeRefToGoType(member))
				}
				sb.WriteString(fmt.Sprintf("\t// Can be one of: %s\n", strings.Join(memberTypes, ", ")))
			}
		}

		goType := g.fieldGoType(field)
		jsonTag := g.fieldJSONTag(field)
		validateTag := g.fieldValidateTag(field)

		// Sanitize field name for Go (handles invalid identifiers like $dollar, 123numeric)
		fieldName := naming.ToGoFieldName(field.JSONName)

		// Build struct tags (json is always present, validate is optional)
		if validateTag != "" {
			sb.WriteString(fmt.Sprintf("\t%s %s `json:%q validate:%q`\n", fieldName, goType, jsonTag, validateTag))
		} else {
			sb.WriteString(fmt.Sprintf("\t%s %s `json:%q`\n", fieldName, goType, jsonTag))
		}
	}

	sb.WriteString("}")

	return sb.String(), nil
}

// generateEnum generates an enum type (string constants or var for complex enums).
func (g *Generator) generateEnum(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// Type comment
	if !g.config.DisableComments && typ.Description != "" {
		sb.WriteString(fmt.Sprintf("// %s %s\n", typ.Name, typ.Description))
	}

	// Analyze enum value types to determine generation strategy
	allStrings, allNumbers := analyzeEnumValueTypes(typ.EnumValues)

	// Determine the appropriate base type
	var enumType string
	if allStrings {
		enumType = "string"
	} else if allNumbers {
		enumType = "int"
	} else {
		enumType = "any"
	}

	// Type declaration
	sb.WriteString(fmt.Sprintf("type %s %s\n\n", typ.Name, enumType))

	// Generate values based on analysis
	if allStrings {
		// Use const for string-only enums
		sb.WriteString("const (\n")
		for _, val := range typ.EnumValues {
			sb.WriteString(fmt.Sprintf("\t%s%s %s = %q\n", typ.Name, val.Name, typ.Name, val.Value))
		}
		sb.WriteString(")")
	} else if allNumbers {
		// Use const for number-only enums with numeric literals
		sb.WriteString("const (\n")
		for _, val := range typ.EnumValues {
			numVal := formatNumericValue(val.Value)
			sb.WriteString(fmt.Sprintf("\t%s%s %s = %s\n", typ.Name, val.Name, typ.Name, numVal))
		}
		sb.WriteString(")")
	} else {
		// Use var with slice for mixed/complex enums
		sb.WriteString(fmt.Sprintf("var %sValues = []%s{\n", typ.Name, typ.Name))
		for _, val := range typ.EnumValues {
			sb.WriteString(fmt.Sprintf("\t%s,\n", formatEnumValue(val.Value)))
		}
		sb.WriteString("}")
	}

	return sb.String(), nil
}

// analyzeEnumValueTypes checks if all enum values are strings or all are numbers
func analyzeEnumValueTypes(values []typegraph.EnumValue) (allStrings, allNumbers bool) {
	if len(values) == 0 {
		return true, false // Empty enums default to string
	}

	allStrings = true
	allNumbers = true

	for _, v := range values {
		switch v.Value.(type) {
		case string:
			allNumbers = false
		case float64, int, int64, int32:
			allStrings = false
		default:
			// bool, nil, objects, arrays - neither string nor number
			allStrings = false
			allNumbers = false
		}
	}

	return allStrings, allNumbers
}

// formatNumericValue formats a numeric enum value as a Go literal
func formatNumericValue(val any) string {
	switch v := val.(type) {
	case float64:
		// Check if it's a whole number
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case int32:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// generateUnion generates a type alias for union types.
func (g *Generator) generateUnion(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// Type comment
	if !g.config.DisableComments && typ.Description != "" {
		sb.WriteString(fmt.Sprintf("// %s %s\n", typ.Name, typ.Description))
	}

	// Type alias (use = for type alias, not type definition)
	sb.WriteString(fmt.Sprintf("type %s = any", typ.Name))

	return sb.String(), nil
}

// formatEnumValue formats an enum value for Go code generation
func formatEnumValue(val any) string {
	switch v := val.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	case float64:
		return fmt.Sprintf("%v", v)
	case int, int64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case nil:
		return "nil"
	case map[string]any:
		return formatMap(v)
	case []any:
		return formatSlice(v)
	default:
		return fmt.Sprintf("%#v", v)
	}
}

// formatMap formats a map for Go code generation
func formatMap(m map[string]any) string {
	var sb strings.Builder
	sb.WriteString("map[string]any{")
	first := true
	for k, v := range m {
		if !first {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%q: %s", k, formatEnumValue(v)))
		first = false
	}
	sb.WriteString("}")
	return sb.String()
}

// formatSlice formats a slice for Go code generation
func formatSlice(s []any) string {
	var sb strings.Builder
	sb.WriteString("[]any{")
	for i, v := range s {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(formatEnumValue(v))
	}
	sb.WriteString("}")
	return sb.String()
}

// generateTypeAlias generates a type alias for primitives.
func (g *Generator) generateTypeAlias(typ *typegraph.Type) (string, error) {
	// For now, we don't generate aliases for primitives
	// This would be used for custom type mappings
	return "", nil
}
