package ts

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mirpo/schemagen/pkg/graph"
	"github.com/mirpo/schemagen/pkg/render"
	"github.com/mirpo/schemagen/pkg/render/ts/tsutil"
	"github.com/mirpo/schemagen/pkg/render/ts/zod"
)

type ZodMode int

const (
	ZodModeOff ZodMode = iota
	ZodModeWithInterface
	ZodModeOnly
)

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

type Generator struct {
	config *Config
}

func NewGeneratorWithConfig(cfg *Config) *Generator {
	if cfg == nil {
		cfg = &Config{}
	}
	return &Generator{
		config: cfg,
	}
}

func (g *Generator) GenerateFile(types []*graph.Type, imports []graph.ImportSpec) (string, error) {
	var sb strings.Builder

	sb.WriteString(render.GenerateHeader(render.HeaderConfig{
		CommentPrefix:    render.CommentPrefixSlash,
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

func (g *Generator) writeImports(sb *strings.Builder, imports []graph.ImportSpec) {
	if len(imports) == 0 {
		return
	}

	for _, imp := range imports {
		if len(imp.TypeNames) == 0 {
			continue
		}

		names := slices.Sorted(slices.Values(imp.TypeNames))

		switch g.config.ZodMode {
		case ZodModeOnly:
			schemaNames := toSchemaNames(names)
			fmt.Fprintf(sb, "import { %s } from '%s';\n",
				strings.Join(schemaNames, ", "),
				imp.ImportPath)

		case ZodModeWithInterface:
			fmt.Fprintf(sb, "import type { %s } from '%s';\n",
				strings.Join(names, ", "),
				imp.ImportPath)
			schemaNames := toSchemaNames(names)
			fmt.Fprintf(sb, "import { %s } from '%s';\n",
				strings.Join(schemaNames, ", "),
				imp.ImportPath)

		default:
			fmt.Fprintf(sb, "import type { %s } from '%s';\n",
				strings.Join(names, ", "),
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

func (g *Generator) writeType(sb *strings.Builder, typ *graph.Type, zodEmitter *zod.Emitter) error {
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

func (g *Generator) generateType(typ *graph.Type) (string, error) {
	switch typ.Kind {
	case graph.KindStruct:
		return g.generateInterface(typ)
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

func (g *Generator) generateInterface(typ *graph.Type) (string, error) {
	var sb strings.Builder

	tsutil.WriteJSDoc(&sb, "", typ.Description)

	if len(typ.Extends) > 0 {
		fmt.Fprintf(&sb, "export type %s = ", typ.Name)

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

func (g *Generator) renderField(sb *strings.Builder, field *graph.Field, indent string, withFormat bool) {
	if withFormat {
		format := ""
		if field.Type != nil {
			format = field.Type.Format
		}
		tsutil.WriteJSDocWithFormat(sb, indent, field.Description, format)
	} else {
		tsutil.WriteJSDocSingleLine(sb, indent, field.Description)
	}

	tsType := g.typeRefToTS(field.Type)
	optional := ""
	if !field.Required {
		optional = "?"
	}

	propName := field.JSONName
	if tsutil.NeedsQuoting(propName) {
		propName = fmt.Sprintf("%q", propName)
	}

	fmt.Fprintf(sb, "%s%s%s: %s;\n", indent, propName, optional, tsType)
}

func (g *Generator) writeFields(sb *strings.Builder, typ *graph.Type, withFormat bool) {
	for _, field := range typ.Fields {
		g.renderField(sb, field, "  ", withFormat)
	}

	if indexSig := g.generateIndexSignature(typ); indexSig != "" {
		sb.WriteString(indexSig)
	}
}

func (g *Generator) generateIndexSignature(typ *graph.Type) string {
	if !g.config.AdditionalProperties {
		return ""
	}

	if typ.AdditionalProps == nil {
		return ""
	}

	if !typ.AdditionalProps.Allowed {
		return ""
	}

	var valueType string
	if typ.AdditionalProps.Type != nil {
		valueType = g.typeRefToTS(typ.AdditionalProps.Type)
	} else {
		valueType = g.anyType()
	}

	return fmt.Sprintf("  [key: string]: %s;\n", valueType)
}

func (g *Generator) generateEnum(typ *graph.Type) (string, error) {
	var sb strings.Builder

	tsutil.WriteJSDoc(&sb, "", typ.Description)

	category := graph.AnalyzeEnumValues(typ.EnumValues)

	if category.AllStrings || category.HasMixed {
		fmt.Fprintf(&sb, "export type %s = ", typ.Name)

		values := make([]string, 0, len(typ.EnumValues))
		hasComplexValues := false
		for _, val := range typ.EnumValues {
			switch val.Value.(type) {
			case string, float64, int, int64, int32, bool, nil:
				values = append(values, render.TSLiterals.FormatValue(val.Value))
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
			fmt.Fprintf(&sb, "  %s = %v", render.SanitizeEnumMember(val.Name), val.Value)
		}
		sb.WriteString("\n}")
	}

	return sb.String(), nil
}

func (g *Generator) generatePrimitiveAlias(typ *graph.Type) (string, error) {
	var sb strings.Builder

	tsutil.WriteJSDoc(&sb, "", typ.Description)

	tsType := g.primitiveToTS(typ.Primitive)
	fmt.Fprintf(&sb, "export type %s = %s;", typ.Name, tsType)

	return sb.String(), nil
}

func (g *Generator) generateUnionAlias(typ *graph.Type) (string, error) {
	var sb strings.Builder

	tsutil.WriteJSDoc(&sb, "", typ.Description)

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
