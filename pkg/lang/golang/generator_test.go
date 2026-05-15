package golang

import (
	"strings"
	"testing"

	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Helpers
*/

func gen(cfg *Config) *Generator {
	return NewGenerator(cfg)
}

func structType(name string, fields ...*typegraph.Field) *typegraph.Type {
	return &typegraph.Type{
		Name:   name,
		Kind:   typegraph.KindStruct,
		Fields: fields,
	}
}

func enumType(name string, values ...typegraph.EnumValue) *typegraph.Type {
	return &typegraph.Type{
		Name:       name,
		Kind:       typegraph.KindEnum,
		EnumValues: values,
	}
}

func field(json string, prim typegraph.PrimitiveKind, required bool) *typegraph.Field {
	return &typegraph.Field{
		JSONName: json,
		Type: &typegraph.TypeRef{
			Kind:      typegraph.KindPrimitive,
			Primitive: prim,
		},
		Required: required,
	}
}

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
		field("id", typegraph.PrimUUID, true),
		field("name", typegraph.PrimString, false),
	)

	out, err := g.GenerateFile([]*typegraph.Type{user}, nil)
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
		typegraph.EnumValue{Name: "Active", Value: "active"},
		typegraph.EnumValue{Name: "Inactive", Value: "inactive"},
	)

	out, err := g.generateEnum(status)
	require.NoError(t, err)

	assert.Contains(t, out, "type Status string")
	assert.Contains(t, out, "const (")
}

func TestGenerateEnum_Number(t *testing.T) {
	g := gen(nil)

	priority := enumType("Priority",
		typegraph.EnumValue{Name: "Low", Value: float64(1)},
		typegraph.EnumValue{Name: "High", Value: float64(2)},
	)

	out, err := g.generateEnum(priority)
	require.NoError(t, err)

	assert.Contains(t, out, "type Priority int")
	assert.Contains(t, out, "PriorityLow Priority = 1")
}

func TestGenerateEnum_Mixed(t *testing.T) {
	g := gen(nil)

	mixed := enumType("Mixed",
		typegraph.EnumValue{Name: "A", Value: "a"},
		typegraph.EnumValue{Name: "B", Value: float64(1)},
	)

	out, err := g.generateEnum(mixed)
	require.NoError(t, err)

	assert.Contains(t, out, "type Mixed any")
	assert.Contains(t, out, "var MixedValues")
	assert.NotContains(t, out, "const (")
}

/*
3. Imports + validation
*/

func TestGenerateFile_Imports(t *testing.T) {
	g := gen(nil)

	event := structType("Event",
		field("id", typegraph.PrimUUID, true),
		field("created_at", typegraph.PrimDateTime, true),
	)

	out, err := g.GenerateFile([]*typegraph.Type{event}, nil)
	require.NoError(t, err)

	assert.Contains(t, out, `"github.com/google/uuid"`)
	assert.Contains(t, out, `"time"`)
}

func TestGenerateFile_NoValidatorImport(t *testing.T) {
	g := gen(nil)

	user := structType("User",
		field("name", typegraph.PrimString, true),
	)

	out, err := g.GenerateFile([]*typegraph.Type{user}, nil)
	require.NoError(t, err)

	assert.NotContains(t, out, "validator",
		"validator package should not be imported — struct tags work without it")
}

func TestFieldValidateTag_ExclusiveMinMax(t *testing.T) {
	g := gen(nil)
	excMin := float64(0)
	excMax := float64(100)

	f := &typegraph.Field{
		JSONName:         "score",
		Type:             &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimFloat64},
		ExclusiveMinimum: &excMin,
		ExclusiveMaximum: &excMax,
	}

	tag := g.fieldValidateTag(f)
	assert.Contains(t, tag, "gt=0")
	assert.Contains(t, tag, "lt=100")
}

func TestFieldValidateTag_Pattern(t *testing.T) {
	g := gen(nil)
	pattern := "^[a-z]+$"

	f := &typegraph.Field{
		JSONName: "code",
		Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimString},
		Pattern:  &pattern,
	}

	tag := g.fieldValidateTag(f)
	assert.Equal(t, "", tag, "Pattern not supported by go-playground/validator struct tags — should be omitted")
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

	user := structType("User", field("name", typegraph.PrimString, false))

	out, err := g.GenerateFile([]*typegraph.Type{user}, nil)
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
		&typegraph.Field{
			JSONName: "data",
			Type:     nil, // nil TypeRef — should not panic
			Required: true,
		},
	)

	assert.NotPanics(t, func() {
		_, _ = g.GenerateFile([]*typegraph.Type{typ}, nil)
	})
}

func TestTypeRefToGoType_Nil(t *testing.T) {
	g := gen(nil)
	assert.NotPanics(t, func() {
		result := g.typeRefToGoType(nil)
		assert.Equal(t, "interface{}", result)
	})
}

/*
6. Ordering & spacing
*/

func TestGenerateFile_TypeOrdering(t *testing.T) {
	g := gen(nil)

	out, err := g.GenerateFile([]*typegraph.Type{
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
