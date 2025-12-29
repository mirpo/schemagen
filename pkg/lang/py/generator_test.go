package py

import (
	"strings"
	"testing"

	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/stretchr/testify/assert"
)

// Helper functions for creating test data

func createTestGenerator(config *Config) *Generator {
	graph := &typegraph.Graph{Types: []*typegraph.Type{}}
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

func createTestField(name string, jsonName string, pyType string, required bool) *typegraph.Field {
	return &typegraph.Field{
		Name:     name,
		JSONName: jsonName,
		Type: &typegraph.TypeRef{
			Kind:   typegraph.KindPrimitive,
			GoType: pyType,
		},
		Required: required,
	}
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func stringPtr(s string) *string {
	return &s
}

// ==================== Phase 1: Foundation Tests ====================

// 1.1: Basic Class Generation (5 tests)

func TestGenerateClass_Simple(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "class User(BaseModel):")
	assert.Contains(t, result, "pass")
}

func TestGenerateClass_WithDescription(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = "User model"

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "class User(BaseModel):")
	assert.Contains(t, result, `"""User model"""`)
}

func TestGenerateClass_EmptyFields(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Empty", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{}

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "pass")
}

func TestGenerateClass_MultipleFields(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("ID", "id", "string", true),
		createTestField("Email", "email", "string", true),
		createTestField("Age", "age", "int", true),
	}

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "id: str")
	assert.Contains(t, result, "email: str")
	assert.Contains(t, result, "age: int")
}

func TestGenerateClass_RequiredVsOptional(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("ID", "id", "string", true),
		createTestField("Email", "email", "string", false),
	}

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "id: str")
	assert.Contains(t, result, "email: str | None = None")
}

// 1.2: Basic Enum Generation (4 tests)

func TestGenerateEnum_StringEnum(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Status", typegraph.KindEnum)
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "ACTIVE", Value: "active"},
		{Name: "INACTIVE", Value: "inactive"},
	}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "class Status(str, Enum):")
	assert.Contains(t, result, `ACTIVE = "active"`)
	assert.Contains(t, result, `INACTIVE = "inactive"`)
}

func TestGenerateEnum_IntEnum(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Priority", typegraph.KindEnum)
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "LOW", Value: 1},
		{Name: "HIGH", Value: 2},
	}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "class Priority(IntEnum):")
	assert.Contains(t, result, "LOW = 1")
	assert.Contains(t, result, "HIGH = 2")
}

func TestGenerateEnum_WithDescription(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Status", typegraph.KindEnum)
	typ.Description = "User status"
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "ACTIVE", Value: "active"},
	}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, `"""User status"""`)
}

func TestGenerateEnum_MixedTypesUsesLiteral(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Mixed", typegraph.KindEnum)
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "STR", Value: "text"},
		{Name: "NUM", Value: 42},
	}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "Literal[")
	assert.Contains(t, result, `"text"`)
	assert.Contains(t, result, "42")
}

// 1.3: Basic Type Conversion (6 tests)

func TestTypeRefToPython_Primitives(t *testing.T) {
	tests := []struct {
		name     string
		goType   string
		optional bool
		expected string
	}{
		{"string type", "string", false, "str"},
		{"int type", "int", false, "int"},
		{"float64 type", "float64", false, "float"},
		{"bool type", "bool", false, "bool"},
		{"optional string", "string", true, "str | None"},
		{"interface type", "interface{}", false, "Any"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := createTestGenerator(nil)
			ref := &typegraph.TypeRef{
				Kind:   typegraph.KindPrimitive,
				GoType: tt.goType,
			}

			result := g.typeRefToPython(ref, tt.optional)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTypeRefToPython_Arrays(t *testing.T) {
	g := createTestGenerator(nil)
	ref := &typegraph.TypeRef{
		Kind: typegraph.KindArray,
		ItemType: &typegraph.TypeRef{
			Kind:   typegraph.KindPrimitive,
			GoType: "string",
		},
	}

	result := g.typeRefToPython(ref, false)
	assert.Equal(t, "list[str]", result)
}

func TestTypeRefToPython_Maps(t *testing.T) {
	g := createTestGenerator(nil)
	ref := &typegraph.TypeRef{
		Kind: typegraph.KindMap,
		ValueType: &typegraph.TypeRef{
			Kind:   typegraph.KindPrimitive,
			GoType: "int",
		},
	}

	result := g.typeRefToPython(ref, false)
	assert.Equal(t, "dict[str, int]", result)
}

func TestTypeRefToPython_NamedTypes(t *testing.T) {
	g := createTestGenerator(nil)
	ref := &typegraph.TypeRef{
		Kind:     typegraph.KindRef,
		TypeName: "User",
	}

	result := g.typeRefToPython(ref, false)
	assert.Equal(t, "User", result)
}

func TestTypeRefToPython_Union(t *testing.T) {
	g := createTestGenerator(nil)
	ref := &typegraph.TypeRef{
		Kind: typegraph.KindUnion,
		UnionMembers: []*typegraph.TypeRef{
			{Kind: typegraph.KindPrimitive, GoType: "string"},
			{Kind: typegraph.KindPrimitive, GoType: "int"},
		},
	}

	result := g.typeRefToPython(ref, false)
	assert.Equal(t, "str | int", result)
}

func TestTypeRefToPython_Nested(t *testing.T) {
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

	result := g.typeRefToPython(ref, false)
	assert.Equal(t, "list[dict[str, str]]", result)
}

// 1.4: File Generation Basics (5 tests)

func TestGenerateFile_WithHeader(t *testing.T) {
	g := createTestGenerator(&Config{DisableHeaders: false})
	typ := createTestType("User", typegraph.KindStruct)

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "DO NOT EDIT")
	assert.Contains(t, result, "from __future__ import annotations")
}

func TestGenerateFile_WithoutHeader(t *testing.T) {
	g := createTestGenerator(&Config{DisableHeaders: true})
	typ := createTestType("User", typegraph.KindStruct)

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.NotContains(t, result, "DO NOT EDIT")
	assert.Contains(t, result, "from __future__ import annotations")
}

func TestGenerateFile_MultipleTypes(t *testing.T) {
	g := createTestGenerator(nil)
	typ1 := createTestType("User", typegraph.KindStruct)
	typ2 := createTestType("Post", typegraph.KindStruct)

	result, err := g.GenerateFile([]*typegraph.Type{typ1, typ2}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "class User(BaseModel):")
	assert.Contains(t, result, "class Post(BaseModel):")
}

func TestGenerateFile_TypeOrdering(t *testing.T) {
	g := createTestGenerator(nil)
	typ1 := createTestType("Zebra", typegraph.KindStruct)
	typ2 := createTestType("Apple", typegraph.KindStruct)

	result, err := g.GenerateFile([]*typegraph.Type{typ1, typ2}, nil)
	assert.NoError(t, err)

	// Types should be sorted alphabetically
	appleIdx := strings.Index(result, "class Apple")
	zebraIdx := strings.Index(result, "class Zebra")
	assert.True(t, appleIdx < zebraIdx, "Types should be alphabetically sorted")
}

func TestGenerateFile_BlankLinesBetweenTypes(t *testing.T) {
	g := createTestGenerator(nil)
	typ1 := createTestType("User", typegraph.KindStruct)
	typ2 := createTestType("Post", typegraph.KindStruct)

	result, err := g.GenerateFile([]*typegraph.Type{typ1, typ2}, nil)
	assert.NoError(t, err)

	// Should have double newline between types
	assert.Contains(t, result, "pass\n\n\nclass")
}

// ==================== Phase 2: Core Features ====================

// 2.1: Field Constraints with Field() (10 tests - TABLE-DRIVEN)

func TestFieldWithConstraints(t *testing.T) {
	tests := []struct {
		name             string
		setupField       func() *typegraph.Field
		expectFieldCall  bool
		expectedContains []string
	}{
		{
			name: "Required field no constraints",
			setupField: func() *typegraph.Field {
				return createTestField("Name", "name", "string", true)
			},
			expectFieldCall:  false,
			expectedContains: []string{"name: str"},
		},
		{
			name: "Optional field no constraints",
			setupField: func() *typegraph.Field {
				f := createTestField("Name", "name", "string", false)
				return f
			},
			expectFieldCall:  false,
			expectedContains: []string{"name: str | None = None"},
		},
		{
			name: "Field with description",
			setupField: func() *typegraph.Field {
				f := createTestField("Name", "name", "string", true)
				f.Description = "User's name"
				return f
			},
			expectFieldCall:  true,
			expectedContains: []string{"Field(...", `description="User's name"`},
		},
		{
			name: "String with minLength",
			setupField: func() *typegraph.Field {
				f := createTestField("Name", "name", "string", true)
				f.MinLength = intPtr(5)
				return f
			},
			expectFieldCall:  true,
			expectedContains: []string{"Field(...", "min_length=5"},
		},
		{
			name: "String with maxLength",
			setupField: func() *typegraph.Field {
				f := createTestField("Name", "name", "string", true)
				f.MaxLength = intPtr(50)
				return f
			},
			expectFieldCall:  true,
			expectedContains: []string{"Field(...", "max_length=50"},
		},
		{
			name: "Number with minimum (ge)",
			setupField: func() *typegraph.Field {
				f := createTestField("Age", "age", "int", true)
				f.Minimum = float64Ptr(0)
				return f
			},
			expectFieldCall:  true,
			expectedContains: []string{"Field(...", "ge=0"},
		},
		{
			name: "Number with maximum (le)",
			setupField: func() *typegraph.Field {
				f := createTestField("Age", "age", "int", true)
				f.Maximum = float64Ptr(150)
				return f
			},
			expectFieldCall:  true,
			expectedContains: []string{"Field(...", "le=150"},
		},
		{
			name: "Number with exclusive minimum (gt)",
			setupField: func() *typegraph.Field {
				f := createTestField("Score", "score", "float64", true)
				f.ExclusiveMinimum = float64Ptr(0)
				return f
			},
			expectFieldCall:  true,
			expectedContains: []string{"Field(...", "gt=0"},
		},
		{
			name: "Array with minItems",
			setupField: func() *typegraph.Field {
				f := &typegraph.Field{
					Name:     "Tags",
					JSONName: "tags",
					Type: &typegraph.TypeRef{
						Kind: typegraph.KindArray,
						ItemType: &typegraph.TypeRef{
							Kind:   typegraph.KindPrimitive,
							GoType: "string",
						},
					},
					Required: true,
					MinItems: intPtr(1),
				}
				return f
			},
			expectFieldCall:  true,
			expectedContains: []string{"Field(...", "min_length=1"},
		},
		{
			name: "Pattern constraint",
			setupField: func() *typegraph.Field {
				f := createTestField("Code", "code", "string", true)
				f.Pattern = stringPtr("^[A-Z]{3}$")
				return f
			},
			expectFieldCall:  true,
			expectedContains: []string{"Field(...", `pattern="^[A-Z]{3}$"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := createTestGenerator(nil)
			typ := createTestType("Test", typegraph.KindStruct)
			typ.Fields = []*typegraph.Field{tt.setupField()}

			result, err := g.generateClass(typ)
			assert.NoError(t, err)

			if tt.expectFieldCall {
				assert.Contains(t, result, "Field(")
			}

			for _, expected := range tt.expectedContains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

// 2.2: Snake Case Field Conversion (4 tests)

func TestSnakeCaseField_Disabled(t *testing.T) {
	g := createTestGenerator(&Config{SnakeCaseField: false})
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("FirstName", "firstName", "string", true),
	}

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "firstName: str")
}

func TestSnakeCaseField_Enabled(t *testing.T) {
	g := createTestGenerator(&Config{SnakeCaseField: true})
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("FirstName", "firstName", "string", true),
	}

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "first_name: str")
	assert.Contains(t, result, `alias="firstName"`)
}

func TestSnakeCaseField_AlreadySnakeCase(t *testing.T) {
	g := createTestGenerator(&Config{SnakeCaseField: true})
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("FirstName", "first_name", "string", true),
	}

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "first_name: str")
	assert.NotContains(t, result, "alias=")
}

func TestSnakeCaseField_OptionalWithAlias(t *testing.T) {
	g := createTestGenerator(&Config{SnakeCaseField: true})
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("FirstName", "firstName", "string", false),
	}

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "first_name: str | None")
	assert.Contains(t, result, "Field(None")
	assert.Contains(t, result, `alias="firstName"`)
}

// 2.3: Python Identifier Sanitization (6 tests)

func TestSanitizePythonIdentifier_ValidNames(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"valid_name", "valid_name"},
		{"ValidName", "ValidName"},
		{"_private", "_private"},
		{"name123", "name123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizePythonIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizePythonIdentifier_InvalidStartChar(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"123name", "field_123name"},
		{"$special", "field__special"},
		{"@attr", "field__attr"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizePythonIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizePythonIdentifier_Keywords(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"class", "class_"},
		{"def", "def_"},
		{"return", "return_"},
		{"if", "if_"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizePythonIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizePythonIdentifier_SpecialChars(t *testing.T) {
	result := sanitizePythonIdentifier("kebab-case-name")
	assert.Equal(t, "kebab_case_name", result)

	result = sanitizePythonIdentifier("dotted.name")
	assert.Equal(t, "dotted_name", result)
}

func TestSanitizePythonIdentifier_EmptyString(t *testing.T) {
	result := sanitizePythonIdentifier("")
	assert.Equal(t, "field", result)
}

func TestGenerateClass_SanitizedFieldNames(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("Class", "class", "string", true),
		createTestField("Name", "kebab-name", "string", true),
	}

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "class_: str")
	assert.Contains(t, result, `alias="class"`)
	assert.Contains(t, result, "kebab_name: str")
	assert.Contains(t, result, `alias="kebab-name"`)
}

// 2.4: Class Composition/Inheritance (3 tests)

func TestGenerateClass_WithExtends(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Extends = []string{"BaseModel"}

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "class User(BaseModel):")
}

func TestGenerateClass_MultipleExtends(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Extends = []string{"Timestamped", "Auditable"}

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "class User(Timestamped, Auditable):")
	assert.NotContains(t, result, "BaseModel") // Should not include BaseModel when extending
}

func TestGenerateClass_ExtendsWithNoFields(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("SpecialUser", typegraph.KindStruct)
	typ.Extends = []string{"User"}
	typ.Fields = []*typegraph.Field{}

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "class SpecialUser(User):")
	assert.Contains(t, result, "pass")
}

// 2.5: Configuration Options (2 tests)

func TestNewGenerator_DefaultConfig(t *testing.T) {
	graph := &typegraph.Graph{}
	g := NewGenerator(graph)

	assert.NotNil(t, g)
	assert.NotNil(t, g.config)
	assert.False(t, g.config.DisableHeaders)
	assert.False(t, g.config.SnakeCaseField)
}

func TestNewGeneratorWithConfig_CustomConfig(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		DisableHeaders:   true,
		DisableTimestamp: true,
		SnakeCaseField:   true,
	}
	g := NewGeneratorWithConfig(graph, cfg)

	assert.NotNil(t, g)
	assert.True(t, g.config.DisableHeaders)
	assert.True(t, g.config.DisableTimestamp)
	assert.True(t, g.config.SnakeCaseField)
}

// ==================== Phase 3: Complex Scenarios ====================

// 3.1: Import Generation (10 tests)

func TestImports_NoImports(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Simple", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("Name", "name", "string", true),
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "from pydantic import BaseModel")
	assert.NotContains(t, result, "from typing import")
}

func TestImports_TypingAny(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:     "Metadata",
			JSONName: "metadata",
			Type: &typegraph.TypeRef{
				Kind:   typegraph.KindPrimitive,
				GoType: "interface{}",
			},
			Required: true,
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "from typing import Any")
	assert.Contains(t, result, "metadata: Any")
}

func TestImports_UUIDType(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:     "ID",
			JSONName: "id",
			Type: &typegraph.TypeRef{
				Kind:   typegraph.KindPrimitive,
				GoType: "uuid.UUID",
			},
			Required: true,
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "from uuid import UUID")
	assert.Contains(t, result, "id: UUID")
}

func TestImports_DateTimeType(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Event", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:     "CreatedAt",
			JSONName: "created_at",
			Type: &typegraph.TypeRef{
				Kind:   typegraph.KindPrimitive,
				GoType: "time.Time",
			},
			Required: true,
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "from datetime import datetime")
	assert.Contains(t, result, "created_at: datetime")
}

func TestImports_FieldConstraints(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:      "Name",
			JSONName:  "name",
			Type:      &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
			Required:  true,
			MinLength: intPtr(1),
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "from pydantic import BaseModel, Field")
}

func TestImports_EnumClass(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Status", typegraph.KindEnum)
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "ACTIVE", Value: "active"},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "from enum import Enum")
}

func TestImports_IntEnumClass(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Priority", typegraph.KindEnum)
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "LOW", Value: 1},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "from enum import IntEnum")
}

func TestImports_LiteralType(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Mixed", typegraph.KindEnum)
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "STR", Value: "text"},
		{Name: "NUM", Value: 42},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "from typing import Literal")
}

func TestImports_NestedTypes(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:     "IDs",
			JSONName: "ids",
			Type: &typegraph.TypeRef{
				Kind: typegraph.KindArray,
				ItemType: &typegraph.TypeRef{
					Kind:   typegraph.KindPrimitive,
					GoType: "uuid.UUID",
				},
			},
			Required: true,
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "from uuid import UUID")
	assert.Contains(t, result, "ids: list[UUID]")
}

func TestImports_MultipleImports(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:     "ID",
			JSONName: "id",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "uuid.UUID"},
			Required: true,
		},
		{
			Name:     "CreatedAt",
			JSONName: "created_at",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "time.Time"},
			Required: true,
		},
		{
			Name:      "Email",
			JSONName:  "email",
			Type:      &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
			Required:  true,
			MinLength: intPtr(1),
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "from datetime import datetime")
	assert.Contains(t, result, "from uuid import UUID")
	assert.Contains(t, result, "from pydantic import BaseModel, Field")
}

// 3.2: Edge Cases (8 tests)

func TestGenerateFile_EmptyTypeList(t *testing.T) {
	g := createTestGenerator(nil)

	result, err := g.GenerateFile([]*typegraph.Type{}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "from __future__ import annotations")
}

func TestGenerateClass_FieldWithNilConstraints(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:             "Age",
			JSONName:         "age",
			Type:             &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "int"},
			Required:         true,
			MinLength:        nil,
			MaxLength:        nil,
			Minimum:          nil,
			Maximum:          nil,
			ExclusiveMinimum: nil,
			ExclusiveMaximum: nil,
			MinItems:         nil,
			MaxItems:         nil,
		},
	}

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "age: int")
	assert.NotContains(t, result, "Field(")
}

func TestGenerateEnum_EmptyValues(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Empty", typegraph.KindEnum)
	typ.EnumValues = []typegraph.EnumValue{}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)
	// Empty enum generates class declaration but no body (which is technically invalid Python)
	assert.Contains(t, result, "class Empty(str, Enum):")
}

func TestGenerateClass_DescriptionWithQuotes(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = `User's "special" model`

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, `"""User's "special" model"""`)
}

func TestGenerateClass_DescriptionWithTripleQuotes(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = `This has """ triple quotes`

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	// Should escape triple quotes
	assert.Contains(t, result, `\"\"\"`)
}

func TestGenerateClass_UnicodeInNames(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Unicöde", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("Näme", "näme", "string", true),
	}

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "class Unicöde(BaseModel):")
}

func TestSanitizeEnumMemberName_NumericNames(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1", "N_1"},
		{"42", "N_42"},
		{"ABC", "ABC"},
		{"", "EMPTY"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeEnumMemberName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatPythonString_QuoteChoice(t *testing.T) {
	// String with double quotes should use single quotes
	result := formatPythonString(`He said "hello"`)
	assert.Contains(t, result, "'")

	// String with single quotes should use double quotes
	result = formatPythonString(`It's working`)
	assert.Contains(t, result, `"`)
}

// 3.3: Integration Tests (5 tests)

func TestGenerateFile_CompleteClass(t *testing.T) {
	g := createTestGenerator(&Config{SnakeCaseField: true})
	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = "User model with all features"
	typ.Fields = []*typegraph.Field{
		{
			Name:     "ID",
			JSONName: "id",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "uuid.UUID"},
			Required: true,
		},
		{
			Name:      "Email",
			JSONName:  "email",
			Type:      &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
			Required:  true,
			MinLength: intPtr(5),
			MaxLength: intPtr(100),
		},
		{
			Name:     "Age",
			JSONName: "age",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "int"},
			Required: false,
			Minimum:  float64Ptr(0),
			Maximum:  float64Ptr(150),
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "class User(BaseModel):")
	assert.Contains(t, result, `"""User model with all features"""`)
	assert.Contains(t, result, "id: UUID")
	assert.Contains(t, result, "email: str")
	assert.Contains(t, result, "age: int | None")
	assert.Contains(t, result, "min_length=5")
	assert.Contains(t, result, "ge=0")
	assert.Contains(t, result, "from uuid import UUID")
	assert.Contains(t, result, "from pydantic import BaseModel, Field")
}

func TestGenerateFile_CompleteEnum(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Status", typegraph.KindEnum)
	typ.Description = "User status values"
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "ACTIVE", Value: "active"},
		{Name: "INACTIVE", Value: "inactive"},
		{Name: "PENDING", Value: "pending"},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "class Status(str, Enum):")
	assert.Contains(t, result, `"""User status values"""`)
	assert.Contains(t, result, `ACTIVE = "active"`)
	assert.Contains(t, result, `INACTIVE = "inactive"`)
	assert.Contains(t, result, `PENDING = "pending"`)
	assert.Contains(t, result, "from enum import Enum")
}

func TestGenerateFile_MixedTypes(t *testing.T) {
	g := createTestGenerator(nil)
	class := createTestType("User", typegraph.KindStruct)
	class.Fields = []*typegraph.Field{
		createTestField("Name", "name", "string", true),
	}
	enum := createTestType("Status", typegraph.KindEnum)
	enum.EnumValues = []typegraph.EnumValue{
		{Name: "ACTIVE", Value: "active"},
	}

	result, err := g.GenerateFile([]*typegraph.Type{class, enum}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "class User(BaseModel):")
	assert.Contains(t, result, "class Status(str, Enum):")
	assert.Contains(t, result, "from enum import Enum")
	assert.Contains(t, result, "from pydantic import BaseModel")
}

func TestGenerateFile_WithImportsAndConstraints(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Event", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:     "ID",
			JSONName: "id",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "uuid.UUID"},
			Required: true,
		},
		{
			Name:     "Timestamp",
			JSONName: "timestamp",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "time.Time"},
			Required: true,
		},
		{
			Name:      "Description",
			JSONName:  "description",
			Type:      &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
			Required:  true,
			MinLength: intPtr(10),
			MaxLength: intPtr(500),
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "from datetime import datetime")
	assert.Contains(t, result, "from uuid import UUID")
	assert.Contains(t, result, "from pydantic import BaseModel, Field")
	assert.Contains(t, result, "id: UUID")
	assert.Contains(t, result, "timestamp: datetime")
	assert.Contains(t, result, "description: str")
	assert.Contains(t, result, "min_length=10")
	assert.Contains(t, result, "max_length=500")
}

func TestGenerateFile_AllConfigOptions(t *testing.T) {
	g := createTestGenerator(&Config{
		DisableHeaders:   true,
		DisableTimestamp: true,
		SnakeCaseField:   true,
	})
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("FirstName", "firstName", "string", true),
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	assert.NoError(t, err)
	assert.NotContains(t, result, "DO NOT EDIT")
	assert.NotContains(t, result, "timestamp")
	assert.Contains(t, result, "first_name: str")
	assert.Contains(t, result, `alias="firstName"`)
}

// ==================== AdditionalProperties Tests ====================

func TestGenerateFile_WithAdditionalPropertiesFlag_AddsConfigDictImport(t *testing.T) {
	graph := &typegraph.Graph{
		Types: []*typegraph.Type{
			{
				Name:        "User",
				Kind:        typegraph.KindStruct,
				Description: "User model",
				Fields:      []*typegraph.Field{},
				AdditionalProps: &typegraph.AdditionalPropsConfig{
					Allowed: true,
					Type:    nil,
				},
			},
		},
	}

	g := NewGeneratorWithConfig(graph, &Config{
		AllowExtraFields: true,
	})

	result, err := g.GenerateFile(graph.Types, nil)
	assert.NoError(t, err)
	assert.Contains(t, result, "from pydantic import BaseModel, ConfigDict")
}

func TestGenerateClass_WithAdditionalPropertiesFlag_AddsModelConfig(t *testing.T) {
	typ := &typegraph.Type{
		Name:        "DynamicObject",
		Kind:        typegraph.KindStruct,
		Description: "Object with dynamic properties",
		Fields: []*typegraph.Field{
			{
				Name:     "Name",
				JSONName: "name",
				Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
				Required: true,
			},
		},
		AdditionalProps: &typegraph.AdditionalPropsConfig{
			Allowed: true,
			Type:    nil,
		},
	}

	g := NewGeneratorWithConfig(&typegraph.Graph{}, &Config{
		AllowExtraFields: true,
	})
	g.imports = make(map[string]bool)
	g.imports["pydantic"] = true
	g.imports["pydantic_config"] = true

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "model_config = ConfigDict(extra='allow')")
}

func TestGenerateClass_WithoutAdditionalPropertiesFlag_NoModelConfig(t *testing.T) {
	typ := &typegraph.Type{
		Name:   "DynamicObject",
		Kind:   typegraph.KindStruct,
		Fields: []*typegraph.Field{},
		AdditionalProps: &typegraph.AdditionalPropsConfig{
			Allowed: true,
			Type:    nil,
		},
	}

	g := NewGeneratorWithConfig(&typegraph.Graph{}, &Config{
		AllowExtraFields: false,
	})
	g.imports = make(map[string]bool)
	g.imports["pydantic"] = true

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.NotContains(t, result, "model_config")
	assert.NotContains(t, result, "ConfigDict")
}

func TestGenerateClass_WithoutAdditionalProps_NoModelConfig(t *testing.T) {
	typ := &typegraph.Type{
		Name:            "StrictObject",
		Kind:            typegraph.KindStruct,
		Fields:          []*typegraph.Field{},
		AdditionalProps: nil,
	}

	g := NewGeneratorWithConfig(&typegraph.Graph{}, &Config{
		AllowExtraFields: true,
	})
	g.imports = make(map[string]bool)
	g.imports["pydantic"] = true

	result, err := g.generateClass(typ)
	assert.NoError(t, err)
	assert.NotContains(t, result, "model_config")
}
