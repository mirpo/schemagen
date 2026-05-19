package ts

import (
	"strings"
	"testing"

	"github.com/mirpo/schemagen/pkg/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper functions

func createTestGenerator(config *Config) *Generator {
	if config == nil {
		return NewGeneratorWithConfig(nil)
	}
	return NewGeneratorWithConfig(config)
}

func createTestType(name string, kind graph.TypeKind) *graph.Type {
	return &graph.Type{
		Name:   name,
		Kind:   kind,
		Fields: []*graph.Field{},
	}
}

func createTestField(name string, jsonName string, prim graph.PrimitiveKind, required bool) *graph.Field {
	return &graph.Field{
		JSONName: jsonName,
		Type: &graph.TypeRef{
			Kind:      graph.KindPrimitive,
			Primitive: prim,
		},
		Required: required,
	}
}

func TestGenerateInterface(t *testing.T) {
	g := createTestGenerator(nil)

	t.Run("simple", func(t *testing.T) {
		typ := createTestType("User", graph.KindStruct)
		typ.Fields = []*graph.Field{createTestField("ID", "id", graph.PrimString, true), createTestField("Name", "name", graph.PrimString, true)}
		result, err := g.generateInterface(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "export interface User {")
		assert.Contains(t, result, "id: string;")
	})

	t.Run("with description", func(t *testing.T) {
		typ := createTestType("User", graph.KindStruct)
		typ.Description = "Represents a user"
		typ.Fields = []*graph.Field{createTestField("ID", "id", graph.PrimString, true)}
		result, err := g.generateInterface(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "* Represents a user")
	})

	t.Run("empty fields", func(t *testing.T) {
		typ := createTestType("Empty", graph.KindStruct)
		result, err := g.generateInterface(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "export interface Empty {")
	})

	t.Run("optional fields", func(t *testing.T) {
		typ := createTestType("User", graph.KindStruct)
		typ.Fields = []*graph.Field{createTestField("ID", "id", graph.PrimString, true), createTestField("Email", "email", graph.PrimString, false)}
		result, err := g.generateInterface(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "email?: string;")
	})
}

func TestGenerateEnum(t *testing.T) {
	g := createTestGenerator(nil)

	t.Run("string union", func(t *testing.T) {
		typ := createTestType("Status", graph.KindEnum)
		typ.EnumType = graph.EnumKindString
		typ.EnumValues = []graph.EnumValue{{Name: "Active", Value: "active"}, {Name: "Inactive", Value: "inactive"}}
		result, err := g.generateEnum(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "export type Status =")
		assert.Contains(t, result, `"active"`)
	})

	t.Run("numeric enum", func(t *testing.T) {
		typ := createTestType("Priority", graph.KindEnum)
		typ.EnumType = graph.EnumKindInt
		typ.EnumValues = []graph.EnumValue{{Name: "Low", Value: 1}, {Name: "High", Value: 2}}
		result, err := g.generateEnum(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "export enum Priority {")
		assert.Contains(t, result, "Low = 1")
	})

	t.Run("mixed types", func(t *testing.T) {
		typ := createTestType("Mixed", graph.KindEnum)
		typ.EnumType = graph.EnumKindString
		typ.EnumValues = []graph.EnumValue{{Name: "String", Value: "text"}, {Name: "Number", Value: 42}, {Name: "Null", Value: nil}}
		result, err := g.generateEnum(typ)
		require.NoError(t, err)
		assert.Contains(t, result, `"text"`)
		assert.Contains(t, result, "42")
		assert.Contains(t, result, "null")
	})

	t.Run("complex values use any fallback", func(t *testing.T) {
		typ := createTestType("Complex", graph.KindEnum)
		typ.EnumType = graph.EnumKindString
		typ.EnumValues = []graph.EnumValue{
			{Name: "Simple", Value: "simple"},
			{Name: "Obj", Value: map[string]any{"complex": true}},
			{Name: "Num", Value: 42},
		}
		result, err := g.generateEnum(typ)
		require.NoError(t, err)
		assert.Contains(t, result, `"simple"`)
		assert.Contains(t, result, "42")
		assert.Contains(t, result, "any")
	})
}

func TestGenerateUnionAlias_WithMembers(t *testing.T) {
	g := createTestGenerator(nil)

	typ := &graph.Type{
		Name: "Shape",
		Kind: graph.KindUnion,
		UnionMembers: []*graph.TypeRef{
			{Kind: graph.KindRef, TypeName: "Circle"},
			{Kind: graph.KindRef, TypeName: "Square"},
			{Kind: graph.KindRef, TypeName: "Triangle"},
		},
	}

	result, err := g.generateUnionAlias(typ)
	require.NoError(t, err)

	assert.Contains(t, result, "export type Shape = Circle | Square | Triangle;")
	assert.NotContains(t, result, "any")
}

func TestGenerateUnionAlias_EmptyMembers(t *testing.T) {
	g := createTestGenerator(nil)

	typ := &graph.Type{
		Name: "Unknown",
		Kind: graph.KindUnion,
	}

	result, err := g.generateUnionAlias(typ)
	require.NoError(t, err)

	assert.Contains(t, result, "export type Unknown = any;")
}

// Phase 1.3: Type Conversion

func TestTypeRefToTS_Primitives(t *testing.T) {
	tests := []struct {
		name     string
		prim     graph.PrimitiveKind
		expected string
	}{
		{"string type", graph.PrimString, "string"},
		{"int type", graph.PrimInt, "number"},
		{"float64 type", graph.PrimFloat64, "number"},
		{"bool type", graph.PrimBool, "boolean"},
		{"unknown type", graph.PrimUnknown, "any"},
	}

	g := createTestGenerator(nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := &graph.TypeRef{
				Kind:      graph.KindPrimitive,
				Primitive: tt.prim,
			}
			result := g.typeRefToTS(ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTypeRefToTS_UnknownVsAny(t *testing.T) {
	ref := &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimUnknown}
	assert.Equal(t, "any", createTestGenerator(nil).typeRefToTS(ref))
	assert.Equal(t, "unknown", createTestGenerator(&Config{UnknownAny: true}).typeRefToTS(ref))
}

func TestTypeRefToTS_Complex(t *testing.T) {
	g := createTestGenerator(nil)

	t.Run("arrays", func(t *testing.T) {
		ref := &graph.TypeRef{Kind: graph.KindArray, ItemType: &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString}}
		assert.Equal(t, "string[]", g.typeRefToTS(ref))
	})

	t.Run("maps", func(t *testing.T) {
		ref := &graph.TypeRef{Kind: graph.KindMap, ValueType: &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimInt}}
		assert.Equal(t, "Record<string, number>", g.typeRefToTS(ref))
	})

	t.Run("named types", func(t *testing.T) {
		ref := &graph.TypeRef{Kind: graph.KindRef, TypeName: "User"}
		assert.Equal(t, "User", g.typeRefToTS(ref))
	})

	t.Run("union", func(t *testing.T) {
		ref := &graph.TypeRef{Kind: graph.KindUnion, UnionMembers: []*graph.TypeRef{{Kind: graph.KindPrimitive, Primitive: graph.PrimString}, {Kind: graph.KindPrimitive, Primitive: graph.PrimInt}}}
		result := g.typeRefToTS(ref)
		assert.Contains(t, result, "string")
		assert.Contains(t, result, "number")
	})

	t.Run("nested array of maps", func(t *testing.T) {
		ref := &graph.TypeRef{Kind: graph.KindArray, ItemType: &graph.TypeRef{Kind: graph.KindMap, ValueType: &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString}}}
		assert.Equal(t, "Record<string, string>[]", g.typeRefToTS(ref))
	})
}

func TestGenerateFile_Headers(t *testing.T) {
	typ := createTestType("User", graph.KindStruct)

	t.Run("with header", func(t *testing.T) {
		result, err := createTestGenerator(&Config{DisableHeaders: false}).GenerateFile([]*graph.Type{typ}, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "DO NOT EDIT")
	})

	t.Run("without header", func(t *testing.T) {
		result, err := createTestGenerator(&Config{DisableHeaders: true}).GenerateFile([]*graph.Type{typ}, nil)
		require.NoError(t, err)
		assert.NotContains(t, result, "DO NOT EDIT")
	})
}

func TestGenerateFile_TypeOrdering(t *testing.T) {
	g := createTestGenerator(nil)
	types := []*graph.Type{createTestType("Zebra", graph.KindStruct), createTestType("Apple", graph.KindStruct)}
	result, err := g.GenerateFile(types, nil)
	require.NoError(t, err)
	assert.Less(t, strings.Index(result, "Zebra"), strings.Index(result, "Apple"))
}

func TestGenerateInterface_QuotedPropertyNames(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Special", graph.KindStruct)
	typ.Fields = []*graph.Field{
		createTestField("ValidName", "validName", graph.PrimString, true),
		createTestField("KebabCase", "kebab-case", graph.PrimString, true),
		createTestField("WithSpace", "with space", graph.PrimString, true),
		createTestField("AtSign", "@special", graph.PrimString, true),
	}

	result, err := g.generateInterface(typ)
	require.NoError(t, err)
	assert.Contains(t, result, "validName: string;")
	assert.Contains(t, result, `"kebab-case": string;`)
	assert.Contains(t, result, `"with space": string;`)
	assert.Contains(t, result, `"@special": string;`)
}

// Phase 2.2: JSDoc Comments and Format Annotations

func TestGenerateInterface_FieldDescriptions(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", graph.KindStruct)
	typ.Fields = []*graph.Field{
		{
			JSONName:    "email",
			Description: "User email address",
			Type:        &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString},
			Required:    true,
		},
	}

	result, err := g.generateInterface(typ)
	require.NoError(t, err)
	assert.Contains(t, result, "/** User email address */")
}

func TestGenerateInterface_FormatAnnotations(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Event", graph.KindStruct)
	typ.Fields = []*graph.Field{
		{
			JSONName: "id",
			Type: &graph.TypeRef{
				Kind:      graph.KindPrimitive,
				Primitive: graph.PrimString,
				Format:    "uuid",
			},
			Required: true,
		},
		{
			JSONName:    "timestamp",
			Description: "Event timestamp",
			Type: &graph.TypeRef{
				Kind:      graph.KindPrimitive,
				Primitive: graph.PrimString,
				Format:    "date-time",
			},
			Required: true,
		},
	}

	result, err := g.generateInterface(typ)
	require.NoError(t, err)
	assert.Contains(t, result, "@format uuid")
	assert.Contains(t, result, "@format date-time")
	assert.Contains(t, result, "Event timestamp")
}

// Phase 2.3: Intersection Types (Composition)

func TestGenerateInterface_WithExtends(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", graph.KindStruct)
	typ.Extends = []string{"BaseModel"}
	typ.Fields = []*graph.Field{
		createTestField("Name", "name", graph.PrimString, true),
	}

	result, err := g.generateInterface(typ)
	require.NoError(t, err)
	assert.Contains(t, result, "export type User = BaseModel & {")
	assert.Contains(t, result, "name: string;")
}

func TestGenerateInterface_MultipleExtends(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", graph.KindStruct)
	typ.Extends = []string{"BaseModel", "Timestamped", "Auditable"}
	typ.Fields = []*graph.Field{
		createTestField("Name", "name", graph.PrimString, true),
	}

	result, err := g.generateInterface(typ)
	require.NoError(t, err)
	assert.Contains(t, result, "BaseModel & Timestamped & Auditable & {")
}

func TestGenerateFile_Imports(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", graph.KindStruct)

	t.Run("single import", func(t *testing.T) {
		imports := []graph.ImportSpec{{ImportPath: "./base", TypeNames: []string{"BaseModel", "Auditable"}}}
		result, err := g.GenerateFile([]*graph.Type{typ}, imports)
		require.NoError(t, err)
		assert.Contains(t, result, "import type { Auditable, BaseModel } from './base';")
	})

	t.Run("multiple imports", func(t *testing.T) {
		imports := []graph.ImportSpec{{ImportPath: "./base", TypeNames: []string{"BaseModel"}}, {ImportPath: "./timestamps", TypeNames: []string{"Timestamped"}}}
		result, err := g.GenerateFile([]*graph.Type{typ}, imports)
		require.NoError(t, err)
		assert.Contains(t, result, "BaseModel")
		assert.Contains(t, result, "Timestamped")
	})
}

func TestGenerateIndexSignature(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		g := createTestGenerator(&Config{AdditionalProperties: false})
		typ := createTestType("Config", graph.KindStruct)
		typ.AdditionalProps = &graph.AdditionalPropsConfig{Allowed: true, Type: &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString}}
		assert.Empty(t, g.generateIndexSignature(typ))
	})

	t.Run("untyped any", func(t *testing.T) {
		g := createTestGenerator(&Config{AdditionalProperties: true})
		typ := createTestType("Config", graph.KindStruct)
		typ.AdditionalProps = &graph.AdditionalPropsConfig{Allowed: true, Type: nil}
		assert.Contains(t, g.generateIndexSignature(typ), "[key: string]: any;")
	})

	t.Run("typed string", func(t *testing.T) {
		g := createTestGenerator(&Config{AdditionalProperties: true})
		typ := createTestType("Config", graph.KindStruct)
		typ.AdditionalProps = &graph.AdditionalPropsConfig{Allowed: true, Type: &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString}}
		assert.Contains(t, g.generateIndexSignature(typ), "[key: string]: string;")
	})
}

func TestGenerateFile_ImportWriteDoesNotMutateInput(t *testing.T) {
	g := createTestGenerator(&Config{DisableHeaders: true})

	imports := []graph.ImportSpec{
		{ImportPath: "./models", TypeNames: []string{"Zebra", "Apple", "Mango"}},
	}

	original := make([]string, len(imports[0].TypeNames))
	copy(original, imports[0].TypeNames)

	_, err := g.GenerateFile([]*graph.Type{
		createTestType("Test", graph.KindStruct),
	}, imports)
	require.NoError(t, err)

	assert.Equal(t, original, imports[0].TypeNames,
		"GenerateFile must not mutate the caller's TypeNames slice")
}

func TestEdgeCases(t *testing.T) {
	g := createTestGenerator(nil)

	t.Run("empty type list", func(t *testing.T) {
		result, err := g.GenerateFile([]*graph.Type{}, nil)
		require.NoError(t, err)
		assert.NotContains(t, result, "export")
	})

	t.Run("empty enum values", func(t *testing.T) {
		typ := createTestType("Status", graph.KindEnum)
		typ.EnumType = graph.EnumKindString
		result, err := g.generateEnum(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "export type Status =")
	})

	t.Run("quoted description", func(t *testing.T) {
		typ := createTestType("User", graph.KindStruct)
		typ.Fields = []*graph.Field{{JSONName: "name", Description: `This is a "quoted" description`, Type: &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString}, Required: true}}
		result, err := g.generateInterface(typ)
		require.NoError(t, err)
		assert.Contains(t, result, `"quoted"`)
	})

	t.Run("unicode in names", func(t *testing.T) {
		typ := createTestType("Café", graph.KindStruct)
		typ.Fields = []*graph.Field{createTestField("Naïve", "naïve", graph.PrimString, true)}
		result, err := g.generateInterface(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "export interface Café")
	})

	t.Run("inline enum", func(t *testing.T) {
		ref := &graph.TypeRef{Kind: graph.KindEnum, EnumValues: []graph.EnumValue{{Value: "option1"}, {Value: 42}, {Value: nil}}}
		result := g.typeRefToTS(ref)
		assert.Contains(t, result, `"option1"`)
		assert.Contains(t, result, "42")
		assert.Contains(t, result, "null")
	})
}

func TestGenerateFile_MixedTypes(t *testing.T) {
	g := createTestGenerator(nil)

	interfaceType := createTestType("User", graph.KindStruct)
	interfaceType.Fields = []*graph.Field{
		createTestField("Name", "name", graph.PrimString, true),
	}

	enumType := createTestType("Status", graph.KindEnum)
	enumType.EnumType = graph.EnumKindString
	enumType.EnumValues = []graph.EnumValue{
		{Name: "Active", Value: "active"},
	}

	result, err := g.GenerateFile([]*graph.Type{interfaceType, enumType}, nil)
	require.NoError(t, err)

	assert.Contains(t, result, "export type Status =")
	assert.Contains(t, result, "export interface User")

	userPos := strings.Index(result, "export interface User")
	statusPos := strings.Index(result, "export type Status")
	assert.Less(t, userPos, statusPos, "User should come before Status (input order)")
}

func TestGenerateFile_AllConfigOptions(t *testing.T) {
	g := createTestGenerator(&Config{
		DisableHeaders:       true,
		DisableTimestamp:     true,
		UnknownAny:           true,
		AdditionalProperties: false,
	})

	typ := createTestType("User", graph.KindStruct)
	typ.Description = "This comment should appear"
	typ.Fields = []*graph.Field{
		{
			JSONName: "data",
			Type:     &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimUnknown},
			Required: true,
		},
	}

	result, err := g.GenerateFile([]*graph.Type{typ}, nil)
	require.NoError(t, err)

	assert.NotContains(t, result, "DO NOT EDIT")
	assert.Contains(t, result, "data: unknown;")
	assert.Contains(t, result, "This comment should appear")
}

func TestGenerateFile_ZodBothModeImportsSchemas(t *testing.T) {
	g := createTestGenerator(&Config{
		ZodMode: ZodModeWithInterface,
	})

	typ := createTestType("Event", graph.KindStruct)
	typ.Fields = []*graph.Field{
		{
			JSONName: "header",
			Type:     &graph.TypeRef{Kind: graph.KindRef, TypeName: "EventHeader"},
			Required: true,
		},
	}

	imports := []graph.ImportSpec{
		{ImportPath: "./header", TypeNames: []string{"EventHeader"}},
	}

	result, err := g.GenerateFile([]*graph.Type{typ}, imports)
	require.NoError(t, err)

	assert.Contains(t, result, "import type { EventHeader } from './header';")
	assert.Contains(t, result, "import { EventHeaderSchema } from './header';")

	assert.Contains(t, result, "EventHeaderSchema")
}
