package golang

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/mirpo/schemagen/pkg/graph"
	"github.com/mirpo/schemagen/pkg/render"
)

const validatorImport = "github.com/go-playground/validator/v10"

type Generator struct {
	config  *Config
	imports map[string]string // import path → alias ("" for normal, "_" for blank)
}

type Config struct {
	PackageName      string
	UsePointers      bool // Use pointers for optional fields
	OmitEmpty        bool // Add omitempty to optional fields
	DisableComments  bool // Don't generate comments
	DisableHeaders   bool // Don't generate file headers
	DisableTimestamp bool // Don't generate timestamp in headers
}

func NewGenerator(cfg *Config) *Generator {
	if cfg == nil {
		cfg = &Config{
			PackageName: "models",
			UsePointers: true,
			OmitEmpty:   true,
		}
	}
	return &Generator{
		config:  cfg,
		imports: make(map[string]string),
	}
}

func (g *Generator) GenerateFile(types []*graph.Type, fileImports []graph.ImportSpec) (string, error) {
	var sb strings.Builder

	sb.WriteString(render.GenerateHeader(render.HeaderConfig{
		CommentPrefix:    render.CommentPrefixSlash,
		DisableHeaders:   g.config.DisableHeaders,
		DisableTimestamp: g.config.DisableTimestamp,
	}))

	fmt.Fprintf(&sb, "package %s\n\n", g.config.PackageName)

	g.resetImports()

	for _, typ := range types {
		g.scanTypeForImports(typ)
	}

	for _, fileImport := range fileImports {
		if fileImport.ImportPath != "" {
			g.imports[fileImport.ImportPath] = ""
		}
	}

	if len(g.imports) > 0 {
		sb.WriteString("import (\n")
		for _, imp := range slices.Sorted(maps.Keys(g.imports)) {
			if alias := g.imports[imp]; alias != "" {
				fmt.Fprintf(&sb, "\t%s %q\n", alias, imp)
			} else {
				fmt.Fprintf(&sb, "\t%q\n", imp)
			}
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

func (g *Generator) generateType(typ *graph.Type) (string, error) {
	switch typ.Kind {
	case graph.KindStruct:
		return g.generateStruct(typ)
	case graph.KindEnum:
		return g.generateEnum(typ)
	case graph.KindUnion:
		return g.generateUnion(typ)
	case graph.KindPrimitive:
		return "", nil
	default:
		return "", fmt.Errorf("unsupported type kind: %s", typ.Kind)
	}
}

func (g *Generator) writeTypeComment(sb *strings.Builder, typ *graph.Type) {
	if !g.config.DisableComments && typ.Description != "" {
		fmt.Fprintf(sb, "// %s %s\n", typ.Name, typ.Description)
	}
}

func (g *Generator) generateStruct(typ *graph.Type) (string, error) {
	var sb strings.Builder

	g.writeTypeComment(&sb, typ)

	fmt.Fprintf(&sb, "type %s struct {\n", typ.Name)

	// Embedded base types (for allOf composition)
	for _, baseType := range typ.Extends {
		fmt.Fprintf(&sb, "\t%s\n", baseType)
	}

	for _, field := range typ.Fields {
		if !g.config.DisableComments {
			if field.Description != "" {
				fmt.Fprintf(&sb, "\t// %s\n", field.Description)
			}
			if field.Type != nil && field.Type.Kind == graph.KindUnion && len(field.Type.UnionMembers) > 0 {
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
		fieldName := toGoFieldName(field.JSONName)

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

func (g *Generator) generateEnum(typ *graph.Type) (string, error) {
	var sb strings.Builder

	g.writeTypeComment(&sb, typ)

	category := graph.AnalyzeEnumValues(typ.EnumValues)
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
			numVal := render.GoLiterals.FormatValue(val.Value)
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

func (g *Generator) generateUnion(typ *graph.Type) (string, error) {
	var sb strings.Builder

	g.writeTypeComment(&sb, typ)

	// Type alias (use = for type alias, not type definition)
	fmt.Fprintf(&sb, "type %s = any", typ.Name)

	return sb.String(), nil
}

func formatEnumValue(val any) string {
	switch v := val.(type) {
	case map[string]any:
		return formatMap(v)
	case []any:
		return formatSlice(v)
	default:
		return render.GoLiterals.FormatValue(val)
	}
}

func formatMap(m map[string]any) string {
	var sb strings.Builder
	sb.WriteString("map[string]any{")
	for i, k := range slices.Sorted(maps.Keys(m)) {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%q: %s", k, formatEnumValue(m[k]))
	}
	sb.WriteString("}")
	return sb.String()
}

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
