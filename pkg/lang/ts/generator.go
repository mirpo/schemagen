package ts

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mirpo/schemagen/pkg/common"
	"github.com/mirpo/schemagen/pkg/enumutil"
	"github.com/mirpo/schemagen/pkg/lang/ts/zod"
	"github.com/mirpo/schemagen/pkg/lang/tscommon"
	"github.com/mirpo/schemagen/pkg/naming"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

// ZodMode represents the Zod generation mode
type ZodMode int

const (
	// ZodModeOff disables Zod generation (default, interfaces only)
	ZodModeOff ZodMode = iota
	// ZodModeWithInterface generates both interfaces and Zod schemas
	ZodModeWithInterface
	// ZodModeOnly generates only Zod schemas with z.infer<> types
	ZodModeOnly
)

// Config contains TypeScript generation configuration.
type Config struct {
	DisableHeaders       bool
	DisableTimestamp     bool
	UnknownAny           bool // Use 'unknown' instead of 'any'
	AdditionalProperties bool // Add index signatures for additionalProperties

	// Zod configuration
	ZodMode        ZodMode // Zod generation mode
	ZodCoerceDates bool    // Use z.coerce.date() for date-time
	ZodStrict      bool    // Add .strict() to object schemas
}

// Generator generates TypeScript code from a type graph.
type Generator struct {
	config *Config
}

// NewGenerator creates a new TypeScript generator.
func NewGenerator() *Generator {
	return NewGeneratorWithConfig(&Config{})
}

// NewGeneratorWithConfig creates a TypeScript generator with custom config.
func NewGeneratorWithConfig(cfg *Config) *Generator {
	if cfg == nil {
		cfg = &Config{}
	}
	return &Generator{
		config: cfg,
	}
}

// GenerateFile produces TypeScript source code for a specific set of types with imports.
func (g *Generator) GenerateFile(types []*typegraph.Type, imports []typegraph.ImportSpec) (string, error) {
	var sb strings.Builder

	sb.WriteString(common.GenerateHeader(common.HeaderConfig{
		CommentPrefix:    common.CommentPrefixSlash,
		DisableHeaders:   g.config.DisableHeaders,
		DisableTimestamp: g.config.DisableTimestamp,
	}))

	if g.config.ZodMode != ZodModeOff {
		sb.WriteString("import { z } from 'zod';\n\n")
	}

	g.writeImports(&sb, imports)

	var zodEmitter *zod.Emitter
	if g.config.ZodMode != ZodModeOff {
		zodEmitter = zod.NewEmitter(&zod.Config{
			CoerceDates: g.config.ZodCoerceDates,
			Strict:      g.config.ZodStrict,
		})
	}

	for i, typ := range types {
		if i > 0 {
			sb.WriteString("\n\n")
		}

		if err := g.writeType(&sb, typ, zodEmitter); err != nil {
			return "", err
		}
	}

	sb.WriteString("\n")
	return sb.String(), nil
}

func (g *Generator) writeImports(sb *strings.Builder, imports []typegraph.ImportSpec) {
	if len(imports) == 0 {
		return
	}

	for _, imp := range imports {
		if len(imp.TypeNames) == 0 {
			continue
		}

		sort.Strings(imp.TypeNames)

		switch g.config.ZodMode {
		case ZodModeOnly:
			schemaNames := toSchemaNames(imp.TypeNames)
			fmt.Fprintf(sb, "import { %s } from '%s';\n",
				strings.Join(schemaNames, ", "),
				imp.ImportPath)

		case ZodModeWithInterface:
			fmt.Fprintf(sb, "import type { %s } from '%s';\n",
				strings.Join(imp.TypeNames, ", "),
				imp.ImportPath)
			schemaNames := toSchemaNames(imp.TypeNames)
			fmt.Fprintf(sb, "import { %s } from '%s';\n",
				strings.Join(schemaNames, ", "),
				imp.ImportPath)

		default:
			fmt.Fprintf(sb, "import type { %s } from '%s';\n",
				strings.Join(imp.TypeNames, ", "),
				imp.ImportPath)
		}
	}
	sb.WriteString("\n")
}

func toSchemaNames(typeNames []string) []string {
	names := make([]string, len(typeNames))
	for i, name := range typeNames {
		names[i] = name + "Schema"
	}
	return names
}

func (g *Generator) writeType(sb *strings.Builder, typ *typegraph.Type, zodEmitter *zod.Emitter) error {
	switch g.config.ZodMode {
	case ZodModeOff:
		code, err := g.generateType(typ)
		if err != nil {
			return err
		}
		sb.WriteString(code)

	case ZodModeWithInterface:
		code, err := g.generateType(typ)
		if err != nil {
			return err
		}
		sb.WriteString(code)
		sb.WriteString("\n\n")
		sb.WriteString(zodEmitter.GenerateSchema(typ))

	case ZodModeOnly:
		sb.WriteString(zodEmitter.GenerateSchemaWithInfer(typ))
	}
	return nil
}

// generateType generates code for a single type.
func (g *Generator) generateType(typ *typegraph.Type) (string, error) {
	switch typ.Kind {
	case typegraph.KindStruct:
		return g.generateInterface(typ)
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

// generateInterface generates a TypeScript interface or intersection type.
func (g *Generator) generateInterface(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// JSDoc comment
	tscommon.WriteJSDoc(&sb, "", typ.Description)

	// If we have allOf composition (extends), use intersection types
	if len(typ.Extends) > 0 {
		// Generate as: export type Name = Base1 & Base2 & { fields }
		fmt.Fprintf(&sb, "export type %s = ", typ.Name)

		// Add base types
		for _, base := range typ.Extends {
			sb.WriteString(base)
			sb.WriteString(" & ")
		}

		sb.WriteString("{\n")
		g.writeFields(&sb, typ, false)
		sb.WriteString("};")
	} else {
		fmt.Fprintf(&sb, "export interface %s {\n", typ.Name)
		g.writeFields(&sb, typ, true)
		sb.WriteString("}")
	}

	return sb.String(), nil
}

func (g *Generator) renderField(sb *strings.Builder, field *typegraph.Field, indent string, withFormat bool) {
	if withFormat {
		format := ""
		if field.Type != nil {
			format = field.Type.Format
		}
		tscommon.WriteJSDocWithFormat(sb, indent, field.Description, format)
	} else {
		tscommon.WriteJSDocSingleLine(sb, indent, field.Description)
	}

	tsType := g.typeRefToTS(field.Type)
	optional := ""
	if !field.Required {
		optional = "?"
	}

	propName := field.JSONName
	if tscommon.NeedsQuoting(propName) {
		propName = fmt.Sprintf("%q", propName)
	}

	fmt.Fprintf(sb, "%s%s%s: %s;\n", indent, propName, optional, tsType)
}

func (g *Generator) writeFields(sb *strings.Builder, typ *typegraph.Type, withFormat bool) {
	for _, field := range typ.Fields {
		g.renderField(sb, field, "  ", withFormat)
	}

	if indexSig := g.generateIndexSignature(typ); indexSig != "" {
		sb.WriteString(indexSig)
	}
}

// generateIndexSignature generates an index signature for additionalProperties if applicable.
func (g *Generator) generateIndexSignature(typ *typegraph.Type) string {
	// Only add index signatures if:
	// 1. The flag is enabled
	// 2. The type has AdditionalProperties configured
	// 3. AdditionalProperties.Allowed is true
	if !g.config.AdditionalProperties {
		return ""
	}

	if typ.AdditionalProps == nil {
		return ""
	}

	if !typ.AdditionalProps.Allowed {
		return ""
	}

	// Determine the value type for the index signature
	var valueType string
	if typ.AdditionalProps.Type != nil {
		// Typed additional properties (e.g., { type: "number" })
		valueType = g.typeRefToTS(typ.AdditionalProps.Type)
	} else {
		// Untyped additional properties (additionalProperties: true)
		valueType = g.anyType()
	}

	return fmt.Sprintf("  [key: string]: %s;\n", valueType)
}

// generateEnum generates a TypeScript enum or union type.
func (g *Generator) generateEnum(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// JSDoc comment
	tscommon.WriteJSDoc(&sb, "", typ.Description)

	// Check if this is a mixed-type enum using shared analyzer
	category := enumutil.AnalyzeEnumValues(typ.EnumValues)

	// For mixed-type enums or string enums, use union types (more idiomatic in TS)
	if typ.EnumType == "string" || category.HasMixed {
		fmt.Fprintf(&sb, "export type %s = ", typ.Name)

		values := make([]string, 0, len(typ.EnumValues))
		hasComplexValues := false
		for _, val := range typ.EnumValues {
			switch val.Value.(type) {
			case string, float64, int, int64, int32, bool, nil:
				values = append(values, common.TSLiterals.FormatValue(val.Value))
			default:
				hasComplexValues = true
			}
		}
		if hasComplexValues {
			values = append(values, g.anyType())
		}
		sb.WriteString(strings.Join(values, " | "))
		sb.WriteString(";")
	} else {
		fmt.Fprintf(&sb, "export enum %s {\n", typ.Name)
		for i, val := range typ.EnumValues {
			if i > 0 {
				sb.WriteString(",\n")
			}
			fmt.Fprintf(&sb, "  %s = %v", naming.SanitizeEnumMember(val.Name), val.Value)
		}
		sb.WriteString("\n}")
	}

	return sb.String(), nil
}

// generatePrimitiveAlias generates a type alias for a primitive type.
func (g *Generator) generatePrimitiveAlias(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// JSDoc comment
	tscommon.WriteJSDoc(&sb, "", typ.Description)

	// Generate type alias
	tsType := g.primitiveToTS(typ.Primitive)
	fmt.Fprintf(&sb, "export type %s = %s;", typ.Name, tsType)

	return sb.String(), nil
}

func (g *Generator) generateUnionAlias(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	tscommon.WriteJSDoc(&sb, "", typ.Description)

	tsType := g.anyType()
	if len(typ.UnionMembers) > 0 {
		members := make([]string, len(typ.UnionMembers))
		for i, m := range typ.UnionMembers {
			members[i] = g.typeRefToTS(m)
		}
		tsType = strings.Join(members, " | ")
	}
	fmt.Fprintf(&sb, "export type %s = %s;", typ.Name, tsType)

	return sb.String(), nil
}

// anyType returns 'any' or 'unknown' based on config.
