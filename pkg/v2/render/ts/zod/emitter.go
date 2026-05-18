package zod

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/lang/tscommon"
	"github.com/mirpo/schemagen/pkg/v2/graph"
)

type Config struct {
	CoerceDates bool // Use z.coerce.date() for datetime
	Strict      bool // Add .strict() to objects
}

type Emitter struct {
	config      *Config
	currentType string
}

func NewEmitter(cfg *Config) *Emitter {
	if cfg == nil {
		cfg = &Config{}
	}
	return &Emitter{config: cfg}
}

func (e *Emitter) GenerateSchema(typ *graph.Type) string {
	return e.generateSchemaInternal(typ, false)
}

func (e *Emitter) GenerateSchemaWithInfer(typ *graph.Type) string {
	return e.generateSchemaInternal(typ, true)
}

func (e *Emitter) generateSchemaInternal(typ *graph.Type, withInfer bool) string {
	var sb strings.Builder
	schemaName := typ.Name + "Schema"

	e.currentType = typ.Name
	defer func() { e.currentType = "" }()

	if withInfer {
		tscommon.WriteJSDoc(&sb, "", typ.Description)
	}

	switch typ.Kind {
	case graph.KindStruct:
		sb.WriteString(e.generateObjectSchema(typ, schemaName))
	case graph.KindEnum:
		sb.WriteString(e.generateEnumSchema(typ, schemaName))
	case graph.KindUnion:
		sb.WriteString(e.generateUnionSchema(typ, schemaName))
	case graph.KindPrimitive:
		sb.WriteString(e.generatePrimitiveSchema(typ, schemaName))
	}

	if withInfer {
		fmt.Fprintf(&sb, "\n\nexport type %s = z.infer<typeof %s>;", typ.Name, schemaName)
	}

	return sb.String()
}

func (e *Emitter) generateObjectSchema(typ *graph.Type, schemaName string) string {
	var sb strings.Builder

	objectFunc, catchall := e.determineObjectMode(typ.AdditionalProps)

	if len(typ.Extends) > 0 {
		baseSchema := typ.Extends[0] + "Schema"
		for _, ext := range typ.Extends[1:] {
			baseSchema = fmt.Sprintf("%s.merge(%sSchema)", baseSchema, ext)
		}

		if len(typ.Fields) > 0 {
			fmt.Fprintf(&sb, "export const %s = %s.extend({\n", schemaName, baseSchema)
			for _, field := range typ.Fields {
				fmt.Fprintf(&sb, "  %s,\n", e.generateField(field))
			}
			sb.WriteString("})")
		} else {
			fmt.Fprintf(&sb, "export const %s = %s", schemaName, baseSchema)
		}
		sb.WriteString(catchall)
	} else {
		fmt.Fprintf(&sb, "export const %s = %s({\n", schemaName, objectFunc)
		for _, field := range typ.Fields {
			fmt.Fprintf(&sb, "  %s,\n", e.generateField(field))
		}
		sb.WriteString("})")
		sb.WriteString(catchall)
	}

	sb.WriteString(metaSuffix(typ.Description))
	sb.WriteString(";")
	return sb.String()
}

func (e *Emitter) determineObjectMode(additionalProps *graph.AdditionalPropsConfig) (objectFunc, catchall string) {
	if additionalProps != nil {
		if additionalProps.Allowed {
			if additionalProps.Type != nil {
				return "z.object", fmt.Sprintf(".catchall(%s)", e.typeRefToZod(additionalProps.Type, nil))
			}
			return "z.looseObject", ""
		}
		return "z.strictObject", ""
	}

	if e.config.Strict {
		return "z.strictObject", ""
	}

	return "z.object", ""
}

func (e *Emitter) generateEnumSchema(typ *graph.Type, schemaName string) string {
	category := graph.AnalyzeEnumValues(typ.EnumValues)

	var schema string
	if category.AllStrings && len(typ.EnumValues) > 0 {
		values := make([]string, len(typ.EnumValues))
		for i, ev := range typ.EnumValues {
			values[i] = formatLiteral(ev.Value)
		}
		schema = fmt.Sprintf("z.enum([%s])", strings.Join(values, ", "))
	} else {
		var sb strings.Builder
		sb.WriteString("z.union([\n")
		for _, ev := range typ.EnumValues {
			fmt.Fprintf(&sb, "  %s,\n", formatZodLiteral(ev.Value))
		}
		sb.WriteString("])")
		schema = sb.String()
	}

	return wrapSchemaExport(schemaName, schema, typ.Description)
}

func (e *Emitter) generateUnionSchema(typ *graph.Type, schemaName string) string {
	if len(typ.UnionMembers) > 0 {
		members := make([]string, len(typ.UnionMembers))
		for i, member := range typ.UnionMembers {
			members[i] = e.typeRefToZod(member, nil)
		}
		return wrapSchemaExport(schemaName, fmt.Sprintf("z.union([%s])", strings.Join(members, ", ")), typ.Description)
	}

	return fmt.Sprintf("export const %s = z.unknown();", schemaName)
}

func (e *Emitter) generatePrimitiveSchema(typ *graph.Type, schemaName string) string {
	return wrapSchemaExport(schemaName, e.primitiveToZod(typ.Primitive, nil), typ.Description)
}

func wrapSchemaExport(schemaName, schema, description string) string {
	return fmt.Sprintf("export const %s = %s", schemaName, schema) + metaSuffix(description) + ";"
}

func metaSuffix(description string) string {
	if description == "" {
		return ""
	}
	return fmt.Sprintf(".meta({ description: %q })", description)
}

func (e *Emitter) generateField(field *graph.Field) string {
	propName := field.JSONName
	if tscommon.NeedsQuoting(propName) {
		propName = fmt.Sprintf(`"%s"`, propName)
	}

	zodType := e.typeRefToZod(field.Type, field)

	if !field.Required {
		zodType += ".optional()"
	}

	zodType += metaSuffix(field.Description)

	if e.containsSelfReference(field.Type) {
		return fmt.Sprintf("get %s() { return %s }", propName, zodType)
	}

	return fmt.Sprintf("%s: %s", propName, zodType)
}

func (e *Emitter) containsSelfReference(ref *graph.TypeRef) bool {
	if ref == nil || e.currentType == "" {
		return false
	}

	switch ref.Kind {
	case graph.KindRef:
		return ref.TypeName == e.currentType
	case graph.KindArray:
		return e.containsSelfReference(ref.ItemType)
	case graph.KindMap:
		return e.containsSelfReference(ref.ValueType)
	case graph.KindUnion:
		for _, member := range ref.UnionMembers {
			if e.containsSelfReference(member) {
				return true
			}
		}
	}
	return false
}
