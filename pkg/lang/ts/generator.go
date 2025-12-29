package ts

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mirpo/schemagen/pkg/common"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

// Config contains TypeScript generation configuration.
type Config struct {
	DisableHeaders       bool
	DisableTimestamp     bool
	UnknownAny           bool // Use 'unknown' instead of 'any'
	AdditionalProperties bool // Add index signatures for additionalProperties
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
		CommentPrefix:    common.CommentPrefixTypeScript,
		DisableHeaders:   g.config.DisableHeaders,
		DisableTimestamp: g.config.DisableTimestamp,
	}))

	// Generate imports
	if len(imports) > 0 {
		for _, imp := range imports {
			if len(imp.TypeNames) == 0 {
				continue
			}

			sort.Strings(imp.TypeNames)

			sb.WriteString(fmt.Sprintf("import type { %s } from '%s';\n",
				strings.Join(imp.TypeNames, ", "),
				imp.ImportPath))
		}
		sb.WriteString("\n")
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
	case typegraph.KindAlias:
		return g.generateTypeAlias(typ)
	default:
		return "", fmt.Errorf("unsupported type kind: %s", typ.Kind)
	}
}

// generateInterface generates a TypeScript interface or intersection type.
func (g *Generator) generateInterface(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// JSDoc comment
	if typ.Description != "" {
		sb.WriteString("/**\n")
		sb.WriteString(fmt.Sprintf(" * %s\n", typ.Description))
		sb.WriteString(" */\n")
	}

	// If we have allOf composition (extends), use intersection types
	if len(typ.Extends) > 0 {
		// Generate as: export type Name = Base1 & Base2 & { fields }
		sb.WriteString(fmt.Sprintf("export type %s = ", typ.Name))

		// Add base types
		for _, base := range typ.Extends {
			sb.WriteString(base)
			sb.WriteString(" & ")
		}

		// Add inline object with fields
		sb.WriteString("{\n")
		for _, field := range typ.Fields {
			// Field comment
			if field.Description != "" {
				sb.WriteString(fmt.Sprintf("  /** %s */\n", field.Description))
			}

			tsType := g.typeRefToTS(field.Type)
			optional := ""
			if !field.Required {
				optional = "?"
			}

			// Quote property name if it's not a valid identifier
			propName := field.JSONName
			if needsQuoting(propName) {
				propName = fmt.Sprintf("%q", propName)
			}

			sb.WriteString(fmt.Sprintf("  %s%s: %s;\n", propName, optional, tsType))
		}

		// Add index signature if AdditionalProperties is configured and flag is enabled
		if indexSig := g.generateIndexSignature(typ); indexSig != "" {
			sb.WriteString(indexSig)
		}

		sb.WriteString("};")
	} else {
		// Regular interface
		sb.WriteString(fmt.Sprintf("export interface %s {\n", typ.Name))

		// Fields
		for _, field := range typ.Fields {
			// Field comment with format annotation if present
			hasDescription := field.Description != ""
			hasFormat := field.Type != nil && field.Type.Format != ""

			if hasDescription || hasFormat {
				if hasDescription && hasFormat {
					// Both description and format
					sb.WriteString("  /**\n")
					sb.WriteString(fmt.Sprintf("   * %s\n", field.Description))
					sb.WriteString(fmt.Sprintf("   * @format %s\n", field.Type.Format))
					sb.WriteString("   */\n")
				} else if hasDescription {
					// Description only
					sb.WriteString(fmt.Sprintf("  /** %s */\n", field.Description))
				} else {
					// Format only
					sb.WriteString("  /**\n")
					sb.WriteString(fmt.Sprintf("   * @format %s\n", field.Type.Format))
					sb.WriteString("   */\n")
				}
			}

			tsType := g.typeRefToTS(field.Type)
			optional := ""
			if !field.Required {
				optional = "?"
			}

			// Quote property name if it's not a valid identifier
			propName := field.JSONName
			if needsQuoting(propName) {
				propName = fmt.Sprintf("%q", propName)
			}

			sb.WriteString(fmt.Sprintf("  %s%s: %s;\n", propName, optional, tsType))
		}

		// Add index signature if AdditionalProperties is configured and flag is enabled
		if indexSig := g.generateIndexSignature(typ); indexSig != "" {
			sb.WriteString(indexSig)
		}

		sb.WriteString("}")
	}

	return sb.String(), nil
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
	if typ.Description != "" {
		sb.WriteString("/**\n")
		sb.WriteString(fmt.Sprintf(" * %s\n", typ.Description))
		sb.WriteString(" */\n")
	}

	// Check if this is a mixed-type enum (has different value types)
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
	isMixed := (hasString && hasNumber) || (hasString && hasOther) || (hasNumber && hasOther) || hasOther

	// For mixed-type enums or string enums, use union types (more idiomatic in TS)
	if typ.EnumType == "string" || isMixed {
		sb.WriteString(fmt.Sprintf("export type %s = ", typ.Name))

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
		sb.WriteString(fmt.Sprintf("export enum %s {\n", typ.Name))
		for i, val := range typ.EnumValues {
			if i > 0 {
				sb.WriteString(",\n")
			}
			// Sanitize enum member name (prefix numeric names with N_)
			memberName := val.Name
			if len(memberName) > 0 && memberName[0] >= '0' && memberName[0] <= '9' {
				memberName = "N_" + memberName
			}
			sb.WriteString(fmt.Sprintf("  %s = %v", memberName, val.Value))
		}
		sb.WriteString("\n}")
	}

	return sb.String(), nil
}

// generatePrimitiveAlias generates a type alias for a primitive type.
func (g *Generator) generatePrimitiveAlias(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// JSDoc comment
	if typ.Description != "" {
		sb.WriteString("/**\n")
		sb.WriteString(fmt.Sprintf(" * %s\n", typ.Description))
		sb.WriteString(" */\n")
	}

	// Generate type alias
	tsType := g.primitiveToTS(typ.GoType)
	sb.WriteString(fmt.Sprintf("export type %s = %s;", typ.Name, tsType))

	return sb.String(), nil
}

// generateUnionAlias generates a type alias for a union type.
func (g *Generator) generateUnionAlias(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// JSDoc comment
	if typ.Description != "" {
		sb.WriteString("/**\n")
		sb.WriteString(fmt.Sprintf(" * %s\n", typ.Description))
		sb.WriteString(" */\n")
	}

	// Generate union type from TargetType if it's a union
	// The type graph should have stored the union members in TargetType
	if typ.TargetType != nil && typ.TargetType.Kind == typegraph.KindUnion {
		tsType := g.typeRefToTS(typ.TargetType)
		sb.WriteString(fmt.Sprintf("export type %s = %s;", typ.Name, tsType))
	} else {
		// Fallback to any/unknown if we don't have proper union information
		sb.WriteString(fmt.Sprintf("export type %s = %s;", typ.Name, g.anyType()))
	}

	return sb.String(), nil
}

// generateTypeAlias generates a type alias for an alias type.
func (g *Generator) generateTypeAlias(typ *typegraph.Type) (string, error) {
	var sb strings.Builder

	// JSDoc comment
	if typ.Description != "" {
		sb.WriteString("/**\n")
		sb.WriteString(fmt.Sprintf(" * %s\n", typ.Description))
		sb.WriteString(" */\n")
	}

	// Generate type alias
	if typ.TargetType != nil {
		tsType := g.typeRefToTS(typ.TargetType)
		sb.WriteString(fmt.Sprintf("export type %s = %s;", typ.Name, tsType))
	} else {
		sb.WriteString(fmt.Sprintf("export type %s = %s;", typ.Name, g.anyType()))
	}

	return sb.String(), nil
}

// anyType returns 'any' or 'unknown' based on config.
