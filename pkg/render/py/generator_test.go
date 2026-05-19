package py

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mirpo/schemagen/pkg/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newGen(cfg *Config) *Generator {
	if cfg == nil {
		cfg = &Config{}
	}
	return NewGeneratorWithConfig(cfg)
}

func TestGenerateFile_SimpleStruct(t *testing.T) {
	g := newGen(nil)

	typ := &graph.Type{
		Name: "User",
		Kind: graph.KindStruct,
		Fields: []*graph.Field{
			{
				JSONName: "id",
				Type:     &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString},
				Required: true,
			},
		},
	}

	out, err := g.GenerateFile([]*graph.Type{typ}, nil)
	require.NoError(t, err)

	assert.Contains(t, out, "class User(BaseModel):")
	assert.Contains(t, out, "id: str")
	assert.Contains(t, out, "from pydantic import BaseModel")
}

func TestGenerateFile_OptionalField(t *testing.T) {
	g := newGen(nil)

	typ := &graph.Type{
		Name: "User",
		Kind: graph.KindStruct,
		Fields: []*graph.Field{
			{
				JSONName: "email",
				Type:     &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString},
				Required: false,
			},
		},
	}

	out, err := g.GenerateFile([]*graph.Type{typ}, nil)
	require.NoError(t, err)

	assert.Contains(t, out, "email: str | None")
	assert.Contains(t, out, "= None")
}

func TestGenerateFile_StringEnum(t *testing.T) {
	g := newGen(nil)

	typ := &graph.Type{
		Name: "Status",
		Kind: graph.KindEnum,
		EnumValues: []graph.EnumValue{
			{Name: "ACTIVE", Value: "active"},
			{Name: "DISABLED", Value: "disabled"},
		},
	}

	out, err := g.GenerateFile([]*graph.Type{typ}, nil)
	require.NoError(t, err)

	assert.Contains(t, out, "class Status(str, Enum):")
	assert.Contains(t, out, `ACTIVE = "active"`)
	assert.Contains(t, out, "from enum import Enum")
}

func TestGenerateFile_IntEnum(t *testing.T) {
	g := newGen(nil)

	typ := &graph.Type{
		Name: "Priority",
		Kind: graph.KindEnum,
		EnumValues: []graph.EnumValue{
			{Name: "LOW", Value: 1},
			{Name: "HIGH", Value: 2},
		},
	}

	out, err := g.GenerateFile([]*graph.Type{typ}, nil)
	require.NoError(t, err)

	assert.Contains(t, out, "class Priority(IntEnum):")
	assert.Contains(t, out, "LOW = 1")
	assert.Contains(t, out, "from enum import IntEnum")
}

func TestGenerateFile_MixedEnum_UsesLiteral(t *testing.T) {
	g := newGen(nil)

	typ := &graph.Type{
		Name: "Mixed",
		Kind: graph.KindEnum,
		EnumValues: []graph.EnumValue{
			{Name: "A", Value: "x"},
			{Name: "B", Value: 42},
			{Name: "C", Value: true},
			{Name: "D", Value: nil},
		},
	}

	out, err := g.GenerateFile([]*graph.Type{typ}, nil)
	require.NoError(t, err)

	assert.Contains(t, out, "Literal[")
	assert.Contains(t, out, `"x"`)
	assert.Contains(t, out, "42")
	assert.Contains(t, out, "True")
	assert.Contains(t, out, "None")
}

func TestGenerateFile_SnakeCaseWithAlias(t *testing.T) {
	g := newGen(&Config{SnakeCaseField: true})

	typ := &graph.Type{
		Name: "User",
		Kind: graph.KindStruct,
		Fields: []*graph.Field{
			{
				JSONName: "firstName",
				Type:     &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString},
				Required: true,
			},
		},
	}

	out, err := g.GenerateFile([]*graph.Type{typ}, nil)
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

			typ := &graph.Type{
				Name: "TestModel",
				Kind: graph.KindStruct,
				Fields: []*graph.Field{
					{
						JSONName: tt.jsonName,
						Type:     &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString},
						Required: true,
					},
				},
			}

			out, err := g.GenerateFile([]*graph.Type{typ}, nil)
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

	f := &graph.Field{
		JSONName:  "tags",
		Type:      &graph.TypeRef{Kind: graph.KindArray},
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

	typ := &graph.Type{
		Name: "User",
		Kind: graph.KindStruct,
	}

	out, err := g.GenerateFile([]*graph.Type{typ}, nil)
	require.NoError(t, err)

	assert.NotContains(t, out, "DO NOT EDIT")
	assert.Contains(t, out, "from __future__ import annotations")
}

func TestGenerateFile_AdditionalProperties(t *testing.T) {
	g := newGen(&Config{AllowExtraFields: true})

	typ := &graph.Type{
		Name: "Dynamic",
		Kind: graph.KindStruct,
		AdditionalProps: &graph.AdditionalPropsConfig{
			Allowed: true,
		},
	}

	out, err := g.GenerateFile([]*graph.Type{typ}, nil)
	require.NoError(t, err)

	assert.Contains(t, out, "ConfigDict")
	assert.Contains(t, out, "model_config = ConfigDict")
}

func TestGenerateFile_TypeOrdering(t *testing.T) {
	g := newGen(nil)

	a := &graph.Type{Name: "A", Kind: graph.KindStruct}
	b := &graph.Type{Name: "B", Kind: graph.KindStruct}

	out, err := g.GenerateFile([]*graph.Type{b, a}, nil)
	require.NoError(t, err)

	bIdx := strings.Index(out, "class B")
	aIdx := strings.Index(out, "class A")

	assert.Less(t, bIdx, aIdx, "input order must be preserved")
}

func TestGenerateFile_ImportWriteDoesNotMutateInput(t *testing.T) {
	g := newGen(&Config{DisableHeaders: true})

	imports := []graph.ImportSpec{
		{ImportPath: ".models", TypeNames: []string{"Zebra", "Apple", "Mango"}},
	}

	original := make([]string, len(imports[0].TypeNames))
	copy(original, imports[0].TypeNames)

	_, err := g.GenerateFile([]*graph.Type{
		{Name: "Test", Kind: graph.KindStruct},
	}, imports)
	require.NoError(t, err)

	assert.Equal(t, original, imports[0].TypeNames,
		"GenerateFile must not mutate the caller's TypeNames slice")
}

func TestGenerateFile_ImportsFromScanPhaseOnly(t *testing.T) {
	tests := []struct {
		name       string
		types      []*graph.Type
		wantImport string
	}{
		{
			name: "primitive alias UUID gets uuid import",
			types: []*graph.Type{
				{Name: "MyUUID", Kind: graph.KindPrimitive, Primitive: graph.PrimUUID},
			},
			wantImport: "from uuid import UUID",
		},
		{
			name: "primitive alias datetime gets datetime import",
			types: []*graph.Type{
				{Name: "MyDate", Kind: graph.KindPrimitive, Primitive: graph.PrimDateTime},
			},
			wantImport: "from datetime import datetime",
		},
		{
			name: "primitive alias email gets pydantic EmailStr import",
			types: []*graph.Type{
				{Name: "MyEmail", Kind: graph.KindPrimitive, Primitive: graph.PrimEmail},
			},
			wantImport: "EmailStr",
		},
		{
			name: "primitive alias URI gets pydantic AnyUrl import",
			types: []*graph.Type{
				{Name: "MyURL", Kind: graph.KindPrimitive, Primitive: graph.PrimURI},
			},
			wantImport: "AnyUrl",
		},
		{
			name: "struct with UUID field gets uuid import",
			types: []*graph.Type{
				{
					Name: "User",
					Kind: graph.KindStruct,
					Fields: []*graph.Field{
						{
							JSONName: "id",
							Type:     &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimUUID},
							Required: true,
						},
					},
				},
			},
			wantImport: "from uuid import UUID",
		},
		{
			name: "union type gets Any import",
			types: []*graph.Type{
				{Name: "MyUnion", Kind: graph.KindUnion},
			},
			wantImport: "from typing import Any",
		},
		{
			name: "string enum gets Enum import",
			types: []*graph.Type{
				{
					Name: "Status",
					Kind: graph.KindEnum,
					EnumValues: []graph.EnumValue{
						{Name: "ACTIVE", Value: "active"},
					},
				},
			},
			wantImport: "from enum import Enum",
		},
		{
			name: "int enum gets IntEnum import",
			types: []*graph.Type{
				{
					Name: "Priority",
					Kind: graph.KindEnum,
					EnumValues: []graph.EnumValue{
						{Name: "LOW", Value: 1},
					},
				},
			},
			wantImport: "from enum import IntEnum",
		},
		{
			name: "mixed enum gets Literal import",
			types: []*graph.Type{
				{
					Name: "Mixed",
					Kind: graph.KindEnum,
					EnumValues: []graph.EnumValue{
						{Name: "A", Value: "x"},
						{Name: "B", Value: 42},
						{Name: "C", Value: true},
					},
				},
			},
			wantImport: "from typing import Literal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newGen(&Config{DisableHeaders: true})
			out, err := g.GenerateFile(tt.types, nil)
			require.NoError(t, err)
			assert.Contains(t, out, tt.wantImport)
		})
	}
}
