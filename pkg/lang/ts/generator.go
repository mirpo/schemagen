package ts

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mirpo/schemagen/pkg/common"
	"github.com/mirpo/schemagen/pkg/enumutil"
	"github.com/mirpo/schemagen/pkg/lang/ts/zod"
	"github.com/mirpo/schemagen/pkg/lang/tscommon"
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
	graph  *typegraph.Graph
	config *Config
}

// NewGenerator creates a new TypeScript generator.
func NewGenerator(graph *typegraph.Graph) *Generator {
	return NewGeneratorWithConfig(graph, &Config{})
}

// NewGeneratorWithConfig creates a TypeScript generator with custom config.
func NewGeneratorWithConfig(graph *typegraph.Graph, cfg *Config) *Generator {
	if cfg == nil {
		cfg = &Config{}
	}
	return &Generator{
		graph:  graph,
		config: cfg,
	}
}

// GenerateFile produces TypeScript source code for a specific set of types with imports.
func (g *Generator) GenerateFile(types []*typegraph.Type, imports []typegraph.ImportSpec) (string, error) {
	var sb strings.Builder

	// Header
	sb.WriteString(common.GenerateHeader(common.HeaderConfig{
		CommentPrefix:    common.CommentPrefixSlash,
		DisableHeaders:   g.config.DisableHeaders,
		DisableTimestamp: g.config.DisableTimestamp,
	}))

	// Add Zod import if generating Zod schemas
	if g.config.ZodMode != ZodModeOff {
		sb.WriteString("import { z } from 'zod';\n\n")
	}

	// Generate type imports (for interfaces or schema references)
	if len(imports) > 0 {
		for _, imp := range imports {
			if len(imp.TypeNames) == 0 {
				continue
			}

			sort.Strings(imp.TypeNames)

			switch g.config.ZodMode {
			case ZodModeOnly:
				// Import schemas only (types are inferred)
				schemaNames := make([]string, len(imp.TypeNames))
				for i, name := range imp.TypeNames {
					schemaNames[i] = name + "Schema"
				}
				fmt.Fprintf(&sb, "import { %s } from '%s';\n",
					strings.Join(schemaNames, ", "),
					imp.ImportPath)

			case ZodModeWithInterface:
				// Import both types (for interfaces) and schemas (for Zod)
				fmt.Fprintf(&sb, "import type { %s } from '%s';\n",
					strings.Join(imp.TypeNames, ", "),
					imp.ImportPath)
				schemaNames := make([]string, len(imp.TypeNames))
				for i, name := range imp.TypeNames {
					schemaNames[i] = name + "Schema"
				}
				fmt.Fprintf(&sb, "import { %s } from '%s';\n",
					strings.Join(schemaNames, ", "),
					imp.ImportPath)

			default:
				// ZodModeOff - import types only
				fmt.Fprintf(&sb, "import type { %s } from '%s';\n",
					strings.Join(imp.TypeNames, ", "),
					imp.ImportPath)
			}
		}
		sb.WriteString("\n")
	}

	// Create Zod emitter if needed
	var zodEmitter *zod.Emitter
	if g.config.ZodMode != ZodModeOff {
		zodEmitter = zod.NewEmitter(g.graph, &zod.Config{
			CoerceDates: g.config.ZodCoerceDates,
			Strict:      g.config.ZodStrict,
		})
	}

	// Generate types (preserving input order for correct schema references)
	for i, typ := range types {
		if i > 0 {
			sb.WriteString("\n\n")
		}

		switch g.config.ZodMode {
		case ZodModeOff:
			// Generate only interfaces
			code, err := g.generateType(typ)
			if err != nil {
				return "", err
			}
			sb.WriteString(code)

		case ZodModeWithInterface:
			// Generate interface first
			code, err := g.generateType(typ)
			if err != nil {
				return "", err
			}
			sb.WriteString(code)
			// Then generate Zod schema
			sb.WriteString("\n\n")
			sb.WriteString(zodEmitter.GenerateSchema(typ))

		case ZodModeOnly:
			// Generate only Zod schema with z.infer type
			sb.WriteString(zodEmitter.GenerateSchemaWithInfer(typ))
		}
	}

	// Add final newline at EOF
	sb.WriteString("\n")
	return sb.String(), nil
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

func (g *Generator) writeFields(sb *strings.Builder, typ *typegraph.Type, withFormat bool) {
	for _, field := range typ.Fields {
		if withFormat {
			format := ""
			if field.Type != nil {
				format = field.Type.Format
			}
			tscommon.WriteJSDocWithFormat(sb, "  ", field.Description, format)
		} else {
			tscommon.WriteJSDocSingleLine(sb, "  ", field.Description)
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

		fmt.Fprintf(sb, "  %s%s: %s;\n", propName, optional, tsType)
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
		for _, val := range typ.EnumValues {
			switch v := val.Value.(type) {
			case string:
				values = append(values, fmt.Sprintf("%q", v))
			case float64, int, int64:
				values = append(values, fmt.Sprintf("%v", v))
			case bool:
				values = append(values, fmt.Sprintf("%t", v))
			case nil:
				values = append(values, "null")
			}
		}
		sb.WriteString(strings.Join(values, " | "))
		sb.WriteString(";")
	} else {
		// For numeric-only enums, use actual enum (though less common in TS)
		fmt.Fprintf(&sb, "export enum %s {\n", typ.Name)
		for i, val := range typ.EnumValues {
			if i > 0 {
				sb.WriteString(",\n")
			}
			// Sanitize enum member name (prefix numeric names with N_)
			memberName := val.Name
			if len(memberName) > 0 && memberName[0] >= '0' && memberName[0] <= '9' {
				memberName = "N_" + memberName
			}
			fmt.Fprintf(&sb, "  %s = %v", memberName, val.Value)
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
