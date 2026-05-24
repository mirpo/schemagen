package golang

import (
	"strings"
	"testing"

	"github.com/mirpo/schemagen/pkg/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Helpers
*/

func gen(cfg *Config) *Generator {
	return NewGenerator(cfg)
}

func structType(name string, fields ...*graph.Field) *graph.Type {
	return &graph.Type{
		Name:   name,
		Kind:   graph.KindStruct,
		Fields: fields,
	}
}

func enumType(name string, values ...graph.EnumValue) *graph.Type {
	return &graph.Type{
		Name:       name,
		Kind:       graph.KindEnum,
		EnumValues: values,
	}
}

func field(json string, prim graph.PrimitiveKind, required bool) *graph.Field {
	return &graph.Field{
		JSONName: json,
		Type: &graph.TypeRef{
			Kind:      graph.KindPrimitive,
			Primitive: prim,
		},
		Required: required,
	}
}

func intPtr(v int) *int           { return &v }
func floatPtr(v float64) *float64 { return &v }

/*
1. Struct generation (core behavior)
*/

func TestGenerateFile_Struct(t *testing.T) {
	g := gen(&Config{
		PackageName: "models",
		UsePointers: true,
		OmitEmpty:   true,
	})

	user := structType("User",
		field("id", graph.PrimUUID, true),
		field("name", graph.PrimString, false),
	)

	out, err := g.GenerateFile([]*graph.Type{user}, nil)
	require.NoError(t, err)

	assert.Contains(t, out, "package models")
	assert.Contains(t, out, "type User struct")
	assert.Contains(t, out, "Id uuid.UUID")
	assert.Contains(t, out, "Name *string")
	assert.Contains(t, out, `json:"name,omitempty"`)
	assert.Contains(t, out, `"github.com/google/uuid"`)
}

/*
2. Enum generation strategies
*/

func TestGenerateEnum_String(t *testing.T) {
	g := gen(nil)

	status := enumType("Status",
		graph.EnumValue{Name: "Active", Value: "active"},
		graph.EnumValue{Name: "Inactive", Value: "inactive"},
	)

	out, err := g.generateEnum(status)
	require.NoError(t, err)

	assert.Contains(t, out, "type Status string")
	assert.Contains(t, out, "const (")
}

func TestGenerateEnum_Number(t *testing.T) {
	g := gen(nil)

	priority := enumType("Priority",
		graph.EnumValue{Name: "Low", Value: float64(1)},
		graph.EnumValue{Name: "High", Value: float64(2)},
	)

	out, err := g.generateEnum(priority)
	require.NoError(t, err)

	assert.Contains(t, out, "type Priority int")
	assert.Contains(t, out, "PriorityLow Priority = 1")
}

func TestGenerateEnum_Mixed(t *testing.T) {
	g := gen(nil)

	mixed := enumType("Mixed",
		graph.EnumValue{Name: "A", Value: "a"},
		graph.EnumValue{Name: "B", Value: float64(1)},
	)

	out, err := g.generateEnum(mixed)
	require.NoError(t, err)

	assert.Contains(t, out, "type Mixed any")
	assert.Contains(t, out, "var MixedValues")
	assert.NotContains(t, out, "const (")
}

/*
3. Union generation
*/

func TestGenerateUnion(t *testing.T) {
	g := gen(nil)

	union := &graph.Type{
		Name:        "Shape",
		Kind:        graph.KindUnion,
		Description: "A shape union",
		UnionMembers: []*graph.TypeRef{
			{Kind: graph.KindRef, TypeName: "Circle"},
			{Kind: graph.KindRef, TypeName: "Square"},
		},
	}

	out, err := g.generateUnion(union)
	require.NoError(t, err)
	assert.Contains(t, out, "type Shape = any")
}

func TestGenerateUnion_NoDescription(t *testing.T) {
	g := gen(&Config{DisableComments: true})

	union := &graph.Type{
		Name: "Value",
		Kind: graph.KindUnion,
	}

	out, err := g.generateUnion(union)
	require.NoError(t, err)
	assert.Contains(t, out, "type Value = any")
	assert.NotContains(t, out, "//")
}

func TestGenerateFile_UnionInOutput(t *testing.T) {
	g := gen(&Config{DisableHeaders: true})

	union := &graph.Type{
		Name: "Response",
		Kind: graph.KindUnion,
		UnionMembers: []*graph.TypeRef{
			{Kind: graph.KindRef, TypeName: "Success"},
			{Kind: graph.KindRef, TypeName: "Error"},
		},
	}

	out, err := g.GenerateFile([]*graph.Type{union}, nil)
	require.NoError(t, err)
	assert.Contains(t, out, "type Response = any")
}

/*
4. Imports + validation
*/

func TestGenerateFile_ValidatorImport(t *testing.T) {
	t.Run("blank import when validate tags exist", func(t *testing.T) {
		g := gen(nil)
		user := structType("User", field("name", graph.PrimString, true))
		out, err := g.GenerateFile([]*graph.Type{user}, nil)
		require.NoError(t, err)
		assert.Contains(t, out, `_ "github.com/go-playground/validator/v10"`)
	})

	t.Run("absent when no validate tags", func(t *testing.T) {
		user := structType("User", field("name", graph.PrimString, false))
		out, err := gen(&Config{OmitEmpty: false}).GenerateFile([]*graph.Type{user}, nil)
		require.NoError(t, err)
		assert.NotContains(t, out, "validator")
	})
}

func TestFieldValidateTag_ExclusiveMinMax(t *testing.T) {
	g := gen(nil)
	excMin := float64(0)
	excMax := float64(100)

	f := &graph.Field{
		JSONName:    "score",
		Type:        &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimFloat64},
		Constraints: graph.Constraints{ExclusiveMinimum: &excMin, ExclusiveMaximum: &excMax},
	}

	tag := g.fieldValidateTag(f)
	assert.Contains(t, tag, "gt=0")
	assert.Contains(t, tag, "lt=100")
}

func TestFieldValidateTag_Comprehensive(t *testing.T) {
	g := gen(nil)

	tests := []struct {
		name     string
		field    *graph.Field
		expected string
	}{
		{
			"required only",
			&graph.Field{JSONName: "x", Required: true, Type: &graph.TypeRef{Kind: graph.KindPrimitive}},
			"required",
		},
		{
			"min/max length",
			&graph.Field{JSONName: "x", Type: &graph.TypeRef{Kind: graph.KindPrimitive}, Constraints: graph.Constraints{MinLength: intPtr(5), MaxLength: intPtr(100)}},
			"min=5,max=100",
		},
		{
			"min/max numeric",
			&graph.Field{JSONName: "x", Type: &graph.TypeRef{Kind: graph.KindPrimitive}, Constraints: graph.Constraints{Minimum: floatPtr(0), Maximum: floatPtr(999)}},
			"gte=0,lte=999",
		},
		{
			"min/max items",
			&graph.Field{JSONName: "x", Type: &graph.TypeRef{Kind: graph.KindArray}, Constraints: graph.Constraints{MinItems: intPtr(1), MaxItems: intPtr(10)}},
			"min=1,max=10",
		},
		{
			"email format",
			&graph.Field{JSONName: "x", Required: true, Type: &graph.TypeRef{Kind: graph.KindPrimitive, Format: "email"}},
			"required,email",
		},
		{
			"uri format",
			&graph.Field{JSONName: "x", Type: &graph.TypeRef{Kind: graph.KindPrimitive, Format: "uri"}},
			"url",
		},
		{
			"url format",
			&graph.Field{JSONName: "x", Type: &graph.TypeRef{Kind: graph.KindPrimitive, Format: "url"}},
			"url",
		},
		{
			"uuid format",
			&graph.Field{JSONName: "x", Type: &graph.TypeRef{Kind: graph.KindPrimitive, Format: "uuid"}},
			"uuid",
		},
		{
			"inline enum oneof strings",
			&graph.Field{JSONName: "x", Type: &graph.TypeRef{Kind: graph.KindEnum, EnumValues: []graph.EnumValue{{Value: "a"}, {Value: "b"}, {Value: "c"}}}},
			"oneof=a b c",
		},
		{
			"inline enum oneof numbers",
			&graph.Field{JSONName: "x", Type: &graph.TypeRef{Kind: graph.KindEnum, EnumValues: []graph.EnumValue{{Value: float64(1)}, {Value: float64(2)}}}},
			"oneof=1 2",
		},
		{
			"no constraints not required",
			&graph.Field{JSONName: "x", Type: &graph.TypeRef{Kind: graph.KindPrimitive}},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, g.fieldValidateTag(tt.field))
		})
	}
}

func TestFieldHasValidation(t *testing.T) {
	tests := []struct {
		name     string
		field    *graph.Field
		expected bool
	}{
		{
			"required",
			&graph.Field{Required: true},
			true,
		},
		{
			"has constraints",
			&graph.Field{Constraints: graph.Constraints{MinLength: intPtr(1)}},
			true,
		},
		{
			"email format",
			&graph.Field{Type: &graph.TypeRef{Format: "email"}},
			true,
		},
		{
			"enum values",
			&graph.Field{Type: &graph.TypeRef{EnumValues: []graph.EnumValue{{Value: "a"}}}},
			true,
		},
		{
			"no validation",
			&graph.Field{Type: &graph.TypeRef{Kind: graph.KindPrimitive}},
			false,
		},
		{
			"nil type no required",
			&graph.Field{},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, fieldHasValidation(tt.field))
		})
	}
}

func TestFieldValidateTag_MultipleOf(t *testing.T) {
	g := gen(nil)
	mult := 0.5

	f := &graph.Field{
		JSONName:    "step",
		Type:        &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimFloat64},
		Constraints: graph.Constraints{MultipleOf: &mult},
	}

	tag := g.fieldValidateTag(f)
	assert.Empty(t, tag, "MultipleOf not supported by go-playground/validator struct tags — should be omitted")
}

func TestFieldValidateTag_Pattern(t *testing.T) {
	g := gen(nil)
	pattern := "^[a-z]+$"

	f := &graph.Field{
		JSONName:    "code",
		Type:        &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString},
		Constraints: graph.Constraints{Pattern: &pattern},
	}

	tag := g.fieldValidateTag(f)
	assert.Empty(t, tag, "Pattern not supported by go-playground/validator struct tags — should be omitted")
}

func TestWriteTypeComment(t *testing.T) {
	g := gen(nil)
	typ := &graph.Type{Name: "User", Kind: graph.KindStruct, Description: "A user"}
	out, err := g.GenerateFile([]*graph.Type{typ}, nil)
	require.NoError(t, err)
	assert.Contains(t, out, "// User A user")
}

/*
4. Config flags
*/

func TestGenerateFile_ConfigFlags(t *testing.T) {
	g := gen(&Config{
		DisableHeaders:  true,
		DisableComments: true,
		UsePointers:     false,
		OmitEmpty:       false,
	})

	user := structType("User", field("name", graph.PrimString, false))

	out, err := g.GenerateFile([]*graph.Type{user}, nil)
	require.NoError(t, err)

	assert.NotContains(t, out, "DO NOT EDIT")
	assert.NotContains(t, out, "//")
	assert.Contains(t, out, "Name string")
	assert.Contains(t, out, `json:"name"`)
}

/*
5. Nil safety
*/

func TestGenerateFile_NilTypeRef(t *testing.T) {
	g := gen(&Config{DisableHeaders: true})

	typ := structType("Event",
		&graph.Field{
			JSONName: "data",
			Type:     nil, // nil TypeRef — should not panic
			Required: true,
		},
	)

	assert.NotPanics(t, func() {
		_, _ = g.GenerateFile([]*graph.Type{typ}, nil)
	})
}

func TestTypeRefToGoType_KindCheckedBeforeTypeName(t *testing.T) {
	g := gen(nil)

	tests := []struct {
		name     string
		ref      *graph.TypeRef
		expected string
	}{
		{
			"KindRef with TypeName",
			&graph.TypeRef{Kind: graph.KindRef, TypeName: "User"},
			"User",
		},
		{
			"KindArray with TypeName should produce slice",
			&graph.TypeRef{Kind: graph.KindArray, TypeName: "ignored", ItemType: &graph.TypeRef{Kind: graph.KindRef, TypeName: "Item"}},
			"[]Item",
		},
		{
			"KindMap with TypeName should produce map",
			&graph.TypeRef{Kind: graph.KindMap, TypeName: "ignored", ValueType: &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString}},
			"map[string]string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.typeRefToGoType(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTypeRefToGoType_UsesAnyNotInterface(t *testing.T) {
	g := gen(nil)

	tests := []struct {
		name string
		ref  *graph.TypeRef
	}{
		{"nil ref", nil},
		{"interface kind", &graph.TypeRef{Kind: graph.KindInterface}},
		{"unknown primitive", &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimUnknown}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.typeRefToGoType(tt.ref)
			assert.Equal(t, "any", result)
		})
	}
}

/*
6. Ordering & spacing
*/

func TestGenerateFile_TypeOrdering(t *testing.T) {
	g := gen(nil)

	out, err := g.GenerateFile([]*graph.Type{
		structType("Zebra"),
		structType("Apple"),
		structType("Middle"),
	}, nil)
	require.NoError(t, err)

	z := strings.Index(out, "type Zebra")
	a := strings.Index(out, "type Apple")
	m := strings.Index(out, "type Middle")

	// Input order is preserved: Zebra, Apple, Middle
	assert.True(t, z < a && a < m, "types must preserve input order")
}
