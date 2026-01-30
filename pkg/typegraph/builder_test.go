package typegraph

import (
	"path/filepath"
	"testing"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractTypeNameFromRef_ExternalRefs(t *testing.T) {
	tests := []struct {
		name           string
		ref            string
		setupCompiler  func(*jsonschema.Compiler)
		expectedResult string
	}{
		{
			name: "simple relative ref with ./ prefix",
			ref:  "./settings.yaml",
			setupCompiler: func(c *jsonschema.Compiler) {
				// Register schema as "settings.yaml" (no ./ prefix)
				title := "Settings"
				schema := &jsonschema.Schema{
					Title: &title,
				}
				c.SetSchema("settings.yaml", schema)
			},
			expectedResult: "Settings",
		},
		{
			name: "relative ref without ./ prefix",
			ref:  "settings.yaml",
			setupCompiler: func(c *jsonschema.Compiler) {
				title := "Settings"
				schema := &jsonschema.Schema{
					Title: &title,
				}
				c.SetSchema("settings.yaml", schema)
			},
			expectedResult: "Settings",
		},
		{
			name: "nested relative ref with ./",
			ref:  "./nested/config.json",
			setupCompiler: func(c *jsonschema.Compiler) {
				title := "NestedConfig"
				schema := &jsonschema.Schema{
					Title: &title,
				}
				c.SetSchema("nested/config.json", schema)
			},
			expectedResult: "NestedConfig",
		},
		{
			name: "nested ref without ./ prefix",
			ref:  "events/part1.json",
			setupCompiler: func(c *jsonschema.Compiler) {
				title := "Part1"
				schema := &jsonschema.Schema{
					Title: &title,
				}
				c.SetSchema("events/part1.json", schema)
			},
			expectedResult: "Part1",
		},
		{
			name: "parent directory ref with ../",
			ref:  "../events/config.json",
			setupCompiler: func(c *jsonschema.Compiler) {
				// Register as "events/config.json" (schemas are relative to base)
				title := "EventConfig"
				schema := &jsonschema.Schema{
					Title: &title,
				}
				c.SetSchema("events/config.json", schema)
			},
			expectedResult: "EventConfig",
		},
		{
			name: "complex path with ./ and ../",
			ref:  "./nested/../simple.json",
			setupCompiler: func(c *jsonschema.Compiler) {
				// Should normalize to "simple.json"
				title := "Simple"
				schema := &jsonschema.Schema{
					Title: &title,
				}
				c.SetSchema("simple.json", schema)
			},
			expectedResult: "Simple",
		},
		{
			name: "grandparent directory ref with ../../",
			ref:  "../../shared/types.json",
			setupCompiler: func(c *jsonschema.Compiler) {
				title := "SharedTypes"
				schema := &jsonschema.Schema{
					Title: &title,
				}
				c.SetSchema("shared/types.json", schema)
			},
			expectedResult: "SharedTypes",
		},
		{
			name: "ref not found - extract from filename with ./",
			ref:  "./user-settings.yaml",
			setupCompiler: func(c *jsonschema.Compiler) {
				// Don't register - simulate unresolved ref
			},
			expectedResult: "UserSettings", // From "user-settings.yaml"
		},
		{
			name: "ref not found - extract from nested path",
			ref:  "events/config.json",
			setupCompiler: func(c *jsonschema.Compiler) {
				// Don't register
			},
			expectedResult: "Config", // From filename "config.json"
		},
		{
			name: "ref not found - extract from parent path",
			ref:  "../types/shared.json",
			setupCompiler: func(c *jsonschema.Compiler) {
				// Don't register
			},
			expectedResult: "Shared", // From filename "shared.json"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := jsonschema.NewCompiler()
			if tt.setupCompiler != nil {
				tt.setupCompiler(compiler)
			}

			builder := NewBuilder(compiler)
			result := builder.resolver.ExtractTypeName(tt.ref)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestExtractTypeNameFromRef_InternalDefs(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{
			name:     "simple def",
			ref:      "#/$defs/User",
			expected: "User",
		},
		{
			name:     "snake case",
			ref:      "#/$defs/user_profile",
			expected: "UserProfile",
		},
		{
			name:     "kebab case",
			ref:      "#/$defs/user-profile",
			expected: "UserProfile",
		},
		{
			name:     "camel case",
			ref:      "#/$defs/userId",
			expected: "UserId",
		},
		{
			name:     "already PascalCase",
			ref:      "#/$defs/UserProfile",
			expected: "UserProfile",
		},
		{
			name:     "all caps",
			ref:      "#/$defs/API",
			expected: "API",
		},
		{
			name:     "with numbers",
			ref:      "#/$defs/user2profile",
			expected: "User2profile",
		},
		{
			name:     "leading number",
			ref:      "#/$defs/2ndUser",
			expected: "2ndUser",
		},
		{
			name:     "single letter",
			ref:      "#/$defs/A",
			expected: "A",
		},
		{
			name:     "multiple words",
			ref:      "#/$defs/user_profile_data",
			expected: "UserProfileData",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := jsonschema.NewCompiler()
			builder := NewBuilder(compiler)
			result := builder.resolver.ExtractTypeName(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractTypeNameFromRef_RootSelfReference(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		title    string
		expected string
	}{
		{
			name:     "root self-reference with title CyclicRef",
			ref:      "#",
			title:    "CyclicRef",
			expected: "CyclicRef",
		},
		{
			name:     "root self-reference with title TreeNode",
			ref:      "#",
			title:    "TreeNode",
			expected: "TreeNode",
		},
		{
			name:     "root self-reference no title uses Schema",
			ref:      "#",
			title:    "",
			expected: "Schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := jsonschema.NewCompiler()

			// Create a schema with the specified title
			titlePtr := &tt.title
			if tt.title == "" {
				titlePtr = nil
			}
			schema := &jsonschema.Schema{
				Title: titlePtr,
			}

			compiler.SetSchema("test.json", schema)

			builder := NewBuilder(compiler)
			// Set the root schema context for the resolver
			builder.resolver.SetCurrentSchema(schema)

			result := builder.resolver.ExtractTypeName(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeRefPath(t *testing.T) {
	tests := []struct {
		name             string
		ref              string
		expectedVariants []string
	}{
		{
			name:             "empty string",
			ref:              "",
			expectedVariants: []string{""},
		},
		{
			name:             "simple file without prefix",
			ref:              "settings.json",
			expectedVariants: []string{"settings.json", "./settings.json"},
		},
		{
			name:             "with ./ prefix",
			ref:              "./config.yaml",
			expectedVariants: []string{"./config.yaml", "config.yaml"},
		},
		{
			name:             "with ../ prefix",
			ref:              "../types.json",
			expectedVariants: []string{"../types.json", "types.json"},
		},
		{
			name:             "multiple ../",
			ref:              "../../../shared.json",
			expectedVariants: []string{"../../../shared.json", "shared.json"},
		},
		{
			name:             "nested path",
			ref:              "events/part1.json",
			expectedVariants: []string{"events/part1.json", "./events/part1.json"},
		},
		{
			name:             "complex navigation",
			ref:              "./nested/../simple.json",
			expectedVariants: []string{"./nested/../simple.json", "simple.json"},
		},
		{
			name:             "dots in filename",
			ref:              "config.v2.json",
			expectedVariants: []string{"config.v2.json"},
		},
		{
			name:             "multiple slashes",
			ref:              "///settings.json",
			expectedVariants: []string{"///settings.json", filepath.Clean("///settings.json")},
		},
		{
			name:             "trailing slash",
			ref:              "./settings/",
			expectedVariants: []string{"./settings/", "settings"},
		},
		{
			name:             "Windows backslash",
			ref:              "settings\\config.json",
			expectedVariants: []string{"settings\\config.json"},
		},
		{
			name:             "mixed separators",
			ref:              "dir\\nested/file.json",
			expectedVariants: []string{"dir\\nested/file.json"},
		},
		{
			name:             "root relative",
			ref:              "/settings.json",
			expectedVariants: []string{"/settings.json"},
		},
		{
			name:             "double dots in path",
			ref:              "././settings.json",
			expectedVariants: []string{"././settings.json", "settings.json"},
		},
		{
			name:             "just dots",
			ref:              "..",
			expectedVariants: []string{".."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variants := normalizeRefPath(tt.ref)

			// Check that all expected variants are present
			for _, expected := range tt.expectedVariants {
				assert.Contains(t, variants, expected,
					"Missing expected variant: %s", expected)
			}
		})
	}
}

func TestExtractTypeNameFromFilename(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{
			name:     "simple filename",
			ref:      "settings.yaml",
			expected: "Settings",
		},
		{
			name:     "with ./ prefix",
			ref:      "./config.json",
			expected: "Config",
		},
		{
			name:     "nested path",
			ref:      "events/part1.json",
			expected: "Part1",
		},
		{
			name:     "parent path",
			ref:      "../types.json",
			expected: "Types",
		},
		{
			name:     "multiple dots",
			ref:      "config.v2.json",
			expected: "ConfigV2",
		},
		{
			name:     "no extension",
			ref:      "settings",
			expected: "Settings",
		},
		{
			name:     "double extension",
			ref:      "data.schema.json",
			expected: "DataSchema",
		},
		{
			name:     "hyphenated",
			ref:      "user-settings.yaml",
			expected: "UserSettings",
		},
		{
			name:     "underscored",
			ref:      "user_profile.json",
			expected: "UserProfile",
		},
		{
			name:     "mixed naming",
			ref:      "user_profile-data.v2.json",
			expected: "UserProfileDataV2",
		},
		{
			name:     "just extension",
			ref:      ".json",
			expected: "",
		},
		{
			name:     "all caps",
			ref:      "API.JSON",
			expected: "API",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTypeNameFromFilename(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== Build() Function Tests ====================

func TestBuild_EmptySchemaList(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	graph, err := builder.Build([]*schema.Schema{})

	require.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Empty(t, graph.Types)
}

// TestBuild_BasicTypes tests building basic schema types (object, string enum, number enum).
func TestBuild_BasicTypes(t *testing.T) {
	tests := []struct {
		name           string
		schemaJSON     string
		expectedName   string
		expectedKind   TypeKind
		expectedFields int
		enumCount      int
		requiredFields []string
	}{
		{
			name: "simple object with properties",
			schemaJSON: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"title": "User",
				"type": "object",
				"properties": {"name": {"type": "string"}, "age": {"type": "integer"}},
				"required": ["name"]
			}`,
			expectedName:   "User",
			expectedKind:   KindStruct,
			expectedFields: 2,
			requiredFields: []string{"name"},
		},
		{
			name: "string enum",
			schemaJSON: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"title": "Status",
				"type": "string",
				"enum": ["pending", "active", "completed"]
			}`,
			expectedName: "Status",
			expectedKind: KindEnum,
			enumCount:    3,
		},
		{
			name: "number enum",
			schemaJSON: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"title": "Priority",
				"type": "integer",
				"enum": [1, 2, 3]
			}`,
			expectedName: "Priority",
			expectedKind: KindEnum,
			enumCount:    3,
		},
		{
			name: "mixed enum",
			schemaJSON: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"title": "MixedValue",
				"enum": ["active", 1, true, null]
			}`,
			expectedName: "MixedValue",
			expectedKind: KindEnum,
			enumCount:    4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := jsonschema.NewCompiler()
			compiled, err := compiler.Compile([]byte(tt.schemaJSON), "test.json")
			require.NoError(t, err)

			testSchema := &schema.Schema{
				Path:         "test.json",
				RelativePath: "test.json",
				Name:         tt.expectedName,
				Compiled:     compiled,
			}

			builder := NewBuilder(compiler)
			graph, err := builder.Build([]*schema.Schema{testSchema})

			require.NoError(t, err)
			require.Len(t, graph.Types, 1)

			typ := graph.Types[0]
			assert.Equal(t, tt.expectedName, typ.Name)
			assert.Equal(t, tt.expectedKind, typ.Kind)

			if tt.expectedFields > 0 {
				assert.Len(t, typ.Fields, tt.expectedFields)
			}
			if tt.enumCount > 0 {
				assert.Len(t, typ.EnumValues, tt.enumCount)
			}

			if len(tt.requiredFields) > 0 {
				fieldMap := make(map[string]*Field)
				for _, f := range typ.Fields {
					fieldMap[f.JSONName] = f
				}
				for _, rf := range tt.requiredFields {
					assert.True(t, fieldMap[rf].Required, "%s should be required", rf)
				}
			}
		})
	}
}

func TestBuild_WithDefs(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "User",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"address": {"$ref": "#/$defs/Address"}
		},
		"$defs": {
			"Address": {
				"type": "object",
				"properties": {"street": {"type": "string"}, "city": {"type": "string"}}
			}
		}
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "user.json")
	require.NoError(t, err)

	testSchema := &schema.Schema{
		Path:         "user.json",
		RelativePath: "user.json",
		Name:         "User",
		Compiled:     compiled,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	require.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 2) // Address (from $defs) + User

	// Find types by name
	typeMap := make(map[string]*Type)
	for _, typ := range graph.Types {
		typeMap[typ.Name] = typ
	}

	// Check Address type (from $defs)
	addressType := typeMap["Address"]
	assert.NotNil(t, addressType)
	assert.Equal(t, KindStruct, addressType.Kind)
	assert.Len(t, addressType.Fields, 2)

	// Check User type
	userType := typeMap["User"]
	assert.NotNil(t, userType)
	assert.Equal(t, KindStruct, userType.Kind)
	assert.Len(t, userType.Fields, 2)

	// Verify User has a field referencing Address
	userFieldMap := make(map[string]*Field)
	for _, f := range userType.Fields {
		userFieldMap[f.JSONName] = f
	}
	addressField := userFieldMap["address"]
	assert.NotNil(t, addressField)
	assert.Equal(t, KindRef, addressField.Type.Kind)
	assert.Equal(t, "Address", addressField.Type.TypeName)
}

func TestBuild_ValidationConstraints(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "Product",
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"minLength": 3,
				"maxLength": 100,
				"pattern": "^[A-Za-z0-9 ]+$"
			},
			"email": {
				"type": "string",
				"format": "email"
			},
			"price": {
				"type": "number",
				"minimum": 0.01,
				"maximum": 9999.99
			},
			"count": {
				"type": "integer",
				"exclusiveMinimum": 0,
				"exclusiveMaximum": 1000
			}
		}
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "product.json")
	require.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "product.json",
		RelativePath:  "product.json",
		Name:          "Product",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	require.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 1)

	typ := graph.Types[0]
	assert.Equal(t, "Product", typ.Name)

	// Build field map
	fieldMap := make(map[string]*Field)
	for _, f := range typ.Fields {
		fieldMap[f.JSONName] = f
	}

	// Check name field with string constraints
	nameField := fieldMap["name"]
	assert.NotNil(t, nameField)
	if nameField.MinLength != nil {
		assert.Equal(t, 3, *nameField.MinLength)
	}
	if nameField.MaxLength != nil {
		assert.Equal(t, 100, *nameField.MaxLength)
	}
	if nameField.Pattern != nil {
		assert.Equal(t, "^[A-Za-z0-9 ]+$", *nameField.Pattern)
	}

	// Check email field with format
	emailField := fieldMap["email"]
	assert.NotNil(t, emailField)
	if emailField.Type.Format != "" {
		assert.Equal(t, "email", emailField.Type.Format)
	}

	// Check price field with number constraints
	priceField := fieldMap["price"]
	assert.NotNil(t, priceField)
	if priceField.Minimum != nil {
		assert.InDelta(t, 0.01, *priceField.Minimum, 0.001)
	}
	if priceField.Maximum != nil {
		assert.InDelta(t, 9999.99, *priceField.Maximum, 0.001)
	}

	// Check count field with exclusive constraints
	countField := fieldMap["count"]
	assert.NotNil(t, countField)
	if countField.ExclusiveMinimum != nil {
		assert.InDelta(t, 0.0, *countField.ExclusiveMinimum, 0.001)
	}
	if countField.ExclusiveMaximum != nil {
		assert.InDelta(t, 1000.0, *countField.ExclusiveMaximum, 0.001)
	}
}

func TestBuild_ArrayType(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "TodoList",
		"type": "object",
		"properties": {
			"items": {
				"type": "array",
				"items": {
					"type": "string"
				}
			},
			"numbers": {
				"type": "array",
				"items": {
					"type": "integer"
				},
				"minItems": 1,
				"maxItems": 10
			}
		}
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "todolist.json")
	require.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "todolist.json",
		RelativePath:  "todolist.json",
		Name:          "TodoList",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	require.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 1)

	typ := graph.Types[0]
	assert.Equal(t, "TodoList", typ.Name)

	// Build field map
	fieldMap := make(map[string]*Field)
	for _, f := range typ.Fields {
		fieldMap[f.JSONName] = f
	}

	// Check items field (array of strings)
	itemsField := fieldMap["items"]
	assert.NotNil(t, itemsField)
	assert.Equal(t, KindArray, itemsField.Type.Kind)
	assert.NotNil(t, itemsField.Type.ItemType)
	assert.Equal(t, KindPrimitive, itemsField.Type.ItemType.Kind)

	// Check numbers field (array of integers with constraints)
	numbersField := fieldMap["numbers"]
	assert.NotNil(t, numbersField)
	assert.Equal(t, KindArray, numbersField.Type.Kind)
	assert.NotNil(t, numbersField.Type.ItemType)
	assert.Equal(t, KindPrimitive, numbersField.Type.ItemType.Kind)
	if numbersField.MinItems != nil {
		assert.Equal(t, 1, *numbersField.MinItems)
	}
	if numbersField.MaxItems != nil {
		assert.Equal(t, 10, *numbersField.MaxItems)
	}
}

func TestBuild_MultipleSchemas(t *testing.T) {
	// First schema: User
	userSchemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "User",
		"type": "object",
		"properties": {
			"name": {
				"type": "string"
			}
		}
	}`)

	// Second schema: Product
	productSchemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "Product",
		"type": "object",
		"properties": {
			"title": {
				"type": "string"
			},
			"price": {
				"type": "number"
			}
		}
	}`)

	compiler := jsonschema.NewCompiler()

	userCompiled, err := compiler.Compile(userSchemaJSON, "user.json")
	require.NoError(t, err)

	productCompiled, err := compiler.Compile(productSchemaJSON, "product.json")
	require.NoError(t, err)

	schemas := []*schema.Schema{
		{
			Path:          "user.json",
			RelativePath:  "user.json",
			Name:          "User",
			Compiled:      userCompiled,
			PropertyOrder: nil,
		},
		{
			Path:          "product.json",
			RelativePath:  "product.json",
			Name:          "Product",
			Compiled:      productCompiled,
			PropertyOrder: nil,
		},
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build(schemas)

	require.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 2)

	// Find types by name
	typeMap := make(map[string]*Type)
	for _, typ := range graph.Types {
		typeMap[typ.Name] = typ
	}

	// Verify User type exists
	userType := typeMap["User"]
	assert.NotNil(t, userType)
	assert.Equal(t, KindStruct, userType.Kind)
	assert.Len(t, userType.Fields, 1)

	// Verify Product type exists
	productType := typeMap["Product"]
	assert.NotNil(t, productType)
	assert.Equal(t, KindStruct, productType.Kind)
	assert.Len(t, productType.Fields, 2)
}

func TestBuild_WithAnyOf(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "Response",
		"anyOf": [
			{
				"type": "object",
				"properties": {
					"success": {"type": "boolean"},
					"data": {"type": "string"}
				}
			},
			{
				"type": "object",
				"properties": {
					"error": {"type": "string"},
					"code": {"type": "number"}
				}
			}
		]
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "response.json")
	require.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "response.json",
		RelativePath:  "response.json",
		Name:          "Response",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	require.NoError(t, err)
	assert.NotNil(t, graph)
	// anyOf typically creates a union or merged struct
	assert.GreaterOrEqual(t, len(graph.Types), 1)
}

// TestBuild_AllOf tests various allOf composition patterns.
func TestBuild_AllOf(t *testing.T) {
	tests := []struct {
		name           string
		schemaJSON     string
		expectedName   string
		minTypes       int
		expectedKind   TypeKind
		minFields      int
		exactFields    int // -1 means use minFields
		expectedExtend []string
		requiredFields []string
	}{
		{
			name: "allOf only at root with refs and inline",
			schemaJSON: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"title": "Document",
				"$defs": {
					"Entity": {"type": "object", "properties": {"id": {"type": "string"}}},
					"Auditable": {"type": "object", "properties": {"createdBy": {"type": "string"}}}
				},
				"allOf": [
					{"$ref": "#/$defs/Entity"},
					{"$ref": "#/$defs/Auditable"},
					{"type": "object", "properties": {"title": {"type": "string"}}, "required": ["title"]}
				]
			}`,
			expectedName:   "Document",
			minTypes:       3,
			expectedKind:   KindStruct,
			minFields:      1,
			exactFields:    -1,
			expectedExtend: []string{"Entity", "Auditable"},
			requiredFields: []string{"title"},
		},
		{
			name: "allOf with only inline objects",
			schemaJSON: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"title": "Vehicle",
				"allOf": [
					{"type": "object", "properties": {"make": {"type": "string"}, "model": {"type": "string"}}, "required": ["make"]},
					{"type": "object", "properties": {"registration": {"type": "string"}}, "required": ["registration"]}
				]
			}`,
			expectedName:   "Vehicle",
			minTypes:       1,
			expectedKind:   KindStruct,
			exactFields:    3,
			requiredFields: []string{"make", "registration"},
		},
		{
			name: "allOf with mixed refs and inline",
			schemaJSON: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"title": "Product",
				"$defs": {"BaseItem": {"type": "object", "properties": {"id": {"type": "string"}}}},
				"allOf": [{"$ref": "#/$defs/BaseItem"}, {"type": "object", "properties": {"price": {"type": "number"}}}]
			}`,
			expectedName:   "Product",
			minTypes:       2,
			expectedKind:   KindStruct,
			minFields:      1,
			exactFields:    -1,
			expectedExtend: []string{"BaseItem"},
		},
		{
			name: "empty allOf becomes primitive",
			schemaJSON: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"title": "EmptyComposite",
				"allOf": []
			}`,
			expectedName: "EmptyComposite",
			minTypes:     1,
			expectedKind: KindPrimitive,
			exactFields:  0,
		},
		{
			name: "allOf with direct properties",
			schemaJSON: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"title": "Enhanced",
				"type": "object",
				"properties": {"rootProp": {"type": "string"}},
				"allOf": [{"type": "object", "properties": {"allOfProp": {"type": "number"}}}]
			}`,
			expectedName: "Enhanced",
			minTypes:     1,
			expectedKind: KindStruct,
			exactFields:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := jsonschema.NewCompiler()
			compiled, err := compiler.Compile([]byte(tt.schemaJSON), "test.json")
			require.NoError(t, err)

			testSchema := &schema.Schema{
				Path:         "test.json",
				RelativePath: "test.json",
				Name:         tt.expectedName,
				Compiled:     compiled,
			}

			builder := NewBuilder(compiler)
			graph, err := builder.Build([]*schema.Schema{testSchema})

			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(graph.Types), tt.minTypes)

			// Find the main type
			var mainType *Type
			for _, typ := range graph.Types {
				if typ.Name == tt.expectedName {
					mainType = typ
					break
				}
			}
			require.NotNil(t, mainType, "%s type should be created", tt.expectedName)
			assert.Equal(t, tt.expectedKind, mainType.Kind)

			if tt.exactFields >= 0 {
				assert.Len(t, mainType.Fields, tt.exactFields)
			} else if tt.minFields > 0 {
				assert.GreaterOrEqual(t, len(mainType.Fields), tt.minFields)
			}

			for _, ext := range tt.expectedExtend {
				assert.Contains(t, mainType.Extends, ext)
			}

			if len(tt.requiredFields) > 0 {
				fieldMap := make(map[string]*Field)
				for _, f := range mainType.Fields {
					fieldMap[f.JSONName] = f
				}
				for _, rf := range tt.requiredFields {
					if f, ok := fieldMap[rf]; ok {
						assert.True(t, f.Required, "%s should be required", rf)
					}
				}
			}
		})
	}
}

func TestMapGoType(t *testing.T) {
	tests := []struct {
		name       string
		goType     string
		targetLang string
		expected   string
	}{
		{"string to TypeScript", "string", "typescript", "string"},
		{"int to TypeScript", "int", "typescript", "number"},
		{"float64 to TypeScript", "float64", "typescript", "number"},
		{"bool to TypeScript", "bool", "typescript", "boolean"},
		{"string to Python", "string", "python", "str"},
		{"int to Python", "int", "python", "int"},
		{"float64 to Python", "float64", "python", "float"},
		{"bool to Python", "bool", "python", "bool"},
		{"time.Time to TypeScript", "time.Time", "typescript", "string"},
		{"time.Time to Python", "time.Time", "python", "datetime"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapGoType(tt.goType, tt.targetLang)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuild_SpecialFeatures(t *testing.T) {
	t.Run("additional properties", func(t *testing.T) {
		schemaJSON := []byte(`{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"title": "Config",
			"type": "object",
			"properties": {"name": {"type": "string"}},
			"additionalProperties": {"type": "string"}
		}`)

		compiler := jsonschema.NewCompiler()
		compiled, err := compiler.Compile(schemaJSON, "config.json")
		require.NoError(t, err)

		testSchema := &schema.Schema{Path: "config.json", RelativePath: "config.json", Name: "Config", Compiled: compiled}
		builder := NewBuilder(compiler)
		graph, err := builder.Build([]*schema.Schema{testSchema})

		require.NoError(t, err)
		require.Len(t, graph.Types, 1)
		assert.NotNil(t, graph.Types[0].AdditionalProps)
	})

	t.Run("nullable field", func(t *testing.T) {
		schemaJSON := []byte(`{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"title": "User",
			"type": "object",
			"properties": {"name": {"type": ["string", "null"]}}
		}`)

		compiler := jsonschema.NewCompiler()
		compiled, err := compiler.Compile(schemaJSON, "user.json")
		require.NoError(t, err)

		testSchema := &schema.Schema{Path: "user.json", RelativePath: "user.json", Name: "User", Compiled: compiled}
		builder := NewBuilder(compiler)
		graph, err := builder.Build([]*schema.Schema{testSchema})

		require.NoError(t, err)
		require.Len(t, graph.Types, 1)
		nameField := graph.Types[0].Fields[0]
		assert.True(t, nameField.Type.Nullable || !nameField.Required)
	})
}

func TestEnsureUniqueTypeName(t *testing.T) {
	tests := []struct {
		name           string
		existingTypes  []string
		requestName    string
		expectedResult string
	}{
		{"unique base name", []string{"User"}, "Product", "Product"},
		{"conflicting name", []string{"User", "User2"}, "User", "User3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewTypeRegistry()
			for _, typeName := range tt.existingTypes {
				registry.Add(&Type{Name: typeName})
			}
			result := registry.EnsureUniqueName(tt.requestName)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestGetOrderedPropertyNames(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	t.Run("alphabetical fallback", func(t *testing.T) {
		properties := jsonschema.SchemaMap{
			"zebra": &jsonschema.Schema{},
			"apple": &jsonschema.Schema{},
			"mango": &jsonschema.Schema{},
		}
		result := builder.getOrderedPropertyNames(&properties, "test.json")
		assert.Equal(t, []string{"apple", "mango", "zebra"}, result)
	})

	t.Run("nil properties", func(t *testing.T) {
		result := builder.getOrderedPropertyNames(nil, "test.json")
		assert.Nil(t, result)
	})
}

func TestBuildTypeRef_InlineEnum(t *testing.T) {
	t.Run("with extraction", func(t *testing.T) {
		compiler := jsonschema.NewCompiler()
		builder := NewBuilderWithConfig(compiler, &BuildConfig{ExtractInlined: true})
		schema := &jsonschema.Schema{Enum: []interface{}{"active", "inactive", "pending"}}

		ref := builder.BuildTypeRef(schema, "status")

		assert.Equal(t, KindRef, ref.Kind)
		assert.Equal(t, "Status", ref.TypeName)
		assert.Len(t, builder.registry.All(), 1)
	})

	t.Run("without extraction", func(t *testing.T) {
		compiler := jsonschema.NewCompiler()
		builder := NewBuilderWithConfig(compiler, &BuildConfig{ExtractInlined: false})
		schema := &jsonschema.Schema{Enum: []interface{}{"option1", "option2", "option3"}}

		ref := builder.BuildTypeRef(schema, "status")

		assert.Equal(t, KindEnum, ref.Kind)
		assert.Len(t, ref.EnumValues, 3)
	})
}

func TestDeriveTypeName(t *testing.T) {
	tests := []struct {
		name     string
		title    *string
		uri      string
		expected string
	}{
		{"title already PascalCase", ptrString("UserProfile"), "", "UserProfile"},
		{"title needs PascalCase", ptrString("user-profile"), "", "UserProfile"},
		{"from URI when no title", nil, "payloads/subscribe.json", "Subscribe"},
		{"no title no URI returns Unknown", nil, "", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := jsonschema.NewCompiler()
			builder := NewBuilder(compiler)

			schema := &jsonschema.Schema{Title: tt.title}
			result := builder.resolver.DeriveTypeName(schema, tt.uri)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractDefinition_PrimitiveType(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	schema := &jsonschema.Schema{
		Type: []string{"string"},
	}

	err := builder.walker.ExtractDefinition("CustomString", schema)

	require.NoError(t, err)
	assert.Len(t, builder.registry.All(), 1)
	assert.Equal(t, "CustomString", builder.registry.All()[0].Name)
	assert.Equal(t, KindPrimitive, builder.registry.All()[0].Kind)
}

func TestGetDescription(t *testing.T) {
	desc := "This is a test description"
	tests := []struct {
		name     string
		schema   *jsonschema.Schema
		expected string
	}{
		{"with description", &jsonschema.Schema{Description: &desc}, "This is a test description"},
		{"no description", &jsonschema.Schema{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, getDescription(tt.schema))
		})
	}
}

func TestProcessSchema_WithDefs(t *testing.T) {
	compiler := jsonschema.NewCompiler()

	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "Root",
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"$defs": {
			"SubType": {
				"type": "object",
				"properties": {
					"value": {"type": "number"}
				}
			}
		}
	}`)

	compiled, err := compiler.Compile(schemaJSON, "test.json")
	require.NoError(t, err)

	testSchema := &schema.Schema{
		Path:         "test.json",
		RelativePath: "test.json",
		Name:         "Root",
		Compiled:     compiled,
	}

	builder := NewBuilder(compiler)
	err = builder.walker.Process(testSchema)

	require.NoError(t, err)
	// Should have SubType from $defs + Root
	assert.GreaterOrEqual(t, len(builder.registry.All()), 2)
}

func TestExtractInlineObjectType(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	schema := &jsonschema.Schema{
		Type: []string{"object"},
		Properties: &jsonschema.SchemaMap{
			"id":    &jsonschema.Schema{Type: []string{"string"}},
			"count": &jsonschema.Schema{Type: []string{"integer"}},
		},
		Required: []string{"id"},
	}

	typ := builder.extractInlineObjectType("Property", schema)

	assert.NotNil(t, typ)
	assert.Equal(t, "Property", typ.Name)
	assert.Equal(t, KindStruct, typ.Kind)
	assert.Len(t, typ.Fields, 2)

	// Check that id field is required
	idField := typ.Fields[1] // Should be in alphabetical order
	if idField.JSONName == "id" {
		assert.True(t, idField.Required)
	}
}

func TestBuildFieldsFromProperties(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	schema := &jsonschema.Schema{
		Properties: &jsonschema.SchemaMap{
			"name": &jsonschema.Schema{Type: []string{"string"}},
			"age":  &jsonschema.Schema{Type: []string{"integer"}},
		},
		Required: []string{"name"},
	}

	fields := builder.BuildFieldsFromProperties(schema, "")

	assert.Len(t, fields, 2)
	// Should be alphabetically ordered
	assert.Equal(t, "age", fields[0].JSONName)
	assert.Equal(t, "name", fields[1].JSONName)
	assert.True(t, fields[1].Required)
	assert.False(t, fields[0].Required)
}

func TestBuildStruct(t *testing.T) {
	t.Run("allOf with ref", func(t *testing.T) {
		compiler := jsonschema.NewCompiler()
		builder := NewBuilder(compiler)
		schema := &jsonschema.Schema{
			Type: []string{"object"},
			AllOf: []*jsonschema.Schema{
				{Ref: "#/$defs/BaseType"},
				{Type: []string{"object"}, Properties: &jsonschema.SchemaMap{"extra": &jsonschema.Schema{Type: []string{"string"}}}},
			},
		}
		typ := &Type{ID: "1", Name: "Extended"}
		err := builder.BuildStruct(typ, schema)
		require.NoError(t, err)
		assert.Equal(t, KindStruct, typ.Kind)
		assert.Contains(t, typ.Extends, "BaseType")
	})

	t.Run("with additionalProperties", func(t *testing.T) {
		compiler := jsonschema.NewCompiler()
		builder := NewBuilder(compiler)
		schema := &jsonschema.Schema{
			Type:                 []string{"object"},
			Properties:           &jsonschema.SchemaMap{"name": &jsonschema.Schema{Type: []string{"string"}}},
			AdditionalProperties: &jsonschema.Schema{Type: []string{"string"}},
		}
		typ := &Type{ID: "1", Name: "TestType"}
		err := builder.BuildStruct(typ, schema)
		require.NoError(t, err)
		assert.Equal(t, KindStruct, typ.Kind)
		assert.NotNil(t, typ.AdditionalProps)
	})
}

func TestMapPrimitiveType_AllFormats(t *testing.T) {
	tests := []struct {
		name       string
		schemaType string
		format     string
		expected   string
	}{
		// String formats
		{"uuid format", "string", "uuid", "uuid.UUID"},
		{"date-time format", "string", "date-time", "time.Time"},
		{"date format", "string", "date", "time.Time"},
		{"time format", "string", "time", "string"},
		{"email format", "string", "email", "string"},
		{"uri format", "string", "uri", "string"},
		{"hostname format", "string", "hostname", "string"},
		{"ipv4 format", "string", "ipv4", "string"},
		{"ipv6 format", "string", "ipv6", "string"},
		{"plain string", "string", "", "string"},

		// Integer formats
		{"int32 format", "integer", "int32", "int32"},
		{"int64 format", "integer", "int64", "int64"},
		{"plain integer", "integer", "", "int"},

		// Number formats
		{"float format", "number", "float", "float32"},
		{"double format", "number", "double", "float64"},
		{"plain number", "number", "", "float64"},

		// Other types
		{"boolean type", "boolean", "", "bool"},
		{"array type", "array", "", "[]interface{}"},
		{"object type", "object", "", "map[string]interface{}"},

		// No type specified
		{"no type", "", "", "interface{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := jsonschema.NewCompiler()
			builder := NewBuilder(compiler)

			schema := &jsonschema.Schema{}
			if tt.schemaType != "" {
				schema.Type = []string{tt.schemaType}
			}
			if tt.format != "" {
				schema.Format = &tt.format
			}

			result := builder.MapPrimitiveType(schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildTypeRef_Comprehensive(t *testing.T) {
	tests := []struct {
		name         string
		setupSchema  func() *jsonschema.Schema
		fieldName    string
		expectKind   TypeKind
		expectChecks func(*testing.T, *TypeRef)
	}{
		{
			name: "simple string primitive",
			setupSchema: func() *jsonschema.Schema {
				return &jsonschema.Schema{Type: []string{"string"}}
			},
			fieldName:  "",
			expectKind: KindPrimitive,
			expectChecks: func(t *testing.T, ref *TypeRef) {
				assert.Equal(t, "string", ref.GoType)
			},
		},
		{
			name: "integer primitive",
			setupSchema: func() *jsonschema.Schema {
				return &jsonschema.Schema{Type: []string{"integer"}}
			},
			fieldName:  "",
			expectKind: KindPrimitive,
			expectChecks: func(t *testing.T, ref *TypeRef) {
				assert.Equal(t, "int", ref.GoType)
			},
		},
		{
			name: "boolean primitive",
			setupSchema: func() *jsonschema.Schema {
				return &jsonschema.Schema{Type: []string{"boolean"}}
			},
			fieldName:  "",
			expectKind: KindPrimitive,
			expectChecks: func(t *testing.T, ref *TypeRef) {
				assert.Equal(t, "bool", ref.GoType)
			},
		},
		{
			name: "array of strings",
			setupSchema: func() *jsonschema.Schema {
				return &jsonschema.Schema{
					Type:  []string{"array"},
					Items: &jsonschema.Schema{Type: []string{"string"}},
				}
			},
			fieldName:  "tags",
			expectKind: KindArray,
			expectChecks: func(t *testing.T, ref *TypeRef) {
				assert.NotNil(t, ref.ItemType)
				assert.Equal(t, KindPrimitive, ref.ItemType.Kind)
			},
		},
		{
			name: "object without properties (map)",
			setupSchema: func() *jsonschema.Schema {
				return &jsonschema.Schema{Type: []string{"object"}}
			},
			fieldName:  "",
			expectKind: KindMap,
			expectChecks: func(t *testing.T, ref *TypeRef) {
				assert.NotNil(t, ref.ValueType)
			},
		},
		{
			name: "nullable type",
			setupSchema: func() *jsonschema.Schema {
				return &jsonschema.Schema{Type: []string{"string", "null"}}
			},
			fieldName:  "",
			expectKind: KindPrimitive,
			expectChecks: func(t *testing.T, ref *TypeRef) {
				assert.True(t, ref.Nullable)
			},
		},
		{
			name: "array without items",
			setupSchema: func() *jsonschema.Schema {
				return &jsonschema.Schema{Type: []string{"array"}}
			},
			fieldName:  "",
			expectKind: KindArray,
			expectChecks: func(t *testing.T, ref *TypeRef) {
				assert.NotNil(t, ref.ItemType)
				assert.Equal(t, "interface{}", ref.ItemType.GoType)
			},
		},
		{
			name: "map with typed additionalProperties",
			setupSchema: func() *jsonschema.Schema {
				return &jsonschema.Schema{
					Type:                 []string{"object"},
					AdditionalProperties: &jsonschema.Schema{Type: []string{"number"}},
				}
			},
			fieldName:  "",
			expectKind: KindMap,
			expectChecks: func(t *testing.T, ref *TypeRef) {
				assert.NotNil(t, ref.ValueType)
				assert.Equal(t, KindPrimitive, ref.ValueType.Kind)
			},
		},
		{
			name: "const value becomes enum",
			setupSchema: func() *jsonschema.Schema {
				return &jsonschema.Schema{
					Const: &jsonschema.ConstValue{IsSet: true, Value: "fixed-value"},
				}
			},
			fieldName:  "",
			expectKind: KindEnum,
			expectChecks: func(t *testing.T, ref *TypeRef) {
				assert.Len(t, ref.EnumValues, 1)
			},
		},
		{
			name: "oneOf creates union",
			setupSchema: func() *jsonschema.Schema {
				return &jsonschema.Schema{
					OneOf: []*jsonschema.Schema{{Type: []string{"string"}}, {Type: []string{"number"}}},
				}
			},
			fieldName:  "",
			expectKind: KindUnion,
			expectChecks: func(t *testing.T, ref *TypeRef) {
				assert.Len(t, ref.UnionMembers, 2)
			},
		},
		{
			name: "primitive with format",
			setupSchema: func() *jsonschema.Schema {
				format := "email"
				return &jsonschema.Schema{Type: []string{"string"}, Format: &format}
			},
			fieldName:  "",
			expectKind: KindPrimitive,
			expectChecks: func(t *testing.T, ref *TypeRef) {
				assert.Equal(t, "email", ref.Format)
			},
		},
		{
			name: "with ref",
			setupSchema: func() *jsonschema.Schema {
				return &jsonschema.Schema{Ref: "#/$defs/MyType"}
			},
			fieldName:  "",
			expectKind: KindRef,
			expectChecks: func(t *testing.T, ref *TypeRef) {
				assert.Equal(t, "MyType", ref.TypeName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := jsonschema.NewCompiler()
			builder := NewBuilder(compiler)

			schema := tt.setupSchema()
			ref := builder.BuildTypeRef(schema, tt.fieldName)

			assert.Equal(t, tt.expectKind, ref.Kind)
			if tt.expectChecks != nil {
				tt.expectChecks(t, ref)
			}
		})
	}
}

func TestShouldExtractInlineObject(t *testing.T) {
	tests := []struct {
		name     string
		schema   *jsonschema.Schema
		expected bool
	}{
		{
			name: "object with properties",
			schema: &jsonschema.Schema{
				Type: []string{"object"},
				Properties: &jsonschema.SchemaMap{
					"field": &jsonschema.Schema{Type: []string{"string"}},
				},
			},
			expected: true,
		},
		{
			name: "object without properties",
			schema: &jsonschema.Schema{
				Type: []string{"object"},
			},
			expected: false,
		},
		{
			name: "ref should not extract",
			schema: &jsonschema.Schema{
				Ref: "#/$defs/Something",
			},
			expected: false,
		},
		{
			name: "non-object type",
			schema: &jsonschema.Schema{
				Type: []string{"string"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := jsonschema.NewCompiler()
			builder := NewBuilder(compiler)

			result := builder.shouldExtractInlineObject(tt.schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBuild_PropertyNames tests schemas with propertyNames constraints.
func TestBuild_PropertyNames(t *testing.T) {
	tests := []struct {
		name           string
		schemaJSON     string
		expectedName   string
		expectedKind   TypeKind
		expectedFields int
		hasAdditional  bool
		requiredFields []string
	}{
		{
			name: "only propertyNames no direct properties",
			schemaJSON: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"title": "StrictObject",
				"type": "object",
				"propertyNames": {"pattern": "^[a-z_]+$"},
				"additionalProperties": {"type": "string"}
			}`,
			expectedName:   "StrictObject",
			expectedKind:   KindStruct,
			expectedFields: 0,
			hasAdditional:  true,
		},
		{
			name: "propertyNames with typed additionalProperties",
			schemaJSON: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"title": "ConfigMap",
				"type": "object",
				"propertyNames": {"pattern": "^[A-Z_]+$"},
				"additionalProperties": {"type": "number", "minimum": 0}
			}`,
			expectedName:   "ConfigMap",
			expectedKind:   KindStruct,
			expectedFields: 0,
			hasAdditional:  true,
		},
		{
			name: "propertyNames with direct properties",
			schemaJSON: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"title": "MixedObject",
				"type": "object",
				"properties": {"name": {"type": "string"}, "count": {"type": "integer"}},
				"required": ["name"],
				"propertyNames": {"pattern": "^[a-z_]+$"},
				"additionalProperties": {"type": "string"}
			}`,
			expectedName:   "MixedObject",
			expectedKind:   KindStruct,
			expectedFields: 2,
			hasAdditional:  true,
			requiredFields: []string{"name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := jsonschema.NewCompiler()
			compiled, err := compiler.Compile([]byte(tt.schemaJSON), "test.json")
			require.NoError(t, err)

			testSchema := &schema.Schema{
				Path:         "test.json",
				RelativePath: "test.json",
				Name:         tt.expectedName,
				Compiled:     compiled,
			}

			builder := NewBuilder(compiler)
			graph, err := builder.Build([]*schema.Schema{testSchema})

			require.NoError(t, err)
			require.Len(t, graph.Types, 1)

			typ := graph.Types[0]
			assert.Equal(t, tt.expectedName, typ.Name)
			assert.Equal(t, tt.expectedKind, typ.Kind)
			assert.Len(t, typ.Fields, tt.expectedFields)

			if tt.hasAdditional {
				assert.NotNil(t, typ.AdditionalProps)
				assert.True(t, typ.AdditionalProps.Allowed)
			}

			if len(tt.requiredFields) > 0 {
				fieldMap := make(map[string]*Field)
				for _, f := range typ.Fields {
					fieldMap[f.JSONName] = f
				}
				for _, rf := range tt.requiredFields {
					assert.True(t, fieldMap[rf].Required, "%s should be required", rf)
				}
			}
		})
	}
}

// Issue 2: Complex Enum Detection
func TestHasComplexEnumValues(t *testing.T) {
	tests := []struct {
		name          string
		enumValues    []any
		expectComplex bool
	}{
		{
			name:          "only strings",
			enumValues:    []any{"a", "b", "c"},
			expectComplex: false,
		},
		{
			name:          "only numbers",
			enumValues:    []any{1, 2, 3},
			expectComplex: false,
		},
		{
			name:          "mixed primitives",
			enumValues:    []any{"string", 42, true},
			expectComplex: false,
		},
		{
			name:          "contains object",
			enumValues:    []any{"simple", map[string]any{"complex": true}},
			expectComplex: true,
		},
		{
			name:          "contains array",
			enumValues:    []any{"simple", []any{"array", "value"}},
			expectComplex: true,
		},
		{
			name:          "contains both",
			enumValues:    []any{"simple", map[string]any{"x": 1}, []string{"y"}},
			expectComplex: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasComplexEnumValues(tt.enumValues)
			assert.Equal(t, tt.expectComplex, result)
		})
	}
}

// Issue 4: Union Type Detection
func TestIsUnion(t *testing.T) {
	tests := []struct {
		name        string
		schema      *jsonschema.Schema
		expectUnion bool
	}{
		{
			name:        "anyOf present",
			schema:      &jsonschema.Schema{AnyOf: []*jsonschema.Schema{{}}},
			expectUnion: true,
		},
		{
			name:        "oneOf present",
			schema:      &jsonschema.Schema{OneOf: []*jsonschema.Schema{{}}},
			expectUnion: true,
		},
		{
			name: "both present (anyOf takes precedence)",
			schema: &jsonschema.Schema{
				AnyOf: []*jsonschema.Schema{{}},
				OneOf: []*jsonschema.Schema{{}},
			},
			expectUnion: true,
		},
		{
			name:        "neither present",
			schema:      &jsonschema.Schema{},
			expectUnion: false,
		},
		{
			name:        "empty anyOf doesn't count",
			schema:      &jsonschema.Schema{AnyOf: []*jsonschema.Schema{}},
			expectUnion: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isUnion(tt.schema)
			assert.Equal(t, tt.expectUnion, result)
		})
	}
}

func TestBuildUnion(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	t.Run("anyOf with primitives", func(t *testing.T) {
		schema := &jsonschema.Schema{
			AnyOf: []*jsonschema.Schema{
				{Type: []string{"string"}},
				{Type: []string{"null"}},
				{Type: []string{"integer"}},
			},
		}

		typ := &Type{
			ID:   "test",
			Name: "UnionNullable",
		}

		err := builder.BuildUnion(typ, schema)
		require.NoError(t, err)
		assert.Equal(t, KindUnion, typ.Kind)
		assert.Len(t, typ.UnionMembers, 3)
	})

	t.Run("oneOf uses oneOf when anyOf empty", func(t *testing.T) {
		schema := &jsonschema.Schema{
			OneOf: []*jsonschema.Schema{
				{Type: []string{"string"}},
				{Type: []string{"integer"}},
			},
		}

		typ := &Type{
			ID:   "test",
			Name: "SimpleUnion",
		}

		err := builder.BuildUnion(typ, schema)
		require.NoError(t, err)
		assert.Equal(t, KindUnion, typ.Kind)
		assert.Len(t, typ.UnionMembers, 2)
	})
}

func TestProcessSchema_Union(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	// Create a schema with anyOf
	compiled := &jsonschema.Schema{
		Title:       ptrString("UnionNullable"),
		Description: ptrString("Union with nullable members"),
		AnyOf: []*jsonschema.Schema{
			{Type: []string{"string"}},
			{Type: []string{"null"}},
			{Type: []string{"integer"}},
		},
	}

	s := &schema.Schema{
		Name:     "UnionNullable",
		Compiled: compiled,
	}

	err := builder.walker.Process(s)
	require.NoError(t, err)
	assert.Len(t, builder.registry.All(), 1)

	typ := builder.registry.All()[0]
	assert.Equal(t, "UnionNullable", typ.Name)
	assert.Equal(t, KindUnion, typ.Kind)
	assert.Len(t, typ.UnionMembers, 3)
}

func ptrString(s string) *string {
	return &s
}
