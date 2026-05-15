package golang

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mirpo/schemagen/pkg/common"
	"github.com/mirpo/schemagen/pkg/enumutil"
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
		CommentPrefix:    common.CommentPrefixSlash,
		DisableHeaders:   g.config.DisableHeaders,
		DisableTimestamp: g.config.DisableTimestamp,
	}))

	// Package declaration
	fmt.Fprintf(&sb, "package %s\n\n", g.config.PackageName)

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
			fmt.Fprintf(&sb, "\t%q\n", imp)
		}
		sb.WriteString(")\n\n")
	}

	// Generate types (preserving input order for correct schema references)
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
		fmt.Fprintf(&sb, "// %s %s\n", typ.Name, typ.Description)
	}

	// Type declaration
	fmt.Fprintf(&sb, "type %s struct {\n", typ.Name)

	// Embedded base types (for allOf composition)
	for _, baseType := range typ.Extends {
		fmt.Fprintf(&sb, "\t%s\n", baseType)
	}

	// Fields
	for _, field := range typ.Fields {
		if !g.config.DisableComments {
			if field.Description != "" {
				fmt.Fprintf(&sb, "\t// %s\n", field.Description)
			}
			if field.Type != nil && field.Type.Kind == typegraph.KindUnion && len(field.Type.UnionMembers) > 0 {
				memberTypes := make([]string, 0, len(field.Type.UnionMembers))
				for _, member := range field.Type.UnionMembers {
					memberTypes = append(memberTypes, g.typeRefToGoType(member))
				}
				fmt.Fprintf(&sb, "\t// Can be one of: %s\n", strings.Join(memberTypes, ", "))
			}
		}

		goType := g.fieldGoType(field)
		jsonTag := g.fieldJSONTag(field)
		validateTag := g.fieldValidateTag(field)

		// Sanitize field name for Go (handles invalid identifiers like $dollar, 123numeric)
		fieldName := naming.ToGoFieldName(field.JSONName)

		// Build struct tags (json is always present, validate is optional)
		if validateTag != "" {
			fmt.Fprintf(&sb, "\t%s %s `json:%q validate:%q`\n", fieldName, goType, jsonTag, validateTag)
		} else {
			fmt.Fprintf(&sb, "\t%s %s `json:%q`\n", fieldName, goType, jsonTag)
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
		fmt.Fprintf(&sb, "// %s %s\n", typ.Name, typ.Description)
	}

	// Analyze enum value types to determine generation strategy
	category := enumutil.AnalyzeEnumValues(typ.EnumValues)
	allStrings, allNumbers := category.AllStrings, category.AllNumbers

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
	fmt.Fprintf(&sb, "type %s %s\n\n", typ.Name, enumType)

	// Generate values based on analysis
	if allStrings {
		// Use const for string-only enums
		sb.WriteString("const (\n")
		for _, val := range typ.EnumValues {
			fmt.Fprintf(&sb, "\t%s%s %s = %q\n", typ.Name, val.Name, typ.Name, val.Value)
		}
		sb.WriteString(")")
	} else if allNumbers {
		// Use const for number-only enums with numeric literals
		sb.WriteString("const (\n")
		for _, val := range typ.EnumValues {
			numVal := formatNumericValue(val.Value)
			fmt.Fprintf(&sb, "\t%s%s %s = %s\n", typ.Name, val.Name, typ.Name, numVal)
		}
		sb.WriteString(")")
	} else {
		// Use var with slice for mixed/complex enums
		fmt.Fprintf(&sb, "var %sValues = []%s{\n", typ.Name, typ.Name)
		for _, val := range typ.EnumValues {
			fmt.Fprintf(&sb, "\t%s,\n", formatEnumValue(val.Value))
		}
		sb.WriteString("}")
	}

	return sb.String(), nil
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
		fmt.Fprintf(&sb, "// %s %s\n", typ.Name, typ.Description)
	}

	// Type alias (use = for type alias, not type definition)
	fmt.Fprintf(&sb, "type %s = any", typ.Name)

	return sb.String(), nil
}

// formatEnumValue formats an enum value for Go code generation
func formatEnumValue(val any) string {
	switch v := val.(type) {
	case map[string]any:
		return formatMap(v)
	case []any:
		return formatSlice(v)
	default:
		return common.GoLiterals.FormatValue(val)
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
		fmt.Fprintf(&sb, "%q: %s", k, formatEnumValue(v))
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
