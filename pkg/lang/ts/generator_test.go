package ts

import (
	"strings"
	"testing"

	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/stretchr/testify/assert"
)

// Helper functions

func createTestGenerator(config *Config) *Generator {
	graph := &typegraph.Graph{
		Types: []*typegraph.Type{},
	}
	if config == nil {
		return NewGenerator(graph)
	}
	return NewGeneratorWithConfig(graph, config)
}

func createTestType(name string, kind typegraph.TypeKind) *typegraph.Type {
	return &typegraph.Type{
		Name:   name,
		Kind:   kind,
		Fields: []*typegraph.Field{},
	}
}

func createTestField(name string, jsonName string, tsType string, required bool) *typegraph.Field {
	return &typegraph.Field{
		Name:     name,
		JSONName: jsonName,
		Type: &typegraph.TypeRef{
			Kind:   typegraph.KindPrimitive,
			GoType: tsType,
		},
		Required: required,
	}
}

// Phase 1.1: Basic Interface Generation

func TestGenerateInterface_Simple(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("ID", "id", "string", true),
		createTestField("Name", "name", "string", true),
	}

	result, err := g.generateInterface(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "export interface User {")
	assert.Contains(t, result, "id: string;")
	assert.Contains(t, result, "name: string;")
}

func TestGenerateInterface_WithDescription(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = "Represents a user in the system"
	typ.Fields = []*typegraph.Field{
		createTestField("ID", "id", "string", true),
	}

	result, err := g.generateInterface(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "/**")
	assert.Contains(t, result, "* Represents a user in the system")
	assert.Contains(t, result, "*/")
	assert.Contains(t, result, "export interface User {")
}

func TestGenerateInterface_EmptyFields(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Empty", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{}

	result, err := g.generateInterface(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "export interface Empty {")
	assert.Contains(t, result, "}")
}

func TestGenerateInterface_MultipleFields(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Product", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("ID", "id", "string", true),
		createTestField("Name", "name", "string", true),
		createTestField("Price", "price", "float64", true),
		createTestField("Available", "available", "bool", true),
	}

	result, err := g.generateInterface(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "id: string;")
	assert.Contains(t, result, "name: string;")
	assert.Contains(t, result, "price: number;")
	assert.Contains(t, result, "available: boolean;")
}

func TestGenerateInterface_OptionalFields(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("ID", "id", "string", true),
		createTestField("Email", "email", "string", false), // optional
	}

	result, err := g.generateInterface(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "id: string;")
	assert.Contains(t, result, "email?: string;") // with ?
}

// Phase 1.2: Basic Enum Generation

func TestGenerateEnum_StringUnion(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Status", typegraph.KindEnum)
	typ.EnumType = "string"
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "Active", Value: "active"},
		{Name: "Inactive", Value: "inactive"},
	}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "export type Status =")
	assert.Contains(t, result, `"active"`)
	assert.Contains(t, result, `"inactive"`)
	assert.Contains(t, result, "|")
}

func TestGenerateEnum_NumericEnum(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Priority", typegraph.KindEnum)
	typ.EnumType = "int"
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "Low", Value: 1},
		{Name: "High", Value: 2},
	}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "export enum Priority {")
	assert.Contains(t, result, "Low = 1")
	assert.Contains(t, result, "High = 2")
}

func TestGenerateEnum_WithDescription(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Status", typegraph.KindEnum)
	typ.Description = "User status values"
	typ.EnumType = "string"
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "Active", Value: "active"},
	}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "/**")
	assert.Contains(t, result, "* User status values")
	assert.Contains(t, result, "*/")
}

func TestGenerateEnum_MixedTypes(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Mixed", typegraph.KindEnum)
	typ.EnumType = "string"
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "String", Value: "text"},
		{Name: "Number", Value: 42},
		{Name: "Bool", Value: true},
		{Name: "Null", Value: nil},
	}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "export type Mixed =")
	assert.Contains(t, result, `"text"`)
	assert.Contains(t, result, "42")
	assert.Contains(t, result, "true")
	assert.Contains(t, result, "null")
}

// Phase 1.3: Type Conversion

func TestTypeRefToTS_Primitives(t *testing.T) {
	tests := []struct {
		name     string
		goType   string
		expected string
	}{
		{"string type", "string", "string"},
		{"int type", "int", "number"},
		{"float64 type", "float64", "number"},
		{"bool type", "bool", "boolean"},
		{"interface type", "interface{}", "any"},
	}

	g := createTestGenerator(nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := &typegraph.TypeRef{
				Kind:   typegraph.KindPrimitive,
				GoType: tt.goType,
			}
			result := g.typeRefToTS(ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTypeRefToTS_UnknownVsAny(t *testing.T) {
	ref := &typegraph.TypeRef{
		Kind:   typegraph.KindPrimitive,
		GoType: "interface{}",
	}

	// Default: any
	g1 := createTestGenerator(nil)
	assert.Equal(t, "any", g1.typeRefToTS(ref))

	// With UnknownAny: unknown
	g2 := createTestGenerator(&Config{UnknownAny: true})
	assert.Equal(t, "unknown", g2.typeRefToTS(ref))
}

func TestTypeRefToTS_Arrays(t *testing.T) {
	g := createTestGenerator(nil)
	ref := &typegraph.TypeRef{
		Kind: typegraph.KindArray,
		ItemType: &typegraph.TypeRef{
			Kind:   typegraph.KindPrimitive,
			GoType: "string",
		},
	}

	result := g.typeRefToTS(ref)
	assert.Equal(t, "string[]", result)
}

func TestTypeRefToTS_Maps(t *testing.T) {
	g := createTestGenerator(nil)
	ref := &typegraph.TypeRef{
		Kind: typegraph.KindMap,
		ValueType: &typegraph.TypeRef{
			Kind:   typegraph.KindPrimitive,
			GoType: "int",
		},
	}

	result := g.typeRefToTS(ref)
	assert.Equal(t, "Record<string, number>", result)
}

func TestTypeRefToTS_NamedTypes(t *testing.T) {
	g := createTestGenerator(nil)
	ref := &typegraph.TypeRef{
		Kind:     typegraph.KindRef, // Named type references use KindRef
		TypeName: "User",
	}

	result := g.typeRefToTS(ref)
	assert.Equal(t, "User", result)
}

func TestTypeRefToTS_Union(t *testing.T) {
	g := createTestGenerator(nil)
	ref := &typegraph.TypeRef{
		Kind: typegraph.KindUnion,
		UnionMembers: []*typegraph.TypeRef{
			{Kind: typegraph.KindPrimitive, GoType: "string"},
			{Kind: typegraph.KindPrimitive, GoType: "int"},
		},
	}

	result := g.typeRefToTS(ref)
	assert.Contains(t, result, "string")
	assert.Contains(t, result, "number")
	assert.Contains(t, result, "|")
}

func TestTypeRefToTS_Nested(t *testing.T) {
	g := createTestGenerator(nil)
	ref := &typegraph.TypeRef{
		Kind: typegraph.KindArray,
		ItemType: &typegraph.TypeRef{
			Kind: typegraph.KindMap,
			ValueType: &typegraph.TypeRef{
				Kind:   typegraph.KindPrimitive,
				GoType: "string",
			},
		},
	}

	result := g.typeRefToTS(ref)
	assert.Equal(t, "Record<string, string>[]", result)
}

// Phase 1.4: File Generation Basics

func TestGenerateFile_WithHeader(t *testing.T) {
	g := createTestGenerator(&Config{
		DisableHeaders: false,
	})
	typ := createTestType("User", typegraph.KindStruct)

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "DO NOT EDIT")
}

func TestGenerateFile_WithoutHeader(t *testing.T) {
	g := createTestGenerator(&Config{
		DisableHeaders: true,
	})
	typ := createTestType("User", typegraph.KindStruct)

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.NotContains(t, result, "DO NOT EDIT")
}

func TestGenerateFile_MultipleTypes(t *testing.T) {
	g := createTestGenerator(nil)
	types := []*typegraph.Type{
		createTestType("User", typegraph.KindStruct),
		createTestType("Product", typegraph.KindStruct),
		createTestType("Order", typegraph.KindStruct),
	}

	result, err := g.GenerateFile(types, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "export interface User")
	assert.Contains(t, result, "export interface Product")
	assert.Contains(t, result, "export interface Order")
}

func TestGenerateFile_TypeOrdering(t *testing.T) {
	g := createTestGenerator(nil)
	types := []*typegraph.Type{
		createTestType("Zebra", typegraph.KindStruct),
		createTestType("Apple", typegraph.KindStruct),
		createTestType("Middle", typegraph.KindStruct),
	}

	result, err := g.GenerateFile(types, nil)
	assert.NoError(t, err)

	// Find positions
	applePos := strings.Index(result, "export interface Apple")
	middlePos := strings.Index(result, "export interface Middle")
	zebraPos := strings.Index(result, "export interface Zebra")

	// Verify alphabetical ordering
	assert.True(t, applePos < middlePos, "Apple should come before Middle")
	assert.True(t, middlePos < zebraPos, "Middle should come before Zebra")
}

func TestGenerateFile_BlankLinesBetweenTypes(t *testing.T) {
	g := createTestGenerator(nil)
	types := []*typegraph.Type{
		createTestType("A", typegraph.KindStruct),
		createTestType("B", typegraph.KindStruct),
	}

	result, err := g.GenerateFile(types, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "}\n\nexport interface B")
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
			result := needsQuoting(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateInterface_QuotedPropertyNames(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Special", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("ValidName", "validName", "string", true),
		createTestField("KebabCase", "kebab-case", "string", true),
		createTestField("WithSpace", "with space", "string", true),
		createTestField("AtSign", "@special", "string", true),
	}

	result, err := g.generateInterface(typ)
	assert.NoError(t, err)
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
			Name:        "Email",
			JSONName:    "email",
			Description: "User email address",
			Type:        &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
			Required:    true,
		},
	}

	result, err := g.generateInterface(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "/** User email address */")
}

func TestGenerateInterface_FormatAnnotations(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Event", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:     "ID",
			JSONName: "id",
			Type: &typegraph.TypeRef{
				Kind:   typegraph.KindPrimitive,
				GoType: "string",
				Format: "uuid",
			},
			Required: true,
		},
		{
			Name:        "Timestamp",
			JSONName:    "timestamp",
			Description: "Event timestamp",
			Type: &typegraph.TypeRef{
				Kind:   typegraph.KindPrimitive,
				GoType: "string",
				Format: "date-time",
			},
			Required: true,
		},
	}

	result, err := g.generateInterface(typ)
	assert.NoError(t, err)
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
		createTestField("Name", "name", "string", true),
	}

	result, err := g.generateInterface(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "export type User = BaseModel & {")
	assert.Contains(t, result, "name: string;")
}

func TestGenerateInterface_MultipleExtends(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Extends = []string{"BaseModel", "Timestamped", "Auditable"}
	typ.Fields = []*typegraph.Field{
		createTestField("Name", "name", "string", true),
	}

	result, err := g.generateInterface(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "BaseModel & Timestamped & Auditable & {")
}

// Phase 2.4: Import Generation

func TestGenerateFile_WithImports(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)

	imports := []typegraph.ImportSpec{
		{
			ImportPath: "./base",
			TypeNames:  []string{"BaseModel", "Auditable"},
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, imports)
	assert.NoError(t, err)
	// TypeNames are sorted alphabetically
	assert.Contains(t, result, "import type { Auditable, BaseModel } from './base';")
}

func TestGenerateFile_MultipleImports(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)

	imports := []typegraph.ImportSpec{
		{
			ImportPath: "./base",
			TypeNames:  []string{"BaseModel"},
		},
		{
			ImportPath: "./timestamps",
			TypeNames:  []string{"Timestamped"},
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, imports)
	assert.NoError(t, err)
	assert.Contains(t, result, "import type { BaseModel } from './base';")
	assert.Contains(t, result, "import type { Timestamped } from './timestamps';")
}

func TestGenerateFile_ImportsSorted(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)

	imports := []typegraph.ImportSpec{
		{
			ImportPath: "./base",
			TypeNames:  []string{"Zebra", "Apple", "Middle"},
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, imports)
	assert.NoError(t, err)
	assert.Contains(t, result, "import type { Apple, Middle, Zebra } from './base';")
}

// Phase 2.5: Index Signatures (AdditionalProperties)

func TestGenerateIndexSignature_Disabled(t *testing.T) {
	g := createTestGenerator(&Config{
		AdditionalProperties: false,
	})
	typ := createTestType("Config", typegraph.KindStruct)
	typ.AdditionalProps = &typegraph.AdditionalPropsConfig{
		Allowed: true,
		Type:    &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
	}

	result := g.generateIndexSignature(typ)
	assert.Empty(t, result)
}

func TestGenerateIndexSignature_UntypedAny(t *testing.T) {
	g := createTestGenerator(&Config{
		AdditionalProperties: true,
	})
	typ := createTestType("Config", typegraph.KindStruct)
	typ.AdditionalProps = &typegraph.AdditionalPropsConfig{
		Allowed: true,
		Type:    nil, // untyped
	}

	result := g.generateIndexSignature(typ)
	assert.Contains(t, result, "[key: string]: any;")
}

func TestGenerateIndexSignature_TypedString(t *testing.T) {
	g := createTestGenerator(&Config{
		AdditionalProperties: true,
	})
	typ := createTestType("Config", typegraph.KindStruct)
	typ.AdditionalProps = &typegraph.AdditionalPropsConfig{
		Allowed: true,
		Type:    &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
	}

	result := g.generateIndexSignature(typ)
	assert.Contains(t, result, "[key: string]: string;")
}

func TestGenerateInterface_WithIndexSignature(t *testing.T) {
	g := createTestGenerator(&Config{
		AdditionalProperties: true,
	})
	typ := createTestType("Config", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("Name", "name", "string", true),
	}
	typ.AdditionalProps = &typegraph.AdditionalPropsConfig{
		Allowed: true,
		Type:    &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "int"},
	}

	result, err := g.generateInterface(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "name: string;")
	assert.Contains(t, result, "[key: string]: number;")
}

// Phase 3.1: Edge Cases

func TestGenerateFile_EmptyTypeList(t *testing.T) {
	g := createTestGenerator(nil)
	result, err := g.GenerateFile([]*typegraph.Type{}, nil)
	assert.NoError(t, err)
	assert.NotContains(t, result, "export")
}

func TestGenerateEnum_EmptyValues(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Status", typegraph.KindEnum)
	typ.EnumType = "string"
	typ.EnumValues = []typegraph.EnumValue{}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "export type Status =")
}

func TestGenerateInterface_FieldDescriptionWithQuotes(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:        "Name",
			JSONName:    "name",
			Description: `This is a "quoted" description`,
			Type:        &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
			Required:    true,
		},
	}

	result, err := g.generateInterface(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, `This is a "quoted" description`)
}

func TestGenerateInterface_UnicodeInNames(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Café", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("Naïve", "naïve", "string", true),
	}

	result, err := g.generateInterface(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "export interface Café")
	assert.Contains(t, result, "naïve: string;")
}

func TestTypeRefToTS_InlineEnum(t *testing.T) {
	g := createTestGenerator(nil)
	ref := &typegraph.TypeRef{
		Kind:       typegraph.KindEnum,
		EnumValues: []interface{}{"option1", "option2", 42, true, nil},
	}

	result := g.typeRefToTS(ref)
	assert.Contains(t, result, `"option1"`)
	assert.Contains(t, result, `"option2"`)
	assert.Contains(t, result, "42")
	assert.Contains(t, result, "true")
	assert.Contains(t, result, "null")
	assert.Contains(t, result, "|")
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
			Name:     "ID",
			JSONName: "id",
			Type: &typegraph.TypeRef{
				Kind:   typegraph.KindPrimitive,
				GoType: "string",
				Format: "uuid",
			},
			Required: true,
		},
		{
			Name:        "Email",
			JSONName:    "email",
			Description: "User email address",
			Type: &typegraph.TypeRef{
				Kind:   typegraph.KindPrimitive,
				GoType: "string",
				Format: "email",
			},
			Required: true,
		},
		{
			Name:     "Age",
			JSONName: "age",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "int"},
			Required: false,
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

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
		createTestField("Name", "name", "string", true),
	}

	enumType := createTestType("Status", typegraph.KindEnum)
	enumType.EnumType = "string"
	enumType.EnumValues = []typegraph.EnumValue{
		{Name: "Active", Value: "active"},
	}

	result, err := g.GenerateFile([]*typegraph.Type{interfaceType, enumType}, nil)
	assert.NoError(t, err)

	// Both types present, alphabetically ordered
	assert.Contains(t, result, "export type Status =")
	assert.Contains(t, result, "export interface User")

	// Status comes before User
	statusPos := strings.Index(result, "export type Status")
	userPos := strings.Index(result, "export interface User")
	assert.True(t, statusPos < userPos, "Status should come before User")
}

func TestGenerateFile_WithImportsAndAdditionalProps(t *testing.T) {
	g := createTestGenerator(&Config{
		AdditionalProperties: true,
	})

	typ := createTestType("Config", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("Name", "name", "string", true),
	}
	typ.AdditionalProps = &typegraph.AdditionalPropsConfig{
		Allowed: true,
		Type:    &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
	}

	imports := []typegraph.ImportSpec{
		{
			ImportPath: "./base",
			TypeNames:  []string{"BaseConfig"},
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, imports)
	assert.NoError(t, err)

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
			Name:     "Data",
			JSONName: "data",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "interface{}"},
			Required: true,
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)

	// Check no headers
	assert.NotContains(t, result, "DO NOT EDIT")
	// Check unknown instead of any
	assert.Contains(t, result, "data: unknown;")
	// Check comments still appear
	assert.Contains(t, result, "This comment should appear")
}
