package zod

import (
	"testing"

	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/stretchr/testify/assert"
)

// Helper functions

func createTestEmitter(cfg *Config) *Emitter {
	graph := &typegraph.Graph{
		Types: []*typegraph.Type{},
	}
	return NewEmitter(graph, cfg)
}

func createTestType(name string, kind typegraph.TypeKind) *typegraph.Type {
	return &typegraph.Type{
		Name:   name,
		Kind:   kind,
		Fields: []*typegraph.Field{},
	}
}

func createTestField(jsonName string, goType string, required bool) *typegraph.Field {
	return &typegraph.Field{
		JSONName: jsonName,
		Type: &typegraph.TypeRef{
			Kind:   typegraph.KindPrimitive,
			GoType: goType,
		},
		Required: required,
	}
}

// Basic Object Schema Generation

func TestEmitter_GenerateObjectSchema_Simple(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("id", "string", true),
		createTestField("name", "string", true),
	}

	result := e.GenerateSchema(typ)
	assert.Contains(t, result, "export const UserSchema = z.object({")
	assert.Contains(t, result, "id: z.string()")
	assert.Contains(t, result, "name: z.string()")
	assert.Contains(t, result, "});")
}

func TestEmitter_GenerateObjectSchema_WithDescription(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = "Represents a user"
	typ.Fields = []*typegraph.Field{
		createTestField("id", "string", true),
	}

	result := e.GenerateSchema(typ)
	assert.Contains(t, result, `.meta({ description: "Represents a user" })`)
}

func TestEmitter_GenerateObjectSchema_Strict(t *testing.T) {
	e := createTestEmitter(&Config{Strict: true})
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("id", "string", true),
	}

	result := e.GenerateSchema(typ)
	assert.Contains(t, result, "}).strict()")
}

func TestEmitter_GenerateObjectSchema_WithInfer(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = "User type"
	typ.Fields = []*typegraph.Field{
		createTestField("id", "string", true),
	}

	result := e.GenerateSchemaWithInfer(typ)
	// JSDoc comment should appear for zod-only mode
	assert.Contains(t, result, "/**")
	assert.Contains(t, result, "* User type")
	assert.Contains(t, result, "*/")
	// z.infer type
	assert.Contains(t, result, "export type User = z.infer<typeof UserSchema>;")
}

func TestEmitter_GenerateObjectSchema_OptionalFields(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		createTestField("id", "string", true),
		createTestField("email", "string", false),
	}

	result := e.GenerateSchema(typ)
	assert.Contains(t, result, "id: z.string()")
	assert.Contains(t, result, "email: z.string().optional()")
}

func TestEmitter_GenerateObjectSchema_FieldDescription(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("User", typegraph.KindStruct)
	typ.Fields = []*typegraph.Field{
		{
			JSONName:    "email",
			Description: "User email address",
			Type:        &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
			Required:    true,
		},
	}

	result := e.GenerateSchema(typ)
	assert.Contains(t, result, `.meta({ description: "User email address" })`)
}

// AllOf / Inheritance

func TestEmitter_GenerateObjectSchema_SingleExtend(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("Employee", typegraph.KindStruct)
	typ.Extends = []string{"Person"}
	typ.Fields = []*typegraph.Field{
		createTestField("employeeId", "string", true),
	}

	result := e.GenerateSchema(typ)
	assert.Contains(t, result, "export const EmployeeSchema = PersonSchema.extend({")
	assert.Contains(t, result, "employeeId: z.string()")
}

func TestEmitter_GenerateObjectSchema_MultipleExtends(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("Employee", typegraph.KindStruct)
	typ.Extends = []string{"Person", "Timestamped"}
	typ.Fields = []*typegraph.Field{
		createTestField("employeeId", "string", true),
	}

	result := e.GenerateSchema(typ)
	assert.Contains(t, result, "PersonSchema.merge(TimestampedSchema).extend({")
}

func TestEmitter_GenerateObjectSchema_ExtendsNoFields(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("Employee", typegraph.KindStruct)
	typ.Extends = []string{"Person"}
	typ.Fields = []*typegraph.Field{}

	result := e.GenerateSchema(typ)
	assert.Contains(t, result, "export const EmployeeSchema = PersonSchema;")
}

// Enum Schema Generation

func TestEmitter_GenerateEnumSchema_StringEnum(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("Status", typegraph.KindEnum)
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "Active", Value: "active"},
		{Name: "Inactive", Value: "inactive"},
	}

	result := e.GenerateSchema(typ)
	assert.Contains(t, result, `export const StatusSchema = z.enum(["active", "inactive"])`)
}

func TestEmitter_GenerateEnumSchema_NumericEnum(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("Priority", typegraph.KindEnum)
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "Low", Value: 1},
		{Name: "High", Value: 2},
	}

	result := e.GenerateSchema(typ)
	assert.Contains(t, result, "z.union([z.literal(1), z.literal(2)])")
}

func TestEmitter_GenerateEnumSchema_MixedTypes(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("Mixed", typegraph.KindEnum)
	typ.EnumValues = []typegraph.EnumValue{
		{Name: "String", Value: "text"},
		{Name: "Number", Value: 42},
		{Name: "Bool", Value: true},
	}

	result := e.GenerateSchema(typ)
	assert.Contains(t, result, "z.union([")
	assert.Contains(t, result, `z.literal("text")`)
	assert.Contains(t, result, "z.literal(42)")
	assert.Contains(t, result, "z.literal(true)")
}

// Zod v4 Top-Level Format Schemas

func TestEmitter_PrimitiveToZod_Formats(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		goType   string
		expected string
	}{
		{"email", "email", "string", "z.email()"},
		{"uri", "uri", "string", "z.url()"},
		{"url", "url", "string", "z.url()"},
		{"uuid", "uuid", "string", "z.uuid()"},
		{"ipv4", "ipv4", "string", "z.ipv4()"},
		{"ipv6", "ipv6", "string", "z.ipv6()"},
		{"datetime", "date-time", "string", "z.iso.datetime()"},
		{"date", "date", "string", "z.iso.date()"},
		{"time", "time", "string", "z.iso.time()"},
	}

	e := createTestEmitter(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.primitiveToZod(tt.goType, tt.format, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEmitter_PrimitiveToZod_CoerceDates(t *testing.T) {
	e := createTestEmitter(&Config{CoerceDates: true})
	result := e.primitiveToZod("string", "date-time", nil)
	assert.Equal(t, "z.coerce.date()", result)
}

func TestEmitter_PrimitiveToZod_BaseTypes(t *testing.T) {
	tests := []struct {
		name     string
		goType   string
		expected string
	}{
		{"string", "string", "z.string()"},
		{"int", "int", "z.int()"},
		{"int32", "int32", "z.int()"},
		{"int64", "int64", "z.int()"},
		{"float64", "float64", "z.number()"},
		{"float32", "float32", "z.number()"},
		{"bool", "bool", "z.boolean()"},
		{"interface", "interface{}", "z.unknown()"},
	}

	e := createTestEmitter(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.primitiveToZod(tt.goType, "", nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// String Constraints

func TestEmitter_StringConstraints(t *testing.T) {
	e := createTestEmitter(nil)

	minLen := 3
	maxLen := 20
	pattern := "^[a-z]+$"

	field := &typegraph.Field{
		JSONName:  "username",
		Type:      &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
		MinLength: &minLen,
		MaxLength: &maxLen,
		Pattern:   &pattern,
		Required:  true,
	}

	result := e.generateField(field)
	assert.Contains(t, result, "z.string().min(3).max(20).regex(/^[a-z]+$/)")
}

func TestEmitter_StringConstraints_MinOnly(t *testing.T) {
	e := createTestEmitter(nil)

	minLen := 1

	field := &typegraph.Field{
		JSONName:  "name",
		Type:      &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
		MinLength: &minLen,
		Required:  true,
	}

	result := e.generateField(field)
	assert.Contains(t, result, "z.string().min(1)")
}

// Number Constraints

func TestEmitter_NumberConstraints(t *testing.T) {
	e := createTestEmitter(nil)

	min := float64(0)
	max := float64(100)

	field := &typegraph.Field{
		JSONName: "age",
		Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "int"},
		Minimum:  &min,
		Maximum:  &max,
		Required: true,
	}

	result := e.generateField(field)
	assert.Contains(t, result, "z.int().gte(0).lte(100)")
}

func TestEmitter_NumberConstraints_Exclusive(t *testing.T) {
	e := createTestEmitter(nil)

	exMin := float64(0)
	exMax := float64(100)

	field := &typegraph.Field{
		JSONName:         "value",
		Type:             &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "float64"},
		ExclusiveMinimum: &exMin,
		ExclusiveMaximum: &exMax,
		Required:         true,
	}

	result := e.generateField(field)
	assert.Contains(t, result, "z.number().gt(0).lt(100)")
}

// Array Constraints

func TestEmitter_ArrayConstraints(t *testing.T) {
	e := createTestEmitter(nil)

	minItems := 1
	maxItems := 10

	field := &typegraph.Field{
		JSONName: "items",
		Type: &typegraph.TypeRef{
			Kind:     typegraph.KindArray,
			ItemType: &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
		},
		MinItems: &minItems,
		MaxItems: &maxItems,
		Required: true,
	}

	result := e.generateField(field)
	assert.Contains(t, result, "z.array(z.string()).min(1).max(10)")
}

// Type References

func TestEmitter_TypeRefToZod_NamedRef(t *testing.T) {
	e := createTestEmitter(nil)
	ref := &typegraph.TypeRef{
		Kind:     typegraph.KindRef,
		TypeName: "User",
	}

	result := e.typeRefToZod(ref, nil)
	assert.Equal(t, "UserSchema", result)
}

func TestEmitter_TypeRefToZod_Array(t *testing.T) {
	e := createTestEmitter(nil)
	ref := &typegraph.TypeRef{
		Kind: typegraph.KindArray,
		ItemType: &typegraph.TypeRef{
			Kind:   typegraph.KindPrimitive,
			GoType: "string",
		},
	}

	result := e.typeRefToZod(ref, nil)
	assert.Equal(t, "z.array(z.string())", result)
}

func TestEmitter_TypeRefToZod_Map(t *testing.T) {
	e := createTestEmitter(nil)
	ref := &typegraph.TypeRef{
		Kind: typegraph.KindMap,
		ValueType: &typegraph.TypeRef{
			Kind:   typegraph.KindPrimitive,
			GoType: "int",
		},
	}

	result := e.typeRefToZod(ref, nil)
	assert.Equal(t, "z.record(z.string(), z.int())", result)
}

func TestEmitter_TypeRefToZod_Union(t *testing.T) {
	e := createTestEmitter(nil)
	ref := &typegraph.TypeRef{
		Kind: typegraph.KindUnion,
		UnionMembers: []*typegraph.TypeRef{
			{Kind: typegraph.KindPrimitive, GoType: "string"},
			{Kind: typegraph.KindPrimitive, GoType: "int"},
		},
	}

	result := e.typeRefToZod(ref, nil)
	assert.Contains(t, result, "z.union([z.string(), z.int()])")
}

func TestEmitter_TypeRefToZod_Nullable(t *testing.T) {
	e := createTestEmitter(nil)
	ref := &typegraph.TypeRef{
		Kind:     typegraph.KindPrimitive,
		GoType:   "string",
		Nullable: true,
	}

	result := e.typeRefToZod(ref, nil)
	assert.Equal(t, "z.string().nullable()", result)
}

func TestEmitter_TypeRefToZod_InlineEnum(t *testing.T) {
	e := createTestEmitter(nil)
	ref := &typegraph.TypeRef{
		Kind:       typegraph.KindEnum,
		EnumValues: []interface{}{"a", "b", "c"},
	}

	result := e.typeRefToZod(ref, nil)
	assert.Contains(t, result, `z.enum(["a", "b", "c"])`)
}

// Property Name Quoting

func TestEmitter_GenerateField_QuotedPropertyName(t *testing.T) {
	e := createTestEmitter(nil)

	field := &typegraph.Field{
		JSONName: "kebab-case",
		Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
		Required: true,
	}

	result := e.generateField(field)
	assert.Contains(t, result, `"kebab-case": z.string()`)
}

func TestEmitter_GenerateField_ValidIdentifier(t *testing.T) {
	e := createTestEmitter(nil)

	field := &typegraph.Field{
		JSONName: "validName",
		Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
		Required: true,
	}

	result := e.generateField(field)
	assert.Contains(t, result, "validName: z.string()")
	assert.NotContains(t, result, `"validName"`)
}

// Union Schema Generation

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

	result := e.GenerateSchema(typ)
	assert.Contains(t, result, "export const ResponseSchema = z.union([SuccessSchema, ErrorSchema])")
}

// Alias Schema Generation

func TestEmitter_GenerateAliasSchema(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("UserID", typegraph.KindAlias)
	typ.TargetType = &typegraph.TypeRef{
		Kind:   typegraph.KindPrimitive,
		GoType: "string",
		Format: "uuid",
	}

	result := e.GenerateSchema(typ)
	assert.Contains(t, result, "export const UserIDSchema = z.uuid()")
}

// Primitive Schema Generation

func TestEmitter_GeneratePrimitiveSchema(t *testing.T) {
	e := createTestEmitter(nil)
	typ := createTestType("Counter", typegraph.KindPrimitive)
	typ.GoType = "int"
	typ.Description = "A counter value"

	result := e.GenerateSchema(typ)
	assert.Contains(t, result, "export const CounterSchema = z.int()")
	assert.Contains(t, result, `.meta({ description: "A counter value" })`)
}

// Regex Pattern Escaping

func TestEmitter_EscapeRegex(t *testing.T) {
	e := createTestEmitter(nil)

	pattern := "^[a-z]+/test$"

	field := &typegraph.Field{
		JSONName: "path",
		Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
		Pattern:  &pattern,
		Required: true,
	}

	result := e.generateField(field)
	// Forward slashes should be escaped
	assert.Contains(t, result, `regex(/^[a-z]+\/test$/)`)
}

// Helpers

func TestHelpers_NeedsQuoting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid identifier", "validName", false},
		{"starts with number", "123abc", true},
		{"contains hyphen", "kebab-case", true},
		{"contains space", "with spaces", true},
		{"underscore valid", "_underscore", false},
		{"dollar valid", "$dollar", false},
		{"empty string", "", true},
		{"reserved word", "class", true},
		{"reserved word for", "for", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needsQuoting(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHelpers_FormatLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string", "test", `"test"`},
		{"int", 42, "42"},
		{"float whole", float64(42), "42"},
		{"float decimal", 3.14, "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"nil", nil, "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLiteral(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHelpers_FormatNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected string
	}{
		{"whole number", 42.0, "42"},
		{"decimal", 3.14, "3.14"},
		{"zero", 0.0, "0"},
		{"negative whole", -10.0, "-10"},
		{"negative decimal", -3.14, "-3.14"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatNumber(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration Tests

func TestEmitter_CompleteObjectSchema(t *testing.T) {
	e := createTestEmitter(&Config{Strict: true})

	minAge := float64(0)
	maxAge := float64(150)
	minLen := 3
	maxLen := 50

	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = "User account"
	typ.Fields = []*typegraph.Field{
		{
			JSONName:    "id",
			Description: "User ID",
			Type:        &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string", Format: "uuid"},
			Required:    true,
		},
		{
			JSONName:    "email",
			Description: "User email",
			Type:        &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string", Format: "email"},
			Required:    true,
		},
		{
			JSONName:  "username",
			Type:      &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "string"},
			MinLength: &minLen,
			MaxLength: &maxLen,
			Required:  true,
		},
		{
			JSONName: "age",
			Type:     &typegraph.TypeRef{Kind: typegraph.KindPrimitive, GoType: "int"},
			Minimum:  &minAge,
			Maximum:  &maxAge,
			Required: false,
		},
	}

	result := e.GenerateSchema(typ)

	// Check structure
	assert.Contains(t, result, "export const UserSchema = z.object({")
	// Check Zod v4 top-level formats
	assert.Contains(t, result, "id: z.uuid()")
	assert.Contains(t, result, "email: z.email()")
	// Check constraints
	assert.Contains(t, result, "username: z.string().min(3).max(50)")
	assert.Contains(t, result, "age: z.int().gte(0).lte(150).optional()")
	// Check strict mode
	assert.Contains(t, result, "}).strict()")
	// Check metadata
	assert.Contains(t, result, `.meta({ description: "User account" })`)
	// Check field descriptions
	assert.Contains(t, result, `.meta({ description: "User ID" })`)
	assert.Contains(t, result, `.meta({ description: "User email" })`)
}

func TestEmitter_CompleteSchemaWithInfer(t *testing.T) {
	e := createTestEmitter(nil)

	typ := createTestType("User", typegraph.KindStruct)
	typ.Description = "User account"
	typ.Fields = []*typegraph.Field{
		createTestField("id", "string", true),
	}

	result := e.GenerateSchemaWithInfer(typ)

	// Check JSDoc comment
	assert.Contains(t, result, "/**")
	assert.Contains(t, result, "* User account")
	assert.Contains(t, result, "*/")
	// Check schema
	assert.Contains(t, result, "export const UserSchema = z.object({")
	// Check z.infer export
	assert.Contains(t, result, "export type User = z.infer<typeof UserSchema>;")
}
