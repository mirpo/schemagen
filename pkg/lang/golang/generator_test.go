package golang

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
	return NewGenerator(graph, config)
}

func createTestType(name string, kind typegraph.TypeKind) *typegraph.Type {
	return &typegraph.Type{
		Name:   name,
		Kind:   kind,
		Fields: []*typegraph.Field{},
	}
}

func createTestField(name string, jsonName string, goType string, required bool) *typegraph.Field {
	return &typegraph.Field{
		Name:     name,
		JSONName: jsonName,
		Type: &typegraph.TypeRef{
			Kind:   typegraph.KindPrimitive,
			GoType: goType,
		},
		Required: required,
	}
}

// Phase 1.1: Basic Struct Generation

func TestGenerateStruct_Simple(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("ID", "id", "string", true),
		createTestField("Name", "name", "string", true),
	}

	result, err := g.generateStruct(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "type User struct {")
	assert.Contains(t, result, "Id string") // "id" → "Id" (not "ID" unless we add abbreviation handling)
	assert.Contains(t, result, "Name string")
	assert.Contains(t, result, `json:"id"`)
	assert.Contains(t, result, `json:"name"`)
}

func TestGenerateStruct_WithDescription(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = "represents a user in the system"
	typ.Fields = []*typegraph.Field{
		createTestField("ID", "id", "string", true),
	}

	result, err := g.generateStruct(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "// User represents a user in the system")
	assert.Contains(t, result, "type User struct {")
}

func TestGenerateStruct_EmptyFields(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Empty", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{}

	result, err := g.generateStruct(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "type Empty struct {")
	assert.Contains(t, result, "}")
}

func TestGenerateStruct_MultipleFields(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Product", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("ID", "id", "string", true),
		createTestField("Name", "name", "string", true),
		createTestField("Price", "price", "float64", true),
		createTestField("Available", "available", "bool", true),
	}

	result, err := g.generateStruct(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "Id string") // "id" → "Id"
	assert.Contains(t, result, "Name string")
	assert.Contains(t, result, "Price float64")
	assert.Contains(t, result, "Available bool")
}

func TestGenerateStruct_CommentsDisabled(t *testing.T) {
	g := createTestGenerator(&Config{
		PackageName:     "models",
		DisableComments: true,
	})
	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = "This should not appear"
	typ.Fields = []*typegraph.Field{
		createTestField("ID", "id", "string", true),
	}
	typ.Fields[0].Description = "This should also not appear"

	result, err := g.generateStruct(typ)
	assert.NoError(t, err)
	assert.NotContains(t, result, "This should not appear")
	assert.NotContains(t, result, "This should also not appear")
	assert.Contains(t, result, "type User struct {")
}

// Phase 1.2: Basic Enum Generation

func TestGenerateEnum_StringOnly(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Status", typegraph.KindEnum)
	typ.EnumType = "string"
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "Active", Value: "active"},
		{Name: "Inactive", Value: "inactive"},
	}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "type Status string")
	assert.Contains(t, result, "const (")
	assert.Contains(t, result, `StatusActive Status = "active"`)
	assert.Contains(t, result, `StatusInactive Status = "inactive"`)
}

func TestGenerateEnum_IntegerOnly(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Priority", typegraph.KindEnum)
	typ.EnumType = "int"
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "Low", Value: float64(1)},
		{Name: "High", Value: float64(2)},
	}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "type Priority int")
	assert.Contains(t, result, "const (")
	assert.Contains(t, result, `PriorityLow Priority = 1`)
	assert.Contains(t, result, `PriorityHigh Priority = 2`)
}

func TestGenerateEnum_WithDescription(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Status", typegraph.KindEnum)
	typ.Description = "represents status values"
	typ.EnumType = "string"
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "Active", Value: "active"},
	}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "// Status represents status values")
	assert.Contains(t, result, "type Status string")
}

func TestGenerateEnum_CommentsDisabled(t *testing.T) {
	g := createTestGenerator(&Config{
		PackageName:     "models",
		DisableComments: true,
	})
	typ := createTestType("Status", typegraph.KindEnum)
	typ.Description = "This should not appear"
	typ.EnumType = "string"
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "Active", Value: "active"},
	}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)
	assert.NotContains(t, result, "This should not appear")
	assert.Contains(t, result, "type Status string")
}

// Phase 1.3: Basic Type Conversion

func TestTypeRefToGoType_Primitives(t *testing.T) {
	tests := []struct {
		name     string
		goType   string
		expected string
	}{
		{"string type", "string", "string"},
		{"int type", "int", "int"},
		{"int64 type", "int64", "int64"},
		{"float64 type", "float64", "float64"},
		{"bool type", "bool", "bool"},
	}

	g := createTestGenerator(nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := &typegraph.TypeRef{
				Kind:   typegraph.KindPrimitive,
				GoType: tt.goType,
			}
			result := g.typeRefToGoType(ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTypeRefToGoType_Arrays(t *testing.T) {
	g := createTestGenerator(nil)
	ref := &typegraph.TypeRef{
		Kind: typegraph.KindArray,
		ItemType: &typegraph.TypeRef{
			Kind:   typegraph.KindPrimitive,
			GoType: "string",
		},
	}

	result := g.typeRefToGoType(ref)
	assert.Equal(t, "[]string", result)
}

func TestTypeRefToGoType_Maps(t *testing.T) {
	g := createTestGenerator(nil)
	ref := &typegraph.TypeRef{
		Kind: typegraph.KindMap,
		ValueType: &typegraph.TypeRef{
			Kind:   typegraph.KindPrimitive,
			GoType: "int",
		},
	}

	result := g.typeRefToGoType(ref)
	assert.Equal(t, "map[string]int", result)
}

func TestTypeRefToGoType_NamedTypes(t *testing.T) {
	g := createTestGenerator(nil)
	ref := &typegraph.TypeRef{
		Kind:     typegraph.KindPrimitive,
		TypeName: "User",
	}

	result := g.typeRefToGoType(ref)
	assert.Equal(t, "User", result)
}

func TestTypeRefToGoType_Union(t *testing.T) {
	g := createTestGenerator(nil)
	ref := &typegraph.TypeRef{
		Kind: typegraph.KindUnion,
	}

	result := g.typeRefToGoType(ref)
	assert.Equal(t, "any", result)
}

func TestTypeRefToGoType_Nested(t *testing.T) {
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

	result := g.typeRefToGoType(ref)
	assert.Equal(t, "[]map[string]string", result)
}

// Phase 1.4: File Generation Basics

func TestGenerateFile_PackageDeclaration(t *testing.T) {
	g := createTestGenerator(&Config{
		PackageName: "mymodels",
	})
	typ := createTestType("User", typegraph.KindStruct)

	result, err := g.GenerateFile([]*typegraph.Type{typ}, []typegraph.ImportSpec{})
	assert.NoError(t, err)
	assert.Contains(t, result, "package mymodels")
}

func TestGenerateFile_WithHeader(t *testing.T) {
	g := createTestGenerator(&Config{
		PackageName:    "models",
		DisableHeaders: false,
	})
	typ := createTestType("User", typegraph.KindStruct)

	result, err := g.GenerateFile([]*typegraph.Type{typ}, []typegraph.ImportSpec{})
	assert.NoError(t, err)
	assert.Contains(t, result, "DO NOT EDIT")
}

func TestGenerateFile_WithoutHeader(t *testing.T) {
	g := createTestGenerator(&Config{
		PackageName:    "models",
		DisableHeaders: true,
	})
	typ := createTestType("User", typegraph.KindStruct)

	result, err := g.GenerateFile([]*typegraph.Type{typ}, []typegraph.ImportSpec{})
	assert.NoError(t, err)
	assert.NotContains(t, result, "DO NOT EDIT")
	assert.Contains(t, result, "package models")
}

func TestGenerateFile_MultipleTypes(t *testing.T) {
	g := createTestGenerator(nil)
	types := []*typegraph.Type{
		createTestType("User", typegraph.KindStruct),
		createTestType("Product", typegraph.KindStruct),
		createTestType("Order", typegraph.KindStruct),
	}

	result, err := g.GenerateFile(types, []typegraph.ImportSpec{})
	assert.NoError(t, err)
	assert.Contains(t, result, "type User struct")
	assert.Contains(t, result, "type Product struct")
	assert.Contains(t, result, "type Order struct")
}

func TestGenerateFile_TypeOrdering(t *testing.T) {
	g := createTestGenerator(nil)
	types := []*typegraph.Type{
		createTestType("Zebra", typegraph.KindStruct),
		createTestType("Apple", typegraph.KindStruct),
		createTestType("Middle", typegraph.KindStruct),
	}

	result, err := g.GenerateFile(types, []typegraph.ImportSpec{})
	assert.NoError(t, err)

	// Find positions of each type declaration
	applePos := strings.Index(result, "type Apple struct")
	middlePos := strings.Index(result, "type Middle struct")
	zebraPos := strings.Index(result, "type Zebra struct")

	// Verify alphabetical ordering
	assert.True(t, applePos < middlePos, "Apple should come before Middle")
	assert.True(t, middlePos < zebraPos, "Middle should come before Zebra")
}

// Helper functions for Phase 2+

func intPtr(v int) *int {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}

// Phase 2.1: Pointer Handling

func TestFieldGoType_OptionalWithPointers(t *testing.T) {
	g := createTestGenerator(&Config{
		PackageName: "models",
		UsePointers: true,
	})
	field := createTestField("Name", "name", "string", false) // optional field

	result := g.fieldGoType(field)
	assert.Equal(t, "*string", result)
}

func TestFieldGoType_OptionalWithoutPointers(t *testing.T) {
	g := createTestGenerator(&Config{
		PackageName: "models",
		UsePointers: false,
	})
	field := createTestField("Name", "name", "string", false) // optional field

	result := g.fieldGoType(field)
	assert.Equal(t, "string", result)
}

func TestFieldGoType_RequiredField(t *testing.T) {
	g := createTestGenerator(&Config{
		PackageName: "models",
		UsePointers: true,
	})
	field := createTestField("Name", "name", "string", true) // required field

	result := g.fieldGoType(field)
	assert.Equal(t, "string", result) // Required fields never have pointers
}

func TestFieldGoType_PointerArrays(t *testing.T) {
	g := createTestGenerator(&Config{
		PackageName: "models",
		UsePointers: true,
	})
	field := &typegraph.Field{
		Name:     "Tags",
		JSONName: "tags",
		Type: &typegraph.TypeRef{
			Kind: typegraph.KindArray,
			ItemType: &typegraph.TypeRef{
				Kind:   typegraph.KindPrimitive,
				GoType: "string",
			},
		},
		Required: false, // optional
	}

	result := g.fieldGoType(field)
	assert.Equal(t, "*[]string", result) // Pointer to array for optional
}

func TestFieldGoType_PointerMaps(t *testing.T) {
	g := createTestGenerator(&Config{
		PackageName: "models",
		UsePointers: true,
	})
	field := &typegraph.Field{
		Name:     "Metadata",
		JSONName: "metadata",
		Type: &typegraph.TypeRef{
			Kind: typegraph.KindMap,
			ValueType: &typegraph.TypeRef{
				Kind:   typegraph.KindPrimitive,
				GoType: "string",
			},
		},
		Required: false, // optional
	}

	result := g.fieldGoType(field)
	assert.Equal(t, "*map[string]string", result) // Pointer to map for optional
}

func TestFieldGoType_Config_UsePointers(t *testing.T) {
	field := createTestField("Name", "name", "string", false)

	// Test with UsePointers=true
	gTrue := createTestGenerator(&Config{UsePointers: true})
	assert.Equal(t, "*string", gTrue.fieldGoType(field))

	// Test with UsePointers=false
	gFalse := createTestGenerator(&Config{UsePointers: false})
	assert.Equal(t, "string", gFalse.fieldGoType(field))
}

// Phase 2.2: JSON Tag Generation

func TestFieldJSONTag(t *testing.T) {
	tests := []struct {
		name     string
		field    *typegraph.Field
		config   *Config
		expected string
	}{
		{
			name:     "Required field (no omitempty)",
			field:    createTestField("ID", "id", "string", true),
			config:   &Config{OmitEmpty: true},
			expected: "id",
		},
		{
			name:     "Optional field with OmitEmpty=true",
			field:    createTestField("Name", "name", "string", false),
			config:   &Config{OmitEmpty: true},
			expected: "name,omitempty",
		},
		{
			name:     "Optional field with OmitEmpty=false",
			field:    createTestField("Name", "name", "string", false),
			config:   &Config{OmitEmpty: false},
			expected: "name",
		},
		{
			name:     "Field with custom JSONName",
			field:    createTestField("UserID", "user_id", "string", true),
			config:   &Config{OmitEmpty: false},
			expected: "user_id",
		},
		{
			name:     "Field with snake_case name",
			field:    createTestField("FirstName", "first_name", "string", true),
			config:   &Config{OmitEmpty: false},
			expected: "first_name",
		},
		{
			name:     "Field with hyphen in name",
			field:    createTestField("UpdatedAt", "updated-at", "string", false),
			config:   &Config{OmitEmpty: true},
			expected: "updated-at,omitempty",
		},
		{
			name:     "Field with special characters",
			field:    createTestField("Special", "@special", "string", true),
			config:   &Config{OmitEmpty: false},
			expected: "@special",
		},
		{
			name: "Empty JSONName",
			field: &typegraph.Field{
				Name:     "Field",
				JSONName: "",
				Required: true,
			},
			config:   &Config{OmitEmpty: false},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := createTestGenerator(tt.config)
			result := g.fieldJSONTag(tt.field)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Phase 2.3: Validation Tag Generation

func TestFieldValidateTag(t *testing.T) {
	tests := []struct {
		name     string
		field    *typegraph.Field
		expected string
	}{
		{
			name: "Required field",
			field: &typegraph.Field{
				Name:     "ID",
				JSONName: "id",
				Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
				Required: true,
			},
			expected: "required",
		},
		{
			name: "String with minLength",
			field: &typegraph.Field{
				Name:      "Name",
				JSONName:  "name",
				Type:      &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
				Required:  false,
				MinLength: intPtr(5),
			},
			expected: "min=5",
		},
		{
			name: "String with maxLength",
			field: &typegraph.Field{
				Name:      "Name",
				JSONName:  "name",
				Type:      &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
				Required:  false,
				MaxLength: intPtr(100),
			},
			expected: "max=100",
		},
		{
			name: "String with min+max",
			field: &typegraph.Field{
				Name:      "Username",
				JSONName:  "username",
				Type:      &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
				Required:  false,
				MinLength: intPtr(3),
				MaxLength: intPtr(20),
			},
			expected: "min=3,max=20",
		},
		{
			name: "Number with minimum",
			field: &typegraph.Field{
				Name:     "Age",
				JSONName: "age",
				Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "int"},
				Required: false,
				Minimum:  float64Ptr(18),
			},
			expected: "gte=18",
		},
		{
			name: "Number with maximum",
			field: &typegraph.Field{
				Name:     "Score",
				JSONName: "score",
				Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "float64"},
				Required: false,
				Maximum:  float64Ptr(100),
			},
			expected: "lte=100",
		},
		{
			name: "Number with min+max",
			field: &typegraph.Field{
				Name:     "Rating",
				JSONName: "rating",
				Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "float64"},
				Required: false,
				Minimum:  float64Ptr(1),
				Maximum:  float64Ptr(5),
			},
			expected: "gte=1,lte=5",
		},
		{
			name: "Array with minItems",
			field: &typegraph.Field{
				Name:     "Tags",
				JSONName: "tags",
				Type:     &typegraph.TypeRef{Kind: typegraph.KindArray},
				Required: false,
				MinItems: intPtr(1),
			},
			expected: "min=1",
		},
		{
			name: "Array with maxItems",
			field: &typegraph.Field{
				Name:     "Tags",
				JSONName: "tags",
				Type:     &typegraph.TypeRef{Kind: typegraph.KindArray},
				Required: false,
				MaxItems: intPtr(10),
			},
			expected: "max=10",
		},
		{
			name: "Email format",
			field: &typegraph.Field{
				Name:     "Email",
				JSONName: "email",
				Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string", Format: "email"},
				Required: false,
			},
			expected: "email",
		},
		{
			name: "URL format",
			field: &typegraph.Field{
				Name:     "Website",
				JSONName: "website",
				Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string", Format: "uri"},
				Required: false,
			},
			expected: "url",
		},
		{
			name: "UUID format",
			field: &typegraph.Field{
				Name:     "ID",
				JSONName: "id",
				Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string", Format: "uuid"},
				Required: false,
			},
			expected: "uuid",
		},
		{
			name: "Enum values (string)",
			field: &typegraph.Field{
				Name:     "Status",
				JSONName: "status",
				Type: &typegraph.TypeRef{
					Kind:       typegraph.KindEnum,
					GoType:     "string",
					EnumValues: []interface{}{"active", "inactive", "pending"},
				},
				Required: false,
			},
			expected: "oneof=active inactive pending",
		},
		{
			name: "Enum values (int)",
			field: &typegraph.Field{
				Name:     "Priority",
				JSONName: "priority",
				Type: &typegraph.TypeRef{
					Kind:       typegraph.KindEnum,
					GoType:     "int",
					EnumValues: []interface{}{1, 2, 3},
				},
				Required: false,
			},
			expected: "oneof=1 2 3",
		},
		{
			name: "Combined constraints",
			field: &typegraph.Field{
				Name:      "Username",
				JSONName:  "username",
				Type:      &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
				Required:  true,
				MinLength: intPtr(5),
				MaxLength: intPtr(10),
			},
			expected: "required,min=5,max=10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := createTestGenerator(nil)
			result := g.fieldValidateTag(tt.field)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Phase 2.4: Struct Composition

func TestGenerateStruct_WithExtends(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Extends = []string{"BaseModel"}
	typ.Fields = []*typegraph.Field{
		createTestField("Name", "name", "string", true),
	}

	result, err := g.generateStruct(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "BaseModel")
	assert.Contains(t, result, "Name string")
}

func TestGenerateStruct_MultipleExtends(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Extends = []string{"BaseModel", "Timestamped", "Auditable"}

	result, err := g.generateStruct(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "BaseModel")
	assert.Contains(t, result, "Timestamped")
	assert.Contains(t, result, "Auditable")
}

func TestGenerateStruct_ExtendsWithFields(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Extends = []string{"BaseModel"}
	typ.Fields = []*typegraph.Field{
		createTestField("Name", "name", "string", true),
		createTestField("Email", "email", "string", true),
	}

	result, err := g.generateStruct(typ)
	assert.NoError(t, err)
	// Base type should come before fields
	basePos := strings.Index(result, "BaseModel")
	namePos := strings.Index(result, "Name string")
	assert.True(t, basePos < namePos, "BaseModel should come before Name field")
}

func TestGenerateStruct_UnionTypeComment(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Response", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:     "Data",
			JSONName: "data",
			Type: &typegraph.TypeRef{
				Kind: typegraph.KindUnion,
				UnionMembers: []*typegraph.TypeRef{
					{Kind: typegraph.KindPrimitive, GoType: "string"},
					{Kind: typegraph.KindPrimitive, GoType: "int"},
					{TypeName: "User"},
				},
			},
			Required: true,
		},
	}

	result, err := g.generateStruct(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "// Can be one of:")
	assert.Contains(t, result, "string")
	assert.Contains(t, result, "int")
	assert.Contains(t, result, "User")
}

// Phase 2.5: Configuration Variations

func TestNewGenerator_DefaultConfig(t *testing.T) {
	graph := &typegraph.Graph{Types: []*typegraph.Type{}}
	g := NewGenerator(graph, nil) // nil config

	// Should have default values
	assert.NotNil(t, g.config)
	assert.Equal(t, "models", g.config.PackageName)
	assert.True(t, g.config.UsePointers)
	assert.True(t, g.config.OmitEmpty)
	assert.False(t, g.config.DisableComments)
}

func TestNewGenerator_CustomConfig(t *testing.T) {
	graph := &typegraph.Graph{Types: []*typegraph.Type{}}
	customCfg := &Config{
		PackageName:      "mypackage",
		UsePointers:      false,
		OmitEmpty:        false,
		DisableComments:  true,
		DisableHeaders:   true,
		DisableTimestamp: true,
	}
	g := NewGenerator(graph, customCfg)

	// Should use custom config
	assert.Equal(t, "mypackage", g.config.PackageName)
	assert.False(t, g.config.UsePointers)
	assert.False(t, g.config.OmitEmpty)
	assert.True(t, g.config.DisableComments)
	assert.True(t, g.config.DisableHeaders)
	assert.True(t, g.config.DisableTimestamp)
}

// Phase 3.1: Import Generation

func TestScanTypeForImports_NoImports(t *testing.T) {
	g := createTestGenerator(nil)
	g.resetImports()

	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:     "Name",
			JSONName: "name",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
			Required: false, // Not required and no other constraints
		},
	}

	g.scanTypeForImports(typ)
	assert.Empty(t, g.imports)
}

func TestScanTypeForImports_ValidationImport(t *testing.T) {
	g := createTestGenerator(nil)
	g.resetImports()

	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:      "Email",
			JSONName:  "email",
			Type:      &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
			Required:  true,
			MinLength: intPtr(5),
		},
	}

	g.scanTypeForImports(typ)
	assert.True(t, g.imports["github.com/go-playground/validator/v10"])
}

func TestScanTypeForImports_UUIDImport(t *testing.T) {
	g := createTestGenerator(nil)
	g.resetImports()

	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:     "ID",
			JSONName: "id",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "uuid.UUID"},
			Required: true,
		},
	}

	g.scanTypeForImports(typ)
	assert.True(t, g.imports["github.com/google/uuid"])
}

func TestScanTypeForImports_TimeImport(t *testing.T) {
	g := createTestGenerator(nil)
	g.resetImports()

	typ := createTestType("Event", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:     "CreatedAt",
			JSONName: "created_at",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "time.Time"},
			Required: true,
		},
	}

	g.scanTypeForImports(typ)
	assert.True(t, g.imports["time"])
}

func TestScanTypeForImports_MultipleImports(t *testing.T) {
	g := createTestGenerator(nil)
	g.resetImports()

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
			Type:      &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string", Format: "email"},
			Required:  false,
			MinLength: intPtr(5),
		},
	}

	g.scanTypeForImports(typ)
	assert.True(t, g.imports["github.com/google/uuid"])
	assert.True(t, g.imports["time"])
	assert.True(t, g.imports["github.com/go-playground/validator/v10"])
}

func TestScanTypeForImports_NestedTypes(t *testing.T) {
	g := createTestGenerator(nil)
	g.resetImports()

	typ := createTestType("Event", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:     "Timestamps",
			JSONName: "timestamps",
			Type: &typegraph.TypeRef{
				Kind: typegraph.KindArray,
				ItemType: &typegraph.TypeRef{
					Kind:   typegraph.KindPrimitive,
					GoType: "time.Time",
				},
			},
			Required: true,
		},
	}

	g.scanTypeForImports(typ)
	assert.True(t, g.imports["time"])
}

func TestScanTypeForImports_MapValues(t *testing.T) {
	g := createTestGenerator(nil)
	g.resetImports()

	typ := createTestType("Config", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:     "Metadata",
			JSONName: "metadata",
			Type: &typegraph.TypeRef{
				Kind: typegraph.KindMap,
				ValueType: &typegraph.TypeRef{
					Kind:   typegraph.KindPrimitive,
					GoType: "time.Time",
				},
			},
			Required: true,
		},
	}

	g.scanTypeForImports(typ)
	assert.True(t, g.imports["time"])
}

func TestScanTypeForImports_ArrayItems(t *testing.T) {
	g := createTestGenerator(nil)
	g.resetImports()

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

	g.scanTypeForImports(typ)
	assert.True(t, g.imports["github.com/google/uuid"])
}

func TestGenerateFile_ImportBlock(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:     "ID",
			JSONName: "id",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "uuid.UUID"},
			Required: true,
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, []typegraph.ImportSpec{})
	assert.NoError(t, err)
	assert.Contains(t, result, "import (")
	assert.Contains(t, result, `"github.com/google/uuid"`)
	assert.Contains(t, result, ")")
}

func TestGenerateFile_ImportSorting(t *testing.T) {
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
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, []typegraph.ImportSpec{})
	assert.NoError(t, err)

	// Check imports are sorted alphabetically
	uuidPos := strings.Index(result, `"github.com/google/uuid"`)
	timePos := strings.Index(result, `"time"`)
	assert.True(t, uuidPos < timePos, "github.com/google/uuid should come before time")
}

// Phase 3.2: Edge Cases

func TestGenerateFile_EmptyTypeList(t *testing.T) {
	g := createTestGenerator(nil)
	result, err := g.GenerateFile([]*typegraph.Type{}, []typegraph.ImportSpec{})
	assert.NoError(t, err)
	assert.Contains(t, result, "package models")
	assert.NotContains(t, result, "type")
}

func TestGenerateStruct_FieldWithNilConstraints(t *testing.T) {
	g := createTestGenerator(&Config{
		PackageName: "models",
		UsePointers: false, // Disable pointers to test non-pointer optional fields
	})
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:      "Name",
			JSONName:  "name",
			Type:      &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
			Required:  false,
			MinLength: nil,
			MaxLength: nil,
		},
	}

	result, err := g.generateStruct(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "Name string")
}

func TestGenerateEnum_EmptyValues(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Status", typegraph.KindEnum)
	typ.EnumType = "string"
	typ.EnumValues = []typegraph.EnumValue{}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "type Status string")
	assert.Contains(t, result, "const (")
}

func TestFieldValidateTag_NilValues(t *testing.T) {
	g := createTestGenerator(nil)
	field := &typegraph.Field{
		Name:      "Name",
		JSONName:  "name",
		Type:      &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
		Required:  false,
		MinLength: nil,
		MaxLength: nil,
		Minimum:   nil,
		Maximum:   nil,
	}

	result := g.fieldValidateTag(field)
	assert.Equal(t, "", result)
}

func TestTypeRefToGoType_EmptyTypeName(t *testing.T) {
	g := createTestGenerator(nil)
	ref := &typegraph.TypeRef{
		Kind:     typegraph.KindPrimitive,
		TypeName: "",
		GoType:   "string",
	}

	result := g.typeRefToGoType(ref)
	assert.Equal(t, "string", result)
}

func TestGenerateStruct_FieldDescriptionWithQuotes(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:        "Name",
			JSONName:    "name",
			Type:        &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
			Description: `This is a "quoted" description`,
			Required:    true,
		},
	}

	result, err := g.generateStruct(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, `This is a "quoted" description`)
}

func TestGenerateStruct_FieldDescriptionWithBackslashes(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Path", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			Name:        "FilePath",
			JSONName:    "file_path",
			Type:        &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
			Description: `Path like C:\Users\Name`,
			Required:    true,
		},
	}

	result, err := g.generateStruct(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, `Path like C:\Users\Name`)
}

func TestGenerateStruct_UnicodeInNames(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("Café", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("Naïve", "naive", "string", true),
	}

	result, err := g.generateStruct(typ)
	assert.NoError(t, err)
	assert.Contains(t, result, "type Café struct")
	assert.Contains(t, result, "Naive string") // Unicode ï is converted to plain i by ToGoFieldName
}

func TestFieldJSONTag_SpecialCharacters(t *testing.T) {
	g := createTestGenerator(nil)
	field := &typegraph.Field{
		Name:     "Field",
		JSONName: "field@123",
		Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
		Required: true,
	}

	result := g.fieldJSONTag(field)
	assert.Equal(t, "field@123", result)
}

func TestGenerateFile_TypeNameWithNumbers(t *testing.T) {
	g := createTestGenerator(nil)
	typ := createTestType("User123", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("Field1", "field1", "string", true),
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, []typegraph.ImportSpec{})
	assert.NoError(t, err)
	assert.Contains(t, result, "type User123 struct")
	assert.Contains(t, result, "Field1 string")
}

// Phase 3.3: Integration Tests

func TestGenerateFile_CompleteStruct(t *testing.T) {
	g := createTestGenerator(&Config{
		PackageName:    "models",
		UsePointers:    true,
		OmitEmpty:      true,
		DisableHeaders: false,
	})

	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = "User represents a system user"
	typ.Extends = []string{"BaseModel"}
	typ.Fields = []*typegraph.Field{
		{
			Name:     "ID",
			JSONName: "id",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "uuid.UUID"},
			Required: true,
		},
		{
			Name:        "Email",
			JSONName:    "email",
			Description: "User email address",
			Type:        &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string", Format: "email"},
			Required:    true,
			MinLength:   intPtr(5),
			MaxLength:   intPtr(100),
		},
		{
			Name:     "Age",
			JSONName: "age",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "int"},
			Required: false,
			Minimum:  float64Ptr(18),
			Maximum:  float64Ptr(120),
		},
		{
			Name:     "CreatedAt",
			JSONName: "created_at",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "time.Time"},
			Required: true,
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, []typegraph.ImportSpec{})
	assert.NoError(t, err)

	// Check header
	assert.Contains(t, result, "DO NOT EDIT")
	// Check package
	assert.Contains(t, result, "package models")
	// Check imports
	assert.Contains(t, result, `"github.com/go-playground/validator/v10"`)
	assert.Contains(t, result, `"github.com/google/uuid"`)
	assert.Contains(t, result, `"time"`)
	// Check type comment
	assert.Contains(t, result, "// User User represents a system user")
	// Check struct
	assert.Contains(t, result, "type User struct {")
	assert.Contains(t, result, "BaseModel")
	// Check fields
	assert.Contains(t, result, "Id uuid.UUID") // "id" → "Id"
	assert.Contains(t, result, "Email string")
	assert.Contains(t, result, "Age *int")
	assert.Contains(t, result, "CreatedAt time.Time")
	// Check tags
	assert.Contains(t, result, `json:"id"`)
	assert.Contains(t, result, `json:"email"`)
	assert.Contains(t, result, `json:"age,omitempty"`)
	assert.Contains(t, result, `validate:"required"`)
	assert.Contains(t, result, `validate:"required,min=5,max=100,email"`)
	assert.Contains(t, result, `validate:"gte=18,lte=120"`)
}

func TestGenerateFile_CompleteEnum(t *testing.T) {
	g := createTestGenerator(&Config{
		PackageName:    "models",
		DisableHeaders: false,
	})

	typ := createTestType("Status", typegraph.KindEnum)
	typ.Description = "Status represents user status"
	typ.EnumType = "string"
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "Active", Value: "active"},
		{Name: "Inactive", Value: "inactive"},
		{Name: "Pending", Value: "pending"},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, []typegraph.ImportSpec{})
	assert.NoError(t, err)

	// Check header
	assert.Contains(t, result, "DO NOT EDIT")
	// Check package
	assert.Contains(t, result, "package models")
	// Check type comment
	assert.Contains(t, result, "// Status Status represents user status")
	// Check type declaration
	assert.Contains(t, result, "type Status string")
	// Check constants
	assert.Contains(t, result, "const (")
	assert.Contains(t, result, `StatusActive Status = "active"`)
	assert.Contains(t, result, `StatusInactive Status = "inactive"`)
	assert.Contains(t, result, `StatusPending Status = "pending"`)
}

func TestGenerateFile_MixedTypes(t *testing.T) {
	g := createTestGenerator(nil)

	structType := createTestType("User", typegraph.KindStruct)
	structType.Fields = []*typegraph.Field{
		createTestField("Name", "name", "string", true),
	}

	enumType := createTestType("Status", typegraph.KindEnum)
	enumType.EnumType = "string"
	enumType.EnumValues = []typegraph.EnumValue{
		{Name: "Active", Value: "active"},
	}

	result, err := g.GenerateFile([]*typegraph.Type{structType, enumType}, []typegraph.ImportSpec{})
	assert.NoError(t, err)

	// Both types should be present, alphabetically ordered
	assert.Contains(t, result, "type Status string")
	assert.Contains(t, result, "type User struct")

	// Status should come before User
	statusPos := strings.Index(result, "type Status string")
	userPos := strings.Index(result, "type User struct")
	assert.True(t, statusPos < userPos, "Status should come before User alphabetically")
}

// Issue 1: Blank Lines Between Type Definitions
func TestGenerateFile_BlankLinesBetweenTypes(t *testing.T) {
	g := createTestGenerator(nil)

	types := []*typegraph.Type{
		createTestType("Apple", typegraph.KindStruct),
		createTestType("Banana", typegraph.KindStruct),
		createTestType("Cherry", typegraph.KindStruct),
	}

	result, err := g.GenerateFile(types, []typegraph.ImportSpec{})
	assert.NoError(t, err)

	// Extract just the types section (after imports)
	lines := strings.Split(result, "\n")
	var typeLines []string
	inTypes := false
	for _, line := range lines {
		if strings.HasPrefix(line, "type ") {
			inTypes = true
		}
		if inTypes {
			typeLines = append(typeLines, line)
		}
	}

	typesSection := strings.Join(typeLines, "\n")

	// Check that there are double newlines between type definitions
	// Pattern: "}\n\ntype" should exist between structs
	assert.Contains(t, typesSection, "}\n\ntype", "Types should be separated by blank lines")

	// Count occurrences - should have 2 separations for 3 types
	separationCount := strings.Count(typesSection, "}\n\ntype")
	assert.Equal(t, 2, separationCount, "Should have 2 blank line separations for 3 types")
}

func TestGenerateFile_WithImportsAndValidation(t *testing.T) {
	g := createTestGenerator(&Config{
		PackageName: "models",
		UsePointers: true,
		OmitEmpty:   true,
	})

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
	}

	// Add file imports
	fileImports := []typegraph.ImportSpec{
		{ImportPath: "github.com/custom/package"},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, fileImports)
	assert.NoError(t, err)

	// Check all imports are present
	assert.Contains(t, result, `"github.com/custom/package"`)
	assert.Contains(t, result, `"github.com/go-playground/validator/v10"`)
	assert.Contains(t, result, `"github.com/google/uuid"`)
	assert.Contains(t, result, `"time"`)
}

func TestGenerateFile_AllConfigOptions(t *testing.T) {
	g := createTestGenerator(&Config{
		PackageName:      "custompackage",
		UsePointers:      false,
		OmitEmpty:        false,
		DisableComments:  true,
		DisableHeaders:   true,
		DisableTimestamp: true,
	})

	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = "This comment should not appear"
	typ.Fields = []*typegraph.Field{
		{
			Name:        "Name",
			JSONName:    "name",
			Description: "This comment should not appear either",
			Type:        &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
			Required:    false,
		},
	}

	result, err := g.GenerateFile([]*typegraph.Type{typ}, []typegraph.ImportSpec{})
	assert.NoError(t, err)

	// Check custom package name
	assert.Contains(t, result, "package custompackage")
	// Check no headers
	assert.NotContains(t, result, "DO NOT EDIT")
	// Check no comments
	assert.NotContains(t, result, "This comment should not appear")
	// Check no pointers for optional fields
	assert.Contains(t, result, "Name string")
	assert.NotContains(t, result, "Name *string")
	// Check no omitempty
	assert.Contains(t, result, `json:"name"`)
	assert.NotContains(t, result, "omitempty")
}

// Issue 2: Complex Enum Generation
func TestGenerateEnum_ComplexValues(t *testing.T) {
	g := createTestGenerator(nil)

	typ := createTestType("Status", typegraph.KindEnum)
	typ.EnumType = "any"
	typ.HasComplexValues = true
	typ.Description = "Enum with mixed types"
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "SIMPLE", Value: "simple"},
		{Name: "OBJECT", Value: map[string]any{"key": "value"}},
		{Name: "NUMBER", Value: float64(42)},
		{Name: "ARRAY", Value: []any{"a", "b"}},
	}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)

	// Check type declaration
	assert.Contains(t, result, "type Status any")

	// Check var declaration (not const) - using []Status to tie values to the type
	assert.Contains(t, result, "var StatusValues = []Status{")

	// Check values are present
	assert.Contains(t, result, `"simple"`)
	assert.Contains(t, result, `map[string]any{"key": "value"}`)
	assert.Contains(t, result, "42")
	assert.Contains(t, result, `[]any{"a", "b"}`)

	// Ensure NO const block
	assert.NotContains(t, result, "const (")
}

// Test for number-only enums (should use const with numeric literals)
func TestGenerateEnum_NumberOnlyValues(t *testing.T) {
	g := createTestGenerator(nil)

	typ := createTestType("Priority", typegraph.KindEnum)
	typ.EnumType = "int"
	typ.HasComplexValues = false
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "1", Value: float64(1)},
		{Name: "2", Value: float64(2)},
		{Name: "3", Value: float64(3)},
	}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)

	// Check type declaration
	assert.Contains(t, result, "type Priority int")

	// Check const block with NUMERIC literals (not quoted)
	assert.Contains(t, result, "const (")
	assert.Contains(t, result, "Priority1 Priority = 1")
	assert.Contains(t, result, "Priority2 Priority = 2")
	assert.Contains(t, result, "Priority3 Priority = 3")

	// Should NOT have %!q garbage
	assert.NotContains(t, result, "%!q")

	// Should NOT have quoted numbers
	assert.NotContains(t, result, `= "1"`)
	assert.NotContains(t, result, `= "2"`)
}

// Test for mixed primitive enums (string + number + bool + null)
// These should use var pattern since they can't all be represented as same const type
func TestGenerateEnum_MixedPrimitives(t *testing.T) {
	g := createTestGenerator(nil)

	typ := createTestType("MixedEnum", typegraph.KindEnum)
	typ.EnumType = "any"         // Builder sets this for mixed
	typ.HasComplexValues = false // Builder doesn't flag bool/null as complex
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "STRING", Value: "hello"},
		{Name: "42", Value: float64(42)},
		{Name: "TRUE", Value: true},
		{Name: "NIL", Value: nil},
	}

	result, err := g.generateEnum(typ)
	assert.NoError(t, err)

	// Mixed primitives should use var pattern like complex enums
	assert.Contains(t, result, "type MixedEnum any")
	assert.Contains(t, result, "var MixedEnumValues = []MixedEnum{")

	// Check values are formatted correctly
	assert.Contains(t, result, `"hello"`)
	assert.Contains(t, result, "42")
	assert.Contains(t, result, "true")
	assert.Contains(t, result, "nil")

	// Should NOT have %!q garbage
	assert.NotContains(t, result, "%!q")

	// Should NOT have const block
	assert.NotContains(t, result, "const (")
}

// Issue 4: Union Generation
func TestGenerateUnion(t *testing.T) {
	g := createTestGenerator(nil)

	typ := createTestType("UnionNullable", typegraph.KindUnion)
	typ.Description = "Union with nullable members"
	typ.UnionMembers = []*typegraph.TypeRef{
		{GoType: "string"},
		{GoType: "nil"},
		{GoType: "int"},
	}

	result, err := g.generateUnion(typ)
	assert.NoError(t, err)

	// Check comment
	assert.Contains(t, result, "// UnionNullable Union with nullable members")

	// Check type alias (with =)
	assert.Contains(t, result, "type UnionNullable = any")

	// Should NOT be "type UnionNullable any" (without =)
	assert.NotContains(t, result, "type UnionNullable any\n")
}

func TestGenerateUnion_NoDescription(t *testing.T) {
	g := createTestGenerator(nil)

	typ := createTestType("SimpleUnion", typegraph.KindUnion)
	typ.UnionMembers = []*typegraph.TypeRef{
		{TypeName: "OptionA"},
		{TypeName: "OptionB"},
	}

	result, err := g.generateUnion(typ)
	assert.NoError(t, err)

	// Should have type alias
	assert.Contains(t, result, "type SimpleUnion = any")

	// Should not have empty comment line
	assert.NotContains(t, result, "// SimpleUnion \n")
}
