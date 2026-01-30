package zod

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/typegraph"
)

// Config holds Zod generation configuration.
type Config struct {
	CoerceDates bool // Use z.coerce.date() for datetime
	Strict      bool // Add .strict() to objects
}

// Emitter generates Zod schemas from types.
type Emitter struct {
	graph       *typegraph.Graph
	config      *Config
	currentType string // Track current type for detecting self-references
}

// NewEmitter creates a new Zod emitter.
func NewEmitter(graph *typegraph.Graph, cfg *Config) *Emitter {
	if cfg == nil {
		cfg = &Config{}
	}
	return &Emitter{graph: graph, config: cfg}
}

// GenerateSchema generates a Zod schema for a type (without type export).
func (e *Emitter) GenerateSchema(typ *typegraph.Type) string {
	return e.generateSchemaInternal(typ, false)
}

// GenerateSchemaWithInfer generates a Zod schema with z.infer type export.
func (e *Emitter) GenerateSchemaWithInfer(typ *typegraph.Type) string {
	return e.generateSchemaInternal(typ, true)
}

func (e *Emitter) generateSchemaInternal(typ *typegraph.Type, withInfer bool) string {
	var sb strings.Builder
	schemaName := typ.Name + "Schema"

	// Track current type for detecting self-references
	e.currentType = typ.Name
	defer func() { e.currentType = "" }()

	// JSDoc comment (for zod-only mode)
	if withInfer && typ.Description != "" {
		sb.WriteString("/**\n")
		sb.WriteString(fmt.Sprintf(" * %s\n", typ.Description))
		sb.WriteString(" */\n")
	}

	// Generate schema based on type kind
	switch typ.Kind {
	case typegraph.KindStruct:
		sb.WriteString(e.generateObjectSchema(typ, schemaName))
	case typegraph.KindEnum:
		sb.WriteString(e.generateEnumSchema(typ, schemaName))
	case typegraph.KindUnion:
		sb.WriteString(e.generateUnionSchema(typ, schemaName))
	case typegraph.KindAlias:
		sb.WriteString(e.generateAliasSchema(typ, schemaName))
	case typegraph.KindPrimitive:
		sb.WriteString(e.generatePrimitiveSchema(typ, schemaName))
	}

	// Add z.infer type export if requested
	if withInfer {
		sb.WriteString(fmt.Sprintf("\n\nexport type %s = z.infer<typeof %s>;", typ.Name, schemaName))
	}

	return sb.String()
}

// generateObjectSchema generates a Zod object schema.
func (e *Emitter) generateObjectSchema(typ *typegraph.Type, schemaName string) string {
	var sb strings.Builder

	// Handle allOf (inheritance via merge/extend)
	if len(typ.Extends) > 0 {
		baseSchema := typ.Extends[0] + "Schema"
		for _, ext := range typ.Extends[1:] {
			baseSchema = fmt.Sprintf("%s.merge(%sSchema)", baseSchema, ext)
		}

		if len(typ.Fields) > 0 {
			sb.WriteString(fmt.Sprintf("export const %s = %s.extend({\n", schemaName, baseSchema))
			for _, field := range typ.Fields {
				sb.WriteString(fmt.Sprintf("  %s,\n", e.generateField(field)))
			}
			sb.WriteString("})")
		} else {
			sb.WriteString(fmt.Sprintf("export const %s = %s", schemaName, baseSchema))
		}
	} else {
		// Regular object
		sb.WriteString(fmt.Sprintf("export const %s = z.object({\n", schemaName))
		for _, field := range typ.Fields {
			sb.WriteString(fmt.Sprintf("  %s,\n", e.generateField(field)))
		}
		sb.WriteString("})")
	}

	// Add .strict() if configured
	if e.config.Strict {
		sb.WriteString(".strict()")
	}

	// Add metadata
	if typ.Description != "" {
		sb.WriteString(fmt.Sprintf(".meta({ description: %q })", typ.Description))
	}

	sb.WriteString(";")
	return sb.String()
}

// generateEnumSchema generates a Zod enum schema.
func (e *Emitter) generateEnumSchema(typ *typegraph.Type, schemaName string) string {
	values := make([]string, len(typ.EnumValues))
	allStrings := true

	for i, ev := range typ.EnumValues {
		if _, ok := ev.Value.(string); !ok {
			allStrings = false
		}
		values[i] = formatLiteral(ev.Value)
	}

	var schema string
	if allStrings && len(typ.EnumValues) > 0 {
		// Use z.enum for string enums
		schema = fmt.Sprintf("z.enum([%s])", strings.Join(values, ", "))
	} else {
		// Use union of literals for mixed types
		literals := make([]string, len(values))
		for i, v := range values {
			literals[i] = fmt.Sprintf("z.literal(%s)", v)
		}
		schema = fmt.Sprintf("z.union([%s])", strings.Join(literals, ", "))
	}

	result := fmt.Sprintf("export const %s = %s", schemaName, schema)

	if typ.Description != "" {
		result += fmt.Sprintf(".meta({ description: %q })", typ.Description)
	}

	return result + ";"
}

// generateUnionSchema generates a Zod union schema.
func (e *Emitter) generateUnionSchema(typ *typegraph.Type, schemaName string) string {
	if typ.TargetType != nil && typ.TargetType.Kind == typegraph.KindUnion {
		members := make([]string, len(typ.TargetType.UnionMembers))
		for i, member := range typ.TargetType.UnionMembers {
			members[i] = e.typeRefToZod(member, nil)
		}

		result := fmt.Sprintf("export const %s = z.union([%s])", schemaName, strings.Join(members, ", "))

		if typ.Description != "" {
			result += fmt.Sprintf(".meta({ description: %q })", typ.Description)
		}

		return result + ";"
	}

	// Fallback for types with UnionMembers directly on Type
	if len(typ.UnionMembers) > 0 {
		members := make([]string, len(typ.UnionMembers))
		for i, member := range typ.UnionMembers {
			members[i] = e.typeRefToZod(member, nil)
		}

		result := fmt.Sprintf("export const %s = z.union([%s])", schemaName, strings.Join(members, ", "))

		if typ.Description != "" {
			result += fmt.Sprintf(".meta({ description: %q })", typ.Description)
		}

		return result + ";"
	}

	// Fallback to unknown
	return fmt.Sprintf("export const %s = z.unknown();", schemaName)
}

// generateAliasSchema generates a Zod schema for a type alias.
func (e *Emitter) generateAliasSchema(typ *typegraph.Type, schemaName string) string {
	if typ.TargetType != nil {
		zodType := e.typeRefToZod(typ.TargetType, nil)

		result := fmt.Sprintf("export const %s = %s", schemaName, zodType)

		if typ.Description != "" {
			result += fmt.Sprintf(".meta({ description: %q })", typ.Description)
		}

		return result + ";"
	}

	return fmt.Sprintf("export const %s = z.unknown();", schemaName)
}

// generatePrimitiveSchema generates a Zod schema for a primitive type alias.
func (e *Emitter) generatePrimitiveSchema(typ *typegraph.Type, schemaName string) string {
	zodType := e.primitiveToZod(typ.GoType, "", nil)

	result := fmt.Sprintf("export const %s = %s", schemaName, zodType)

	if typ.Description != "" {
		result += fmt.Sprintf(".meta({ description: %q })", typ.Description)
	}

	return result + ";"
}

// generateField generates a single field for an object schema.
func (e *Emitter) generateField(field *typegraph.Field) string {
	// Property name (quote if necessary)
	propName := field.JSONName
	if needsQuoting(propName) {
		propName = fmt.Sprintf(`"%s"`, propName)
	}

	// Generate Zod type with constraints
	zodType := e.typeRefToZod(field.Type, field)

	// Make optional if not required
	if !field.Required {
		zodType += ".optional()"
	}

	// Add field description via meta
	if field.Description != "" {
		zodType += fmt.Sprintf(".meta({ description: %q })", field.Description)
	}

	return fmt.Sprintf("%s: %s", propName, zodType)
}
