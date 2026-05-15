package py

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newGen(cfg *Config) *Generator {
	if cfg == nil {
		cfg = &Config{}
	}
	return NewGeneratorWithConfig(&typegraph.Graph{}, cfg)
}

func TestGenerateFile_SimpleStruct(t *testing.T) {
	g := newGen(nil)

	typ := &typegraph.Type{
		Name: "User",
		Kind: typegraph.KindStruct,
		Fields: []*typegraph.Field{
			{
				Name:     "ID",
				JSONName: "id",
				Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimString},
				Required: true,
			},
		},
	}

	out, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	require.NoError(t, err)

	assert.Contains(t, out, "class User(BaseModel):")
	assert.Contains(t, out, "id: str")
	assert.Contains(t, out, "from pydantic import BaseModel")
}

func TestGenerateFile_OptionalField(t *testing.T) {
	g := newGen(nil)

	typ := &typegraph.Type{
		Name: "User",
		Kind: typegraph.KindStruct,
		Fields: []*typegraph.Field{
			{
				Name:     "Email",
				JSONName: "email",
				Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimString},
				Required: false,
			},
		},
	}

	out, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	require.NoError(t, err)

	assert.Contains(t, out, "email: str | None")
	assert.Contains(t, out, "= None")
}

func TestGenerateFile_StringEnum(t *testing.T) {
	g := newGen(nil)

	typ := &typegraph.Type{
		Name: "Status",
		Kind: typegraph.KindEnum,
		EnumValues: []typegraph.EnumValue{
			{Name: "ACTIVE", Value: "active"},
			{Name: "DISABLED", Value: "disabled"},
		},
	}

	out, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	require.NoError(t, err)

	assert.Contains(t, out, "class Status(str, Enum):")
	assert.Contains(t, out, `ACTIVE = "active"`)
	assert.Contains(t, out, "from enum import Enum")
}

func TestGenerateFile_IntEnum(t *testing.T) {
	g := newGen(nil)

	typ := &typegraph.Type{
		Name: "Priority",
		Kind: typegraph.KindEnum,
		EnumValues: []typegraph.EnumValue{
			{Name: "LOW", Value: 1},
			{Name: "HIGH", Value: 2},
		},
	}

	out, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	require.NoError(t, err)

	assert.Contains(t, out, "class Priority(IntEnum):")
	assert.Contains(t, out, "LOW = 1")
	assert.Contains(t, out, "from enum import IntEnum")
}

func TestGenerateFile_MixedEnum_UsesLiteral(t *testing.T) {
	g := newGen(nil)

	typ := &typegraph.Type{
		Name: "Mixed",
		Kind: typegraph.KindEnum,
		EnumValues: []typegraph.EnumValue{
			{Name: "A", Value: "x"},
			{Name: "B", Value: 42},
		},
	}

	out, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	require.NoError(t, err)

	assert.Contains(t, out, "Literal[")
	assert.Contains(t, out, `"x"`)
	assert.Contains(t, out, "42")
}

func TestGenerateFile_SnakeCaseWithAlias(t *testing.T) {
	g := newGen(&Config{SnakeCaseField: true})

	typ := &typegraph.Type{
		Name: "User",
		Kind: typegraph.KindStruct,
		Fields: []*typegraph.Field{
			{
				Name:     "FirstName",
				JSONName: "firstName",
				Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimString},
				Required: true,
			},
		},
	}

	out, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	require.NoError(t, err)

	assert.Contains(t, out, "first_name: str")
	assert.Contains(t, out, `alias="firstName"`)
}

func TestGenerateFile_FieldImportForAliasOnly(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		jsonName string
		wantName string
	}{
		{
			name:     "snake_case alias requires Field import",
			config:   &Config{SnakeCaseField: true},
			jsonName: "firstName",
			wantName: "first_name",
		},
		{
			name:     "numeric prefix sanitization requires Field import",
			config:   &Config{},
			jsonName: "123abc",
			wantName: "field_123abc",
		},
		{
			name:     "python keyword escaping requires Field import",
			config:   &Config{},
			jsonName: "class",
			wantName: "class_",
		},
		{
			name:     "special char sanitization requires Field import",
			config:   &Config{},
			jsonName: "$value",
			wantName: "field__value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newGen(tt.config)

			typ := &typegraph.Type{
				Name: "TestModel",
				Kind: typegraph.KindStruct,
				Fields: []*typegraph.Field{
					{
						JSONName: tt.jsonName,
						Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, Primitive: typegraph.PrimString},
						Required: true,
					},
				},
			}

			out, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
			require.NoError(t, err)

			assert.Contains(t, out, "from pydantic import BaseModel, Field",
				"Field must be imported from pydantic when alias is needed")
			assert.Contains(t, out, tt.wantName+": str = Field(")
			assert.Contains(t, out, fmt.Sprintf("alias=%q", tt.jsonName))
		})
	}
}

func TestBuildFieldParams_NoMinLengthCollision(t *testing.T) {
	g := newGen(nil)
	minLen := 3
	minItems := 1

	f := &typegraph.Field{
		JSONName:  "tags",
		Type:      &typegraph.TypeRef{Kind: typegraph.KindArray},
		Required:  true,
		MinLength: &minLen,
		MinItems:  &minItems,
	}

	params := g.buildFieldParams(f, true, false, "tags")
	combined := strings.Join(params, ", ")
	count := strings.Count(combined, "min_length")
	assert.LessOrEqual(t, count, 1,
		"min_length should not appear twice: %s", combined)
}

func TestGenerateFile_DisableHeaders(t *testing.T) {
	g := newGen(&Config{DisableHeaders: true})

	typ := &typegraph.Type{
		Name: "User",
		Kind: typegraph.KindStruct,
	}

	out, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	require.NoError(t, err)

	assert.NotContains(t, out, "DO NOT EDIT")
	assert.Contains(t, out, "from __future__ import annotations")
}

func TestGenerateFile_AdditionalProperties(t *testing.T) {
	g := newGen(&Config{AllowExtraFields: true})

	typ := &typegraph.Type{
		Name: "Dynamic",
		Kind: typegraph.KindStruct,
		AdditionalProps: &typegraph.AdditionalPropsConfig{
			Allowed: true,
		},
	}

	out, err := g.GenerateFile([]*typegraph.Type{typ}, nil)
	require.NoError(t, err)

	assert.Contains(t, out, "ConfigDict")
	assert.Contains(t, out, "model_config = ConfigDict")
}

func TestGenerateFile_TypeOrdering(t *testing.T) {
	g := newGen(nil)

	a := &typegraph.Type{Name: "A", Kind: typegraph.KindStruct}
	b := &typegraph.Type{Name: "B", Kind: typegraph.KindStruct}

	out, err := g.GenerateFile([]*typegraph.Type{b, a}, nil)
	require.NoError(t, err)

	bIdx := strings.Index(out, "class B")
	aIdx := strings.Index(out, "class A")

	assert.Less(t, bIdx, aIdx, "input order must be preserved")
}
