package zod

import (
	"testing"

	"github.com/mirpo/schemagen/pkg/v2/graph"
	"github.com/stretchr/testify/assert"
)

// Helper functions

func createTestEmitter(cfg *Config) *Emitter {
	return NewEmitter(cfg)
}

func createTestType(name string, kind graph.TypeKind) *graph.Type {
	return &graph.Type{Name: name, Kind: kind, Fields: []*graph.Field{}}
}

func createTestField(jsonName string, prim graph.PrimitiveKind, required bool) *graph.Field {
	return &graph.Field{
		JSONName: jsonName,
		Type:     &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: prim},
		Required: required,
	}
}

// Object Schema Generation

func TestEmitter_GenerateObjectSchema(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (*Emitter, *graph.Type)
		contains []string
		excludes []string
	}{
		{
			name: "simple object",
			setup: func() (*Emitter, *graph.Type) {
				e := createTestEmitter(nil)
				typ := createTestType("User", graph.KindStruct)
				typ.Fields = []*graph.Field{
					createTestField("id", graph.PrimString, true),
					createTestField("name", graph.PrimString, true),
				}
				return e, typ
			},
			contains: []string{"export const UserSchema = z.object({", "id: z.string()", "name: z.string()"},
		},
		{
			name: "with description",
			setup: func() (*Emitter, *graph.Type) {
				e := createTestEmitter(nil)
				typ := createTestType("User", graph.KindStruct)
				typ.Description = "Represents a user"
				typ.Fields = []*graph.Field{createTestField("id", graph.PrimString, true)}
				return e, typ
			},
			contains: []string{`.meta({ description: "Represents a user" })`},
		},
		{
			name: "strict flag uses strictObject constructor",
			setup: func() (*Emitter, *graph.Type) {
				e := createTestEmitter(&Config{Strict: true})
				typ := createTestType("User", graph.KindStruct)
				typ.Fields = []*graph.Field{createTestField("id", graph.PrimString, true)}
				return e, typ
			},
			contains: []string{"export const UserSchema = z.strictObject({"},
		},
		{
			name: "optional fields",
			setup: func() (*Emitter, *graph.Type) {
				e := createTestEmitter(nil)
				typ := createTestType("User", graph.KindStruct)
				typ.Fields = []*graph.Field{
					createTestField("id", graph.PrimString, true),
					createTestField("email", graph.PrimString, false),
				}
				return e, typ
			},
			contains: []string{"id: z.string()", "email: z.string().optional()"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, typ := tt.setup()
			result := e.GenerateSchema(typ)
			for _, c := range tt.contains {
				assert.Contains(t, result, c)
			}
			for _, x := range tt.excludes {
				assert.NotContains(t, result, x)
			}
		})
	}
}

// Inheritance (allOf/extends)

func TestEmitter_GenerateObjectSchema_Extends(t *testing.T) {
	tests := []struct {
		name     string
		extends  []string
		fields   []*graph.Field
		contains string
	}{
		{"single", []string{"Person"}, []*graph.Field{createTestField("employeeId", graph.PrimString, true)}, "PersonSchema.extend({"},
		{"multiple", []string{"Person", "Timestamped"}, []*graph.Field{createTestField("id", graph.PrimString, true)}, "PersonSchema.merge(TimestampedSchema).extend({"},
		{"no fields", []string{"Person"}, []*graph.Field{}, "export const EmployeeSchema = PersonSchema;"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := createTestEmitter(nil)
			typ := createTestType("Employee", graph.KindStruct)
			typ.Extends = tt.extends
			typ.Fields = tt.fields
			result := e.GenerateSchema(typ)
			assert.Contains(t, result, tt.contains)
		})
	}
}

// Enum Schema Generation

func TestEmitter_GenerateEnumSchema(t *testing.T) {
	tests := []struct {
		name     string
		values   []graph.EnumValue
		contains []string
	}{
		{
			"string enum",
			[]graph.EnumValue{{Name: "Active", Value: "active"}, {Name: "Inactive", Value: "inactive"}},
			[]string{`z.enum(["active", "inactive"])`},
		},
		{
			"numeric enum",
			[]graph.EnumValue{{Name: "Low", Value: 1}, {Name: "High", Value: 2}},
			[]string{"z.union([\n", "  z.literal(1),\n", "  z.literal(2),\n"},
		},
		{
			"with object value",
			[]graph.EnumValue{{Name: "Simple", Value: "simple"}, {Name: "Complex", Value: map[string]any{"complex": true}}},
			[]string{"z.union([\n", "  z.literal(\"simple\"),\n", "  z.object({ complex: z.literal(true) }).strict(),\n"},
		},
		{
			"with array value",
			[]graph.EnumValue{{Name: "Simple", Value: "simple"}, {Name: "Array", Value: []any{"a", "b"}}},
			[]string{"z.union([\n", "  z.literal(\"simple\"),\n", `  z.tuple([z.literal("a"), z.literal("b")]),` + "\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := createTestEmitter(nil)
			typ := createTestType("TestEnum", graph.KindEnum)
			typ.EnumValues = tt.values
			result := e.GenerateSchema(typ)
			for _, c := range tt.contains {
				assert.Contains(t, result, c)
			}
		})
	}
}

// Primitive Types and Formats

func TestEmitter_PrimitiveToZod(t *testing.T) {
	tests := []struct {
		name     string
		prim     graph.PrimitiveKind
		expected string
	}{
		{"string", graph.PrimString, "z.string()"},
		{"int", graph.PrimInt, "z.int()"},
		{"int32", graph.PrimInt32, "z.int()"},
		{"int64", graph.PrimInt64, "z.int()"},
		{"float64", graph.PrimFloat64, "z.number()"},
		{"float32", graph.PrimFloat32, "z.number()"},
		{"bool", graph.PrimBool, "z.boolean()"},
		{"unknown", graph.PrimUnknown, "z.unknown()"},
		{"email", graph.PrimEmail, "z.email()"},
		{"uri", graph.PrimURI, "z.url()"},
		{"uuid", graph.PrimUUID, "z.uuid()"},
		{"ipv4", graph.PrimIPv4, "z.ipv4()"},
		{"ipv6", graph.PrimIPv6, "z.ipv6()"},
		{"datetime", graph.PrimDateTime, "z.iso.datetime()"},
		{"date", graph.PrimDate, "z.iso.date()"},
		{"time", graph.PrimTime, "z.iso.time()"},
		{"hostname", graph.PrimHostname, "z.string()"},
	}

	e := createTestEmitter(nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, e.primitiveToZod(tt.prim, nil))
		})
	}
}

func TestEmitter_PrimitiveToZod_CoerceDates(t *testing.T) {
	e := createTestEmitter(&Config{CoerceDates: true})
	assert.Equal(t, "z.coerce.date()", e.primitiveToZod(graph.PrimDateTime, nil))
}

// Constraints

func TestEmitter_Constraints(t *testing.T) {
	e := createTestEmitter(nil)

	t.Run("string min/max/pattern", func(t *testing.T) {
		minLen, maxLen, pattern := 3, 20, "^[a-z]+$"
		field := &graph.Field{
			JSONName: "username", Required: true,
			Type:      &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString},
			MinLength: &minLen, MaxLength: &maxLen, Pattern: &pattern,
		}
		assert.Contains(t, e.generateField(field), "z.string().min(3).max(20).regex(/^[a-z]+$/)")
	})

	t.Run("number min/max", func(t *testing.T) {
		min, max := float64(0), float64(100)
		field := &graph.Field{
			JSONName: "age", Required: true,
			Type:    &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimInt},
			Minimum: &min, Maximum: &max,
		}
		assert.Contains(t, e.generateField(field), "z.int().gte(0).lte(100)")
	})

	t.Run("number exclusive bounds", func(t *testing.T) {
		exMin, exMax := float64(0), float64(100)
		field := &graph.Field{
			JSONName: "value", Required: true,
			Type:             &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimFloat64},
			ExclusiveMinimum: &exMin, ExclusiveMaximum: &exMax,
		}
		assert.Contains(t, e.generateField(field), "z.number().gt(0).lt(100)")
	})

	t.Run("array min/max items", func(t *testing.T) {
		minItems, maxItems := 1, 10
		field := &graph.Field{
			JSONName: "items", Required: true,
			Type:     &graph.TypeRef{Kind: graph.KindArray, ItemType: &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString}},
			MinItems: &minItems, MaxItems: &maxItems,
		}
		assert.Contains(t, e.generateField(field), "z.array(z.string()).min(1).max(10)")
	})

	t.Run("regex escaping", func(t *testing.T) {
		pattern := "^[a-z]+/test$"
		field := &graph.Field{
			JSONName: "path", Required: true,
			Type:    &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString},
			Pattern: &pattern,
		}
		assert.Contains(t, e.generateField(field), `regex(/^[a-z]+\/test$/)`)
	})
}

// Type References

func TestEmitter_TypeRefToZod(t *testing.T) {
	tests := []struct {
		name     string
		ref      *graph.TypeRef
		expected string
	}{
		{"named ref", &graph.TypeRef{Kind: graph.KindRef, TypeName: "User"}, "UserSchema"},
		{"array", &graph.TypeRef{Kind: graph.KindArray, ItemType: &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString}}, "z.array(z.string())"},
		{"map", &graph.TypeRef{Kind: graph.KindMap, ValueType: &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimInt}}, "z.record(z.string(), z.int())"},
		{"nullable", &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString, Nullable: true}, "z.string().nullable()"},
		{"inline enum", &graph.TypeRef{Kind: graph.KindEnum, EnumValues: []graph.EnumValue{{Value: "a"}, {Value: "b"}, {Value: "c"}}}, `z.enum(["a", "b", "c"])`},
	}

	e := createTestEmitter(nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.typeRefToZod(tt.ref, nil)
			assert.Contains(t, result, tt.expected)
		})
	}

	t.Run("union", func(t *testing.T) {
		ref := &graph.TypeRef{
			Kind: graph.KindUnion,
			UnionMembers: []*graph.TypeRef{
				{Kind: graph.KindPrimitive, Primitive: graph.PrimString},
				{Kind: graph.KindPrimitive, Primitive: graph.PrimInt},
			},
		}
		assert.Contains(t, e.typeRefToZod(ref, nil), "z.union([z.string(), z.int()])")
	})
}

// Property Name Quoting

func TestEmitter_GenerateField_PropertyNames(t *testing.T) {
	e := createTestEmitter(nil)

	t.Run("needs quoting", func(t *testing.T) {
		field := &graph.Field{JSONName: "kebab-case", Type: &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString}, Required: true}
		assert.Contains(t, e.generateField(field), `"kebab-case": z.string()`)
	})

	t.Run("valid identifier", func(t *testing.T) {
		field := &graph.Field{JSONName: "validName", Type: &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimString}, Required: true}
		result := e.generateField(field)
		assert.Contains(t, result, "validName: z.string()")
		assert.NotContains(t, result, `"validName"`)
	})
}

// Union and Primitive Schema Generation

func TestEmitter_GenerateUnionSchema(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("Response", graph.KindUnion)
	typ.UnionMembers = []*graph.TypeRef{
		{Kind: graph.KindRef, TypeName: "Success"},
		{Kind: graph.KindRef, TypeName: "Error"},
	}
	assert.Contains(t, e.GenerateSchema(typ), "z.union([SuccessSchema, ErrorSchema])")
}

func TestEmitter_GeneratePrimitiveSchema(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("Counter", graph.KindPrimitive)
	typ.Primitive = graph.PrimInt
	typ.Description = "A counter value"
	result := e.GenerateSchema(typ)
	assert.Contains(t, result, "z.int()")
	assert.Contains(t, result, `.meta({ description: "A counter value" })`)
}

// z.infer Generation

func TestEmitter_GenerateSchemaWithInfer(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("User", graph.KindStruct)
	typ.Description = "User account"
	typ.Fields = []*graph.Field{createTestField("id", graph.PrimString, true)}

	result := e.GenerateSchemaWithInfer(typ)
	assert.Contains(t, result, "/**")
	assert.Contains(t, result, "* User account")
	assert.Contains(t, result, "export type User = z.infer<typeof UserSchema>;")
}

// Additional Properties

func TestEmitter_AdditionalProperties(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		props    *graph.AdditionalPropsConfig
		contains string
		excludes []string
	}{
		{
			name:     "true uses looseObject",
			props:    &graph.AdditionalPropsConfig{Allowed: true, Type: nil},
			contains: "z.looseObject({",
		},
		{
			name:     "false uses strictObject",
			props:    &graph.AdditionalPropsConfig{Allowed: false},
			contains: "z.strictObject({",
		},
		{
			name:     "typed uses catchall",
			props:    &graph.AdditionalPropsConfig{Allowed: true, Type: &graph.TypeRef{Kind: graph.KindPrimitive, Primitive: graph.PrimInt}},
			contains: "}).catchall(z.int())",
		},
		{
			name:     "schema overrides strict flag",
			config:   &Config{Strict: true},
			props:    &graph.AdditionalPropsConfig{Allowed: true, Type: nil},
			contains: "z.looseObject({",
			excludes: []string{"z.strictObject"},
		},
		{
			name:     "default uses object",
			props:    nil,
			contains: "z.object({",
			excludes: []string{"z.strictObject", "z.looseObject"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := createTestEmitter(tt.config)
			typ := createTestType("TestObject", graph.KindStruct)
			typ.Fields = []*graph.Field{createTestField("name", graph.PrimString, true)}
			typ.AdditionalProps = tt.props

			result := e.GenerateSchema(typ)
			assert.Contains(t, result, tt.contains)
			for _, x := range tt.excludes {
				assert.NotContains(t, result, x)
			}
		})
	}
}

// Inline Objects

func TestEmitter_GenerateInlineObject(t *testing.T) {
	e := createTestEmitter(nil)
	fields := []*graph.Field{createTestField("nested", graph.PrimString, true)}
	result := e.generateInlineObject(fields)
	assert.Contains(t, result, "z.object({")
	assert.NotContains(t, result, "z.strictObject")
}

// Helper Functions

func TestHelpers_FormatLiteral(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{"test", `"test"`},
		{42, "42"},
		{float64(42), "42"},
		{3.14, "3.14"},
		{true, "true"},
		{false, "false"},
		{nil, "null"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, formatLiteral(tt.input))
	}
}

func TestHelpers_FormatZodLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "hello", `z.literal("hello")`},
		{"number", 42, "z.literal(42)"},
		{"float", 3.14, "z.literal(3.14)"},
		{"bool true", true, "z.literal(true)"},
		{"bool false", false, "z.literal(false)"},
		{"null", nil, "z.literal(null)"},

		{"simple object", map[string]any{"complex": true}, `z.object({ complex: z.literal(true) }).strict()`},
		{"object with string", map[string]any{"name": "test"}, `z.object({ name: z.literal("test") }).strict()`},
		{"object with number", map[string]any{"count": float64(42)}, `z.object({ count: z.literal(42) }).strict()`},

		{"simple array", []any{"a", "b"}, `z.tuple([z.literal("a"), z.literal("b")])`},
		{"mixed array", []any{"text", float64(42)}, `z.tuple([z.literal("text"), z.literal(42)])`},

		{"nested object", map[string]any{"nested": map[string]any{"deep": float64(1)}}, `z.object({ nested: z.object({ deep: z.literal(1) }).strict() }).strict()`},
		{"nested array", []any{[]any{"inner"}}, `z.tuple([z.tuple([z.literal("inner")])])`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatZodLiteral(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHelpers_FormatNumber(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{42.0, "42"},
		{3.14, "3.14"},
		{0.0, "0"},
		{-10.0, "-10"},
		{-3.14, "-3.14"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, formatNumber(tt.input))
	}
}
