package ts

import (
	"strings"
	"testing"

	"github.com/mirpo/schemagen/pkg/lang/tscommon"
	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper functions

func createTestGenerator(config *Config) *Generator {
	if config == nil {
		return NewGenerator()
	}
	return NewGeneratorWithConfig(config)
}

func createTestType(name string, kind typegraph.TypeKind) *typegraph.Type {
	return &typegraph.Type{
		Name:   name,
		Kind:   kind,
		Fields: []*typegraph.Field{},
	}
}

func createTestField(name string, jsonName string, prim typegraph.PrimitiveKind, required bool) *typegraph.Field {
	return &typegraph.Field{
		JSONName: jsonName,
		Type: &typegraph.TypeRef{
			Kind:      typegraph.KindPrimitive,
			Primitive: prim,
		},
		Required: required,
	}
}

func TestGenerateInterface(t *testing.T) {
	g := createTestGenerator(nil)

	t.Run("simple", func(t *testing.T) {
		typ := createTestType("User", typegraph.KindStruct)
		typ.Fields = []*typegraph.Field{createTestField("ID", "id", typegraph.PrimString, true), createTestField("Name", "name", typegraph.PrimString, true)}
		result, err := g.generateInterface(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "export interface User {")
		assert.Contains(t, result, "id: string;")
	})

	t.Run("with description", func(t *testing.T) {
		typ := createTestType("User", typegraph.KindStruct)
		typ.Description = "Represents a user"
		typ.Fields = []*typegraph.Field{createTestField("ID", "id", typegraph.PrimString, true)}
		result, err := g.generateInterface(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "* Represents a user")
	})

	t.Run("empty fields", func(t *testing.T) {
		typ := createTestType("Empty", typegraph.KindStruct)
		result, err := g.generateInterface(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "export interface Empty {")
	})

	t.Run("multiple fields", func(t *testing.T) {
		typ := createTestType("Product", typegraph.KindStruct)
		typ.Fields = []*typegraph.Field{
			createTestField("ID", "id", typegraph.PrimString, true),
			createTestField("Price", "price", typegraph.PrimFloat64, true),
			createTestField("Available", "available", typegraph.PrimBool, true),
		}
		result, err := g.generateInterface(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "price: number;")
		assert.Contains(t, result, "available: boolean;")
	})

	t.Run("optional fields", func(t *testing.T) {
		typ := createTestType("User", typegraph.KindStruct)
		typ.Fields = []*typegraph.Field{createTestField("ID", "id", typegraph.PrimString, true), createTestField("Email", "email", typegraph.PrimString, false)}
		result, err := g.generateInterface(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "email?: string;")
	})
}

func TestGenerateEnum(t *testing.T) {
	g := createTestGenerator(nil)

	t.Run("string union", func(t *testing.T) {
		typ := createTestType("Status", typegraph.KindEnum)
		typ.EnumType = "string"
		typ.EnumValues = []typegraph.EnumValue{{Name: "Active", Value: "active"}, {Name: "Inactive", Value: "inactive"}}
		result, err := g.generateEnum(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "export type Status =")
		assert.Contains(t, result, `"active"`)
	})

	t.Run("numeric enum", func(t *testing.T) {
		typ := createTestType("Priority", typegraph.KindEnum)
		typ.EnumType = "int"
		typ.EnumValues = []typegraph.EnumValue{{Name: "Low", Value: 1}, {Name: "High", Value: 2}}
		result, err := g.generateEnum(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "export enum Priority {")
		assert.Contains(t, result, "Low = 1")
	})

	t.Run("with description", func(t *testing.T) {
		typ := createTestType("Status", typegraph.KindEnum)
		typ.Description = "User status values"
		typ.EnumType = "string"
		typ.EnumValues = []typegraph.EnumValue{{Name: "Active", Value: "active"}}
		result, err := g.generateEnum(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "* User status values")
	})

	t.Run("mixed types", func(t *testing.T) {
		typ := createTestType("Mixed", typegraph.KindEnum)
		typ.EnumType = "string"
		typ.EnumValues = []typegraph.EnumValue{{Name: "String", Value: "text"}, {Name: "Number", Value: 42}, {Name: "Null", Value: nil}}
		result, err := g.generateEnum(typ)
		require.NoError(t, err)
		assert.Contains(t, result, `"text"`)
		assert.Contains(t, result, "42")
		assert.Contains(t, result, "null")
	})
}

func TestGenerateUnionAlias_WithMembers(t *testing.T) {
	g := createTestGenerator(nil)

	typ := &typegraph.Type{
		Name: "Shape",
		Kind: typegraph.KindUnion,
		UnionMembers: []*typegraph.TypeRef{
			{Kind: typegraph.KindRef, TypeName: "Circle"},
			{Kind: typegraph.KindRef, TypeName: "Square"},
			{Kind: typegraph.KindRef, TypeName: "Triangle"},
		},
	}

	result, err := g.generateUnionAlias(typ)
	require.NoError(t, err)

	assert.Contains(t, result, "export type Shape = Circle | Square | Triangle;")
	assert.NotContains(t, result, "any")
}

func TestGenerateUnionAlias_EmptyMembers(t *testing.T) {
	g := createTestGenerator(nil)

	typ := &typegraph.Type{
		Name: "Unknown",
		Kind: typegraph.KindUnion,
	}

	result, err := g.generateUnionAlias(typ)
	require.NoError(t, err)

	assert.Contains(t, result, "export type Unknown = any;")
}

func TestGenerateUnionAlias_WithDescription(t *testing.T) {
	g := createTestGenerator(nil)

	typ := &typegraph.Type{
		Name:        "Shape",
		Kind:        typegraph.KindUnion,
		Description: "A geometric shape",
		UnionMembers: []*typegraph.TypeRef{
			{Kind: typegraph.KindRef, TypeName: "Circle"},
			{Kind: typegraph.KindRef, TypeName: "Square"},
		},
	}

	result, err := g.generateUnionAlias(typ)
	require.NoError(t, err)

	assert.Contains(t, result, "A geometric shape")
	assert.Contains(t, result, "Circle | Square")
}

// Phase 1.3: Type Conversion

func TestTypeRefToTS_Primitives(t *testing.T) {
	tests := []struct {
		name     string
		prim     typegraph.PrimitiveKind
		expected string
	}{
		{"string type", typegraph.PrimString, "string"},
		{"int type", typegraph.PrimInt, "number"},
		{"float64 type", typegraph.PrimFloat64, "number"},
		{"bool type", typegraph.PrimBool, "boolean"},
		{"unknown type", typegraph.PrimUnknown, "any"},
	}

	g := createTestGenerator(nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := &typegraph.TypeRef{
				Kind:      typegraph.KindPrimitive,
				Primitive: tt.prim,
			}
			result := g.typeRefToTS(ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTypeRefToTS_UnknownVsAny(t *testing.T) {
	ref := &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimUnknown}
	assert.Equal(t, "any", createTestGenerator(nil).typeRefToTS(ref))
	assert.Equal(t, "unknown", createTestGenerator(&Config{UnknownAny: true}).typeRefToTS(ref))
}

func TestTypeRefToTS_Complex(t *testing.T) {
	g := createTestGenerator(nil)

	t.Run("arrays", func(t *testing.T) {
		ref := &typegraph.TypeRef{Kind: typegraph.KindArray, ItemType: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimString}}
		assert.Equal(t, "string[]", g.typeRefToTS(ref))
	})

	t.Run("maps", func(t *testing.T) {
		ref := &typegraph.TypeRef{Kind: typegraph.KindMap, ValueType: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimInt}}
		assert.Equal(t, "Record<string, number>", g.typeRefToTS(ref))
	})

	t.Run("named types", func(t *testing.T) {
		ref := &typegraph.TypeRef{Kind: typegraph.KindRef, TypeName: "User"}
		assert.Equal(t, "User", g.typeRefToTS(ref))
	})

	t.Run("union", func(t *testing.T) {
		ref := &typegraph.TypeRef{Kind: typegraph.KindUnion, UnionMembers: []*typegraph.TypeRef{{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimString}, {Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimInt}}}
		result := g.typeRefToTS(ref)
		assert.Contains(t, result, "string")
		assert.Contains(t, result, "number")
	})

	t.Run("nested array of maps", func(t *testing.T) {
		ref := &typegraph.TypeRef{Kind: typegraph.KindArray, ItemType: &typegraph.TypeRef{Kind: typegraph.KindMap, ValueType: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimString}}}
		assert.Equal(t, "Record<string, string>[]", g.typeRefToTS(ref))
	})
}

func TestGenerateFile_Headers(t *testing.T) {
	typ := createTestType("User", typegraph.KindStruct)

	t.Run("with header", func(t *testing.T) {
		result, err := createTestGenerator(&Config{DisableHeaders: false}).GenerateFile([]*typegraph.Type{typ}, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "DO NOT EDIT")
	})

	t.Run("without header", func(t *testing.T) {
		result, err := createTestGenerator(&Config{DisableHeaders: true}).GenerateFile([]*typegraph.Type{typ}, nil)
		require.NoError(t, err)
		assert.NotContains(t, result, "DO NOT EDIT")
	})
}

func TestGenerateFile_MultipleTypes(t *testing.T) {
	g := createTestGenerator(nil)
	types := []*typegraph.Type{createTestType("User", typegraph.KindStruct), createTestType("Product", typegraph.KindStruct)}
	result, err := g.GenerateFile(types, nil)
	require.NoError(t, err)
	assert.Contains(t, result, "export interface User")
	assert.Contains(t, result, "export interface Product")
}

func TestGenerateFile_TypeOrdering(t *testing.T) {
	g := createTestGenerator(nil)
	types := []*typegraph.Type{createTestType("Zebra", typegraph.KindStruct), createTestType("Apple", typegraph.KindStruct)}
	result, err := g.GenerateFile(types, nil)
	require.NoError(t, err)
	assert.Less(t, strings.Index(result, "Zebra"), strings.Index(result, "Apple"))
}

// Phase 2.1: Property Name Quoting

func TestNeedsQuoting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid identifier", "validName", false},
		{"starts with number", "123abc", true},
		{"contains hyphen", "kebab-case", true},
		{"contains @", "@special", true},
		{"contains space", "with spaces", true},
		{"contains dot", "with.dot", true},
		{"underscore valid", "_underscore", false},
		{"dollar valid", "$dollar", false},
		{"empty string", "", true},
		{"Unicode valid", "café", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tscommon.NeedsQuoting(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateInterface_QuotedPropertyNames(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Special", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("ValidName", "validName", typegraph.PrimString, true),
		createTestField("KebabCase", "kebab-case", typegraph.PrimString, true),
		createTestField("WithSpace", "with space", typegraph.PrimString, true),
		createTestField("AtSign", "@special", typegraph.PrimString, true),
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
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			JSONName:    "email",
			Description: "User email address",
			Type:        &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimString},
			Required:    true,
		},
	}

	result, err := g.generateInterface(typ)
	require.NoError(t, err)
	assert.Contains(t, result, "/** User email address */")
}

func TestGenerateInterface_FormatAnnotations(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Event", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			JSONName: "id",
			Type: &typegraph.TypeRef{
				Kind:      typegraph.KindPrimitive,
				Primitive: typegraph.PrimString,
				Format:    "uuid",
			},
			Required: true,
		},
		{
			JSONName:    "timestamp",
			Description: "Event timestamp",
			Type: &typegraph.TypeRef{
				Kind:      typegraph.KindPrimitive,
				Primitive: typegraph.PrimString,
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
	typ := createTestType("User", typegraph.KindStruct)
	typ.Extends = []string{"BaseModel"}
	typ.Fields = []*typegraph.Field{
		createTestField("Name", "name", typegraph.PrimString, true),
	}

	result, err := g.generateInterface(typ)
	require.NoError(t, err)
	assert.Contains(t, result, "export type User = BaseModel & {")
	assert.Contains(t, result, "name: string;")
}

func TestGenerateInterface_MultipleExtends(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Extends = []string{"BaseModel", "Timestamped", "Auditable"}
	typ.Fields = []*typegraph.Field{
		createTestField("Name", "name", typegraph.PrimString, true),
	}

	result, err := g.generateInterface(typ)
	require.NoError(t, err)
	assert.Contains(t, result, "BaseModel & Timestamped & Auditable & {")
}

func TestGenerateFile_Imports(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)

	t.Run("single import", func(t *testing.T) {
		imports := []typegraph.ImportSpec{{ImportPath: "./base", TypeNames: []string{"BaseModel", "Auditable"}}}
		result, err := g.GenerateFile([]*typegraph.Type{typ}, imports)
		require.NoError(t, err)
		assert.Contains(t, result, "import type { Auditable, BaseModel } from './base';")
	})

	t.Run("multiple imports", func(t *testing.T) {
		imports := []typegraph.ImportSpec{{ImportPath: "./base", TypeNames: []string{"BaseModel"}}, {ImportPath: "./timestamps", TypeNames: []string{"Timestamped"}}}
		result, err := g.GenerateFile([]*typegraph.Type{typ}, imports)
		require.NoError(t, err)
		assert.Contains(t, result, "BaseModel")
		assert.Contains(t, result, "Timestamped")
	})

	t.Run("sorted type names", func(t *testing.T) {
		imports := []typegraph.ImportSpec{{ImportPath: "./base", TypeNames: []string{"Zebra", "Apple", "Middle"}}}
		result, err := g.GenerateFile([]*typegraph.Type{typ}, imports)
		require.NoError(t, err)
		assert.Contains(t, result, "import type { Apple, Middle, Zebra } from './base';")
	})
}

func TestGenerateIndexSignature(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		g := createTestGenerator(&Config{AdditionalProperties: false})
		typ := createTestType("Config", typegraph.KindStruct)
		typ.AdditionalProps = &typegraph.AdditionalPropsConfig{Allowed: true, Type: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimString}}
		assert.Empty(t, g.generateIndexSignature(typ))
	})

	t.Run("untyped any", func(t *testing.T) {
		g := createTestGenerator(&Config{AdditionalProperties: true})
		typ := createTestType("Config", typegraph.KindStruct)
		typ.AdditionalProps = &typegraph.AdditionalPropsConfig{Allowed: true, Type: nil}
		assert.Contains(t, g.generateIndexSignature(typ), "[key: string]: any;")
	})

	t.Run("typed string", func(t *testing.T) {
		g := createTestGenerator(&Config{AdditionalProperties: true})
		typ := createTestType("Config", typegraph.KindStruct)
		typ.AdditionalProps = &typegraph.AdditionalPropsConfig{Allowed: true, Type: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimString}}
		assert.Contains(t, g.generateIndexSignature(typ), "[key: string]: string;")
	})

	t.Run("in interface", func(t *testing.T) {
		g := createTestGenerator(&Config{AdditionalProperties: true})
		typ := createTestType("Config", typegraph.KindStruct)
		typ.Fields = []*typegraph.Field{createTestField("Name", "name", typegraph.PrimString, true)}
		typ.AdditionalProps = &typegraph.AdditionalPropsConfig{Allowed: true, Type: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimInt}}
		result, err := g.generateInterface(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "[key: string]: number;")
	})
}

func TestEdgeCases(t *testing.T) {
	g := createTestGenerator(nil)

	t.Run("empty type list", func(t *testing.T) {
		result, err := g.GenerateFile([]*typegraph.Type{}, nil)
		require.NoError(t, err)
		assert.NotContains(t, result, "export")
	})

	t.Run("empty enum values", func(t *testing.T) {
		typ := createTestType("Status", typegraph.KindEnum)
		typ.EnumType = "string"
		result, err := g.generateEnum(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "export type Status =")
	})

	t.Run("quoted description", func(t *testing.T) {
		typ := createTestType("User", typegraph.KindStruct)
		typ.Fields = []*typegraph.Field{{JSONName: "name", Description: `This is a "quoted" description`, Type: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimString}, Required: true}}
		result, err := g.generateInterface(typ)
		require.NoError(t, err)
		assert.Contains(t, result, `"quoted"`)
	})

	t.Run("unicode in names", func(t *testing.T) {
		typ := createTestType("Café", typegraph.KindStruct)
		typ.Fields = []*typegraph.Field{createTestField("Naïve", "naïve", typegraph.PrimString, true)}
		result, err := g.generateInterface(typ)
		require.NoError(t, err)
		assert.Contains(t, result, "export interface Café")
	})

	t.Run("inline enum", func(t *testing.T) {
		ref := &typegraph.TypeRef{Kind: typegraph.KindEnum, EnumValues: []interface{}{"option1", 42, nil}}
		result := g.typeRefToTS(ref)
		assert.Contains(t, result, `"option1"`)
		assert.Contains(t, result, "42")
		assert.Contains(t, result, "null")
	})
}

// Phase 3.2: Integration Tests

func TestGenerateFile_CompleteInterface(t *testing.T) {
	g := createTestGenerator(&Config{
		DisableHeaders: false,
		UnknownAny:     false,
	})

	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = "User represents a system user"
	// No extends - use regular interface to test format annotations
	typ.Fields = []*typegraph.Field{
		{
			JSONName: "id",
			Type: &typegraph.TypeRef{
				Kind:      typegraph.KindPrimitive,
				Primitive: typegraph.PrimString,
				Format:    "uuid",
			},
			Required: true,
		},
		{
			JSONName:    "email",
			Description: "User email address",
			Type: &typegraph.TypeRef{
				Kind:      typegraph.KindPrimitive,
				Primitive: typegraph.PrimString,
				Format:    "email",
			},
			Required: true,
		},
		{
			JSONName: "age",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimInt},
			Required: false,
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	require.NoError(t, err)

	// Check header
	assert.Contains(t, result, "DO NOT EDIT")
	// Check type comment
	assert.Contains(t, result, "* User represents a system user")
	// Check regular interface (no extends)
	assert.Contains(t, result, "export interface User {")
	// Check fields
	assert.Contains(t, result, "id: string;")
	assert.Contains(t, result, "email: string;")
	assert.Contains(t, result, "age?: number;")
	// Check format annotations (only in regular interfaces, not intersection types)
	assert.Contains(t, result, "@format uuid")
	assert.Contains(t, result, "@format email")
	assert.Contains(t, result, "User email address")
}

func TestGenerateFile_CompleteEnum(t *testing.T) {
	g := createTestGenerator(&Config{
		DisableHeaders: false,
	})

	typ := createTestType("Status", typegraph.KindEnum)
	typ.Description = "User status values"
	typ.EnumType = "string"
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "Active", Value: "active"},
		{Name: "Inactive", Value: "inactive"},
		{Name: "Pending", Value: "pending"},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	require.NoError(t, err)

	// Check header
	assert.Contains(t, result, "DO NOT EDIT")
	// Check comment
	assert.Contains(t, result, "* User status values")
	// Check union type
	assert.Contains(t, result, "export type Status =")
	assert.Contains(t, result, `"active" | "inactive" | "pending"`)
}

func TestGenerateFile_MixedTypes(t *testing.T) {
	g := createTestGenerator(nil)

	interfaceType := createTestType("User", typegraph.KindStruct)
	interfaceType.Fields = []*typegraph.Field{
		createTestField("Name", "name", typegraph.PrimString, true),
	}

	enumType := createTestType("Status", typegraph.KindEnum)
	enumType.EnumType = "string"
	enumType.EnumValues = []typegraph.EnumValue{
		{Name: "Active", Value: "active"},
	}

	result, err := g.GenerateFile([]*typegraph.Type{interfaceType, enumType}, nil)
	require.NoError(t, err)

	// Both types present
	assert.Contains(t, result, "export type Status =")
	assert.Contains(t, result, "export interface User")

	// User comes before Status (input order preserved)
	userPos := strings.Index(result, "export interface User")
	statusPos := strings.Index(result, "export type Status")
	assert.Less(t, userPos, statusPos, "User should come before Status (input order)")
}

func TestGenerateFile_WithImportsAndAdditionalProps(t *testing.T) {
	g := createTestGenerator(&Config{
		AdditionalProperties: true,
	})

	typ := createTestType("Config", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("Name", "name", typegraph.PrimString, true),
	}
	typ.AdditionalProps = &typegraph.AdditionalPropsConfig{
		Allowed: true,
		Type:    &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimString},
	}

	imports := []typegraph.ImportSpec{
		{
			ImportPath: "./base",
			TypeNames:  []string{"BaseConfig"},
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, imports)
	require.NoError(t, err)

	// Check import
	assert.Contains(t, result, "import type { BaseConfig } from './base';")
	// Check interface
	assert.Contains(t, result, "export interface Config")
	assert.Contains(t, result, "name: string;")
	// Check index signature
	assert.Contains(t, result, "[key: string]: string;")
}

func TestGenerateFile_AllConfigOptions(t *testing.T) {
	g := createTestGenerator(&Config{
		DisableHeaders:       true,
		DisableTimestamp:     true,
		UnknownAny:           true,
		AdditionalProperties: false,
	})

	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = "This comment should appear"
	typ.Fields = []*typegraph.Field{
		{
			JSONName: "data",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimUnknown},
			Required: true,
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	require.NoError(t, err)

	// Check no headers
	assert.NotContains(t, result, "DO NOT EDIT")
	// Check unknown instead of any
	assert.Contains(t, result, "data: unknown;")
	// Check comments still appear
	assert.Contains(t, result, "This comment should appear")
}

func TestGenerateFile_ZodBothModeImportsSchemas(t *testing.T) {
	g := createTestGenerator(&Config{
		ZodMode: ZodModeWithInterface,
	})

	typ := createTestType("Event", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			JSONName: "header",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindRef, TypeName: "EventHeader"},
			Required: true,
		},
	}

	imports := []typegraph.ImportSpec{
		{ImportPath: "./header", TypeNames: []string{"EventHeader"}},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, imports)
	require.NoError(t, err)

	// Should import both type and schema for cross-file references
	assert.Contains(t, result, "import type { EventHeader } from './header';")
	assert.Contains(t, result, "import { EventHeaderSchema } from './header';")

	// Should use the schema in the Zod definition
	assert.Contains(t, result, "EventHeaderSchema")
}
