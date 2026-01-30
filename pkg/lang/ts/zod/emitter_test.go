package zod

import (
	"testing"

	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/stretchr/testify/assert"
)

// Helper functions

func createTestEmitter(cfg *Config) *Emitter {
	return NewEmitter(&typegraph.Graph{Types: []*typegraph.Type{}}, cfg)
}

func createTestType(name string, kind typegraph.TypeKind) *typegraph.Type {
	return &typegraph.Type{Name: name, Kind: kind, Fields: []*typegraph.Field{}}
}

func createTestField(jsonName, goType string, required bool) *typegraph.Field {
	return &typegraph.Field{
		JSONName: jsonName,
		Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: goType},
		Required: required,
	}
}

// Object Schema Generation

func TestEmitter_GenerateObjectSchema(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (*Emitter, *typegraph.Type)
		contains []string
		excludes []string
	}{
		{
			name: "simple object",
			setup: func() (*Emitter, *typegraph.Type) {
				e := createTestEmitter(nil)
				typ := createTestType("User", typegraph.KindStruct)
				typ.Fields = []*typegraph.Field{
					createTestField("id", "string", true),
					createTestField("name", "string", true),
				}
				return e, typ
			},
			contains: []string{"export const UserSchema = z.object({", "id: z.string()", "name: z.string()"},
		},
		{
			name: "with description",
			setup: func() (*Emitter, *typegraph.Type) {
				e := createTestEmitter(nil)
				typ := createTestType("User", typegraph.KindStruct)
				typ.Description = "Represents a user"
				typ.Fields = []*typegraph.Field{createTestField("id", "string", true)}
				return e, typ
			},
			contains: []string{`.meta({ description: "Represents a user" })`},
		},
		{
			name: "strict flag uses strictObject constructor",
			setup: func() (*Emitter, *typegraph.Type) {
				e := createTestEmitter(&Config{Strict: true})
				typ := createTestType("User", typegraph.KindStruct)
				typ.Fields = []*typegraph.Field{createTestField("id", "string", true)}
				return e, typ
			},
			contains: []string{"export const UserSchema = z.strictObject({"},
		},
		{
			name: "optional fields",
			setup: func() (*Emitter, *typegraph.Type) {
				e := createTestEmitter(nil)
				typ := createTestType("User", typegraph.KindStruct)
				typ.Fields = []*typegraph.Field{
					createTestField("id", "string", true),
					createTestField("email", "string", false),
				}
				return e, typ
			},
			contains: []string{"id: z.string()", "email: z.string().optional()"},
		},
		{
			name: "field description",
			setup: func() (*Emitter, *typegraph.Type) {
				e := createTestEmitter(nil)
				typ := createTestType("User", typegraph.KindStruct)
				typ.Fields = []*typegraph.Field{{
					JSONName:    "email",
					Description: "User email address",
					Type:        &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
					Required:    true,
				}}
				return e, typ
			},
			contains: []string{`.meta({ description: "User email address" })`},
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
		fields   []*typegraph.Field
		contains string
	}{
		{"single", []string{"Person"}, []*typegraph.Field{createTestField("employeeId", "string", true)}, "PersonSchema.extend({"},
		{"multiple", []string{"Person", "Timestamped"}, []*typegraph.Field{createTestField("id", "string", true)}, "PersonSchema.merge(TimestampedSchema).extend({"},
		{"no fields", []string{"Person"}, []*typegraph.Field{}, "export const EmployeeSchema = PersonSchema;"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := createTestEmitter(nil)
			typ := createTestType("Employee", typegraph.KindStruct)
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
		values   []typegraph.EnumValue
		contains []string
	}{
		{
			"string enum",
			[]typegraph.EnumValue{{Name: "Active", Value: "active"}, {Name: "Inactive", Value: "inactive"}},
			[]string{`z.enum(["active", "inactive"])`},
		},
		{
			"numeric enum",
			[]typegraph.EnumValue{{Name: "Low", Value: 1}, {Name: "High", Value: 2}},
			[]string{"z.union([z.literal(1), z.literal(2)])"},
		},
		{
			"mixed types",
			[]typegraph.EnumValue{{Name: "String", Value: "text"}, {Name: "Number", Value: 42}, {Name: "Bool", Value: true}},
			[]string{"z.union([", `z.literal("text")`, "z.literal(42)", "z.literal(true)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := createTestEmitter(nil)
			typ := createTestType("TestEnum", typegraph.KindEnum)
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
		name, goType, format, expected string
	}{
		// Formats
		{"email", "string", "email", "z.email()"},
		{"uri", "string", "uri", "z.url()"},
		{"url", "string", "url", "z.url()"},
		{"uuid", "string", "uuid", "z.uuid()"},
		{"ipv4", "string", "ipv4", "z.ipv4()"},
		{"ipv6", "string", "ipv6", "z.ipv6()"},
		{"datetime", "string", "date-time", "z.iso.datetime()"},
		{"date", "string", "date", "z.iso.date()"},
		{"time", "string", "time", "z.iso.time()"},
		// Base types
		{"string", "string", "", "z.string()"},
		{"int", "int", "", "z.int()"},
		{"int32", "int32", "", "z.int()"},
		{"int64", "int64", "", "z.int()"},
		{"float64", "float64", "", "z.number()"},
		{"float32", "float32", "", "z.number()"},
		{"bool", "bool", "", "z.boolean()"},
		{"interface", "interface{}", "", "z.unknown()"},
	}

	e := createTestEmitter(nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, e.primitiveToZod(tt.goType, tt.format, nil))
		})
	}
}

func TestEmitter_PrimitiveToZod_CoerceDates(t *testing.T) {
	e := createTestEmitter(&Config{CoerceDates: true})
	assert.Equal(t, "z.coerce.date()", e.primitiveToZod("string", "date-time", nil))
}

// Constraints

func TestEmitter_Constraints(t *testing.T) {
	e := createTestEmitter(nil)

	t.Run("string min/max/pattern", func(t *testing.T) {
		minLen, maxLen, pattern := 3, 20, "^[a-z]+$"
		field := &typegraph.Field{
			JSONName: "username", Required: true,
			Type:      &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
			MinLength: &minLen, MaxLength: &maxLen, Pattern: &pattern,
		}
		assert.Contains(t, e.generateField(field), "z.string().min(3).max(20).regex(/^[a-z]+$/)")
	})

	t.Run("number min/max", func(t *testing.T) {
		min, max := float64(0), float64(100)
		field := &typegraph.Field{
			JSONName: "age", Required: true,
			Type:    &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "int"},
			Minimum: &min, Maximum: &max,
		}
		assert.Contains(t, e.generateField(field), "z.int().gte(0).lte(100)")
	})

	t.Run("number exclusive bounds", func(t *testing.T) {
		exMin, exMax := float64(0), float64(100)
		field := &typegraph.Field{
			JSONName: "value", Required: true,
			Type:             &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "float64"},
			ExclusiveMinimum: &exMin, ExclusiveMaximum: &exMax,
		}
		assert.Contains(t, e.generateField(field), "z.number().gt(0).lt(100)")
	})

	t.Run("array min/max items", func(t *testing.T) {
		minItems, maxItems := 1, 10
		field := &typegraph.Field{
			JSONName: "items", Required: true,
			Type:     &typegraph.TypeRef{Kind: typegraph.KindArray, ItemType: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"}},
			MinItems: &minItems, MaxItems: &maxItems,
		}
		assert.Contains(t, e.generateField(field), "z.array(z.string()).min(1).max(10)")
	})

	t.Run("regex escaping", func(t *testing.T) {
		pattern := "^[a-z]+/test$"
		field := &typegraph.Field{
			JSONName: "path", Required: true,
			Type:    &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
			Pattern: &pattern,
		}
		assert.Contains(t, e.generateField(field), `regex(/^[a-z]+\/test$/)`)
	})
}

// Type References

func TestEmitter_TypeRefToZod(t *testing.T) {
	tests := []struct {
		name     string
		ref      *typegraph.TypeRef
		expected string
	}{
		{"named ref", &typegraph.TypeRef{Kind: typegraph.KindRef, TypeName: "User"}, "UserSchema"},
		{"array", &typegraph.TypeRef{Kind: typegraph.KindArray, ItemType: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"}}, "z.array(z.string())"},
		{"map", &typegraph.TypeRef{Kind: typegraph.KindMap, ValueType: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "int"}}, "z.record(z.string(), z.int())"},
		{"nullable", &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string", Nullable: true}, "z.string().nullable()"},
		{"inline enum", &typegraph.TypeRef{Kind: typegraph.KindEnum, EnumValues: []interface{}{"a", "b", "c"}}, `z.enum(["a", "b", "c"])`},
	}

	e := createTestEmitter(nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.typeRefToZod(tt.ref, nil)
			assert.Contains(t, result, tt.expected)
		})
	}

	t.Run("union", func(t *testing.T) {
		ref := &typegraph.TypeRef{
			Kind: typegraph.KindUnion,
			UnionMembers: []*typegraph.TypeRef{
				{Kind: typegraph.KindPrimitive, GoType: "string"},
				{Kind: typegraph.KindPrimitive, GoType: "int"},
			},
		}
		assert.Contains(t, e.typeRefToZod(ref, nil), "z.union([z.string(), z.int()])")
	})
}

// Property Name Quoting

func TestEmitter_GenerateField_PropertyNames(t *testing.T) {
	e := createTestEmitter(nil)

	t.Run("needs quoting", func(t *testing.T) {
		field := &typegraph.Field{JSONName: "kebab-case", Type: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"}, Required: true}
		assert.Contains(t, e.generateField(field), `"kebab-case": z.string()`)
	})

	t.Run("valid identifier", func(t *testing.T) {
		field := &typegraph.Field{JSONName: "validName", Type: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"}, Required: true}
		result := e.generateField(field)
		assert.Contains(t, result, "validName: z.string()")
		assert.NotContains(t, result, `"validName"`)
	})
}

// Union, Alias, Primitive Schema Generation

func TestEmitter_GenerateUnionSchema(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("Response", typegraph.KindUnion)
	typ.TargetType = &typegraph.TypeRef{
		Kind: typegraph.KindUnion,
		UnionMembers: []*typegraph.TypeRef{
			{Kind: typegraph.KindRef, TypeName: "Success"},
			{Kind: typegraph.KindRef, TypeName: "Error"},
		},
	}
	assert.Contains(t, e.GenerateSchema(typ), "z.union([SuccessSchema, ErrorSchema])")
}

func TestEmitter_GenerateAliasSchema(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("UserID", typegraph.KindAlias)
	typ.TargetType = &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string", Format: "uuid"}
	assert.Contains(t, e.GenerateSchema(typ), "z.uuid()")
}

func TestEmitter_GeneratePrimitiveSchema(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("Counter", typegraph.KindPrimitive)
	typ.GoType = "int"
	typ.Description = "A counter value"
	result := e.GenerateSchema(typ)
	assert.Contains(t, result, "z.int()")
	assert.Contains(t, result, `.meta({ description: "A counter value" })`)
}

// z.infer Generation

func TestEmitter_GenerateSchemaWithInfer(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = "User account"
	typ.Fields = []*typegraph.Field{createTestField("id", "string", true)}

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
		props    *typegraph.AdditionalPropsConfig
		contains string
		excludes []string
	}{
		{
			name:     "true uses looseObject",
			props:    &typegraph.AdditionalPropsConfig{Allowed: true, Type: nil},
			contains: "z.looseObject({",
		},
		{
			name:     "false uses strictObject",
			props:    &typegraph.AdditionalPropsConfig{Allowed: false},
			contains: "z.strictObject({",
		},
		{
			name:     "typed uses catchall",
			props:    &typegraph.AdditionalPropsConfig{Allowed: true, Type: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "int"}},
			contains: "}).catchall(z.int())",
		},
		{
			name:     "schema overrides strict flag",
			config:   &Config{Strict: true},
			props:    &typegraph.AdditionalPropsConfig{Allowed: true, Type: nil},
			contains: "z.looseObject({",
			excludes: []string{"z.strictObject"},
		},
		{
			name:     "strict flag as fallback",
			config:   &Config{Strict: true},
			props:    nil,
			contains: "z.strictObject({",
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
			typ := createTestType("TestObject", typegraph.KindStruct)
			typ.Fields = []*typegraph.Field{createTestField("name", "string", true)}
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
	fields := []*typegraph.Field{createTestField("nested", "string", true)}

	t.Run("default uses object", func(t *testing.T) {
		e := createTestEmitter(nil)
		result := e.generateInlineObject(fields)
		assert.Contains(t, result, "z.object({")
		assert.NotContains(t, result, "z.strictObject")
	})

	t.Run("strict flag uses strictObject", func(t *testing.T) {
		e := createTestEmitter(&Config{Strict: true})
		result := e.generateInlineObject(fields)
		assert.Contains(t, result, "z.strictObject({")
		assert.NotContains(t, result, ".strict()")
	})
}

// Helper Functions

func TestHelpers_NeedsQuoting(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"validName", false},
		{"123abc", true},
		{"kebab-case", true},
		{"with spaces", true},
		{"_underscore", false},
		{"$dollar", false},
		{"", true},
		{"class", true},
		{"for", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, needsQuoting(tt.input))
		})
	}
}

func TestHelpers_FormatLiteral(t *testing.T) {
	tests := []struct {
		input    interface{}
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

// Integration Test

func TestEmitter_CompleteObjectSchema(t *testing.T) {
	e := createTestEmitter(&Config{Strict: true})
	minAge, maxAge := float64(0), float64(150)
	minLen, maxLen := 3, 50

	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = "User account"
	typ.Fields = []*typegraph.Field{
		{JSONName: "id", Description: "User ID", Type: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string", Format: "uuid"}, Required: true},
		{JSONName: "email", Description: "User email", Type: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string", Format: "email"}, Required: true},
		{JSONName: "username", Type: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"}, MinLength: &minLen, MaxLength: &maxLen, Required: true},
		{JSONName: "age", Type: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "int"}, Minimum: &minAge, Maximum: &maxAge, Required: false},
	}

	result := e.GenerateSchema(typ)
	assert.Contains(t, result, "z.strictObject({")
	assert.Contains(t, result, "id: z.uuid()")
	assert.Contains(t, result, "email: z.email()")
	assert.Contains(t, result, "username: z.string().min(3).max(50)")
	assert.Contains(t, result, "age: z.int().gte(0).lte(150).optional()")
	assert.Contains(t, result, `.meta({ description: "User account" })`)
}
