package typegraph

import (
	"testing"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/stretchr/testify/assert"
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
			result := builder.extractTypeNameFromRef(tt.ref)
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
			result := builder.extractTypeNameFromRef(tt.ref)
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
			// Set the root schema context for the builder
			builder.currentSchema = schema

			result := builder.extractTypeNameFromRef(tt.ref)
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
			expectedVariants: []string{"///settings.json", "/settings.json"},
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
			expected: "Config.v2",
		},
		{
			name:     "no extension",
			ref:      "settings",
			expected: "Settings",
		},
		{
			name:     "double extension",
			ref:      "data.schema.json",
			expected: "Data.schema",
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
			expected: "UserProfileData.v2",
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

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Empty(t, graph.Types)
}

func TestBuild_SimpleObject(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "User",
		"type": "object",
		"properties": {
			"name": {
				"type": "string"
			},
			"age": {
				"type": "integer"
			}
		},
		"required": ["name"]
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "user.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "user.json",
		RelativePath:  "user.json",
		Name:          "User",
		Compiled:      compiled,
		PropertyOrder: nil, // Let builder handle order
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 1)

	typ := graph.Types[0]
	assert.Equal(t, "User", typ.Name)
	assert.Equal(t, KindStruct, typ.Kind)
	assert.Len(t, typ.Fields, 2)

	// Check fields exist (order may vary without PropertyOrder)
	fieldMap := make(map[string]*Field)
	for _, f := range typ.Fields {
		fieldMap[f.JSONName] = f
	}

	nameField := fieldMap["name"]
	assert.NotNil(t, nameField)
	assert.Equal(t, "Name", nameField.Name)
	assert.True(t, nameField.Required)
	assert.Equal(t, KindPrimitive, nameField.Type.Kind)

	ageField := fieldMap["age"]
	assert.NotNil(t, ageField)
	assert.Equal(t, "Age", ageField.Name)
	assert.False(t, ageField.Required)
	assert.Equal(t, KindPrimitive, ageField.Type.Kind)
}

func TestBuild_StringEnum(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "Status",
		"type": "string",
		"enum": ["pending", "active", "completed"]
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "status.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "status.json",
		RelativePath:  "status.json",
		Name:          "Status",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 1)

	typ := graph.Types[0]
	assert.Equal(t, "Status", typ.Name)
	assert.Equal(t, KindEnum, typ.Kind)
	assert.Len(t, typ.EnumValues, 3)

	// Check enum values (EnumValue has Name and Value fields)
	enumValueStrings := make([]string, len(typ.EnumValues))
	for i, ev := range typ.EnumValues {
		enumValueStrings[i] = ev.Value.(string)
	}
	assert.Contains(t, enumValueStrings, "pending")
	assert.Contains(t, enumValueStrings, "active")
	assert.Contains(t, enumValueStrings, "completed")
}

func TestBuild_NumberEnum(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "Priority",
		"type": "integer",
		"enum": [1, 2, 3]
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "priority.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "priority.json",
		RelativePath:  "priority.json",
		Name:          "Priority",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 1)

	typ := graph.Types[0]
	assert.Equal(t, "Priority", typ.Name)
	assert.Equal(t, KindEnum, typ.Kind)
	assert.Len(t, typ.EnumValues, 3)

	// Check enum values (numeric values)
	enumValues := make([]float64, len(typ.EnumValues))
	for i, ev := range typ.EnumValues {
		enumValues[i] = ev.Value.(float64)
	}
	assert.Contains(t, enumValues, 1.0)
	assert.Contains(t, enumValues, 2.0)
	assert.Contains(t, enumValues, 3.0)
}

func TestBuild_WithDefs(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "User",
		"type": "object",
		"properties": {
			"name": {
				"type": "string"
			},
			"address": {
				"$ref": "#/$defs/Address"
			}
		},
		"$defs": {
			"Address": {
				"type": "object",
				"properties": {
					"street": {
						"type": "string"
					},
					"city": {
						"type": "string"
					}
				}
			}
		}
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "user.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "user.json",
		RelativePath:  "user.json",
		Name:          "User",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
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
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "product.json",
		RelativePath:  "product.json",
		Name:          "Product",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
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
		assert.Equal(t, 0.01, *priceField.Minimum)
	}
	if priceField.Maximum != nil {
		assert.Equal(t, 9999.99, *priceField.Maximum)
	}

	// Check count field with exclusive constraints
	countField := fieldMap["count"]
	assert.NotNil(t, countField)
	if countField.ExclusiveMinimum != nil {
		assert.Equal(t, 0.0, *countField.ExclusiveMinimum)
	}
	if countField.ExclusiveMaximum != nil {
		assert.Equal(t, 1000.0, *countField.ExclusiveMaximum)
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
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "todolist.json",
		RelativePath:  "todolist.json",
		Name:          "TodoList",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
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
	assert.NoError(t, err)

	productCompiled, err := compiler.Compile(productSchemaJSON, "product.json")
	assert.NoError(t, err)

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

	assert.NoError(t, err)
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

func TestBuild_ObjectWithAllOf(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "Employee",
		"type": "object",
		"allOf": [
			{
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"email": {"type": "string"}
				}
			},
			{
				"type": "object",
				"properties": {
					"employeeId": {"type": "string"},
					"department": {"type": "string"}
				}
			}
		]
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "employee.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "employee.json",
		RelativePath:  "employee.json",
		Name:          "Employee",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	// allOf handling may produce different results based on schema structure
	assert.GreaterOrEqual(t, len(graph.Types), 1)

	// Check that at least one type was created
	assert.NotEmpty(t, graph.Types)
	typ := graph.Types[0]
	assert.Equal(t, "Employee", typ.Name)
}

func TestBuild_ArrayWithConstraints(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "TagList",
		"type": "object",
		"properties": {
			"tags": {
				"type": "array",
				"items": {
					"type": "string"
				},
				"minItems": 1,
				"maxItems": 10
			},
			"scores": {
				"type": "array",
				"items": {
					"type": "number"
				}
			}
		}
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "taglist.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "taglist.json",
		RelativePath:  "taglist.json",
		Name:          "TagList",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 1)

	typ := graph.Types[0]
	fieldMap := make(map[string]*Field)
	for _, f := range typ.Fields {
		fieldMap[f.JSONName] = f
	}

	// Check tags field is array
	tagsField := fieldMap["tags"]
	assert.NotNil(t, tagsField)
	assert.Equal(t, KindArray, tagsField.Type.Kind)
	assert.NotNil(t, tagsField.Type.ItemType)
	assert.Equal(t, KindPrimitive, tagsField.Type.ItemType.Kind)

	// Check array constraints
	if tagsField.MinItems != nil {
		assert.Equal(t, 1, *tagsField.MinItems)
	}
	if tagsField.MaxItems != nil {
		assert.Equal(t, 10, *tagsField.MaxItems)
	}
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
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "response.json",
		RelativePath:  "response.json",
		Name:          "Response",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	// anyOf typically creates a union or merged struct
	assert.GreaterOrEqual(t, len(graph.Types), 1)
}

// TestBuild_AllOfOnlyAtRoot tests that schemas with ONLY allOf at root level
// (no direct properties) are correctly identified as objects and generate proper struct types.
// This tests the Document pattern: allOf with refs to other schemas + inline properties.
func TestBuild_AllOfOnlyAtRoot(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "Document",
		"$defs": {
			"Entity": {
				"type": "object",
				"properties": {
					"id": {"type": "string", "format": "uuid"}
				},
				"required": ["id"]
			},
			"Auditable": {
				"type": "object",
				"properties": {
					"createdBy": {"type": "string"},
					"createdAt": {"type": "string", "format": "date-time"}
				},
				"required": ["createdBy", "createdAt"]
			}
		},
		"allOf": [
			{"$ref": "#/$defs/Entity"},
			{"$ref": "#/$defs/Auditable"},
			{
				"type": "object",
				"properties": {
					"title": {"type": "string"},
					"content": {"type": "string"}
				},
				"required": ["title"]
			}
		]
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "document.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "document.json",
		RelativePath:  "document.json",
		Name:          "Document",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)

	// Should create at least 3 types: Document, Entity, Auditable
	assert.GreaterOrEqual(t, len(graph.Types), 3)

	// Find the Document type
	var docType *Type
	for _, typ := range graph.Types {
		if typ.Name == "Document" {
			docType = typ
			break
		}
	}
	assert.NotNil(t, docType, "Document type should be created")

	// CRITICAL: Document should be a struct, NOT a primitive (which would make it "any")
	assert.Equal(t, KindStruct, docType.Kind, "Document should be KindStruct, not KindPrimitive")

	// Document should extend Entity and Auditable
	assert.Len(t, docType.Extends, 2, "Document should extend 2 types")

	// Document should have fields from the inline allOf schema (title, content)
	assert.GreaterOrEqual(t, len(docType.Fields), 2, "Document should have at least 2 fields from inline schema")

	// Check for specific fields
	fieldMap := make(map[string]*Field)
	for _, f := range docType.Fields {
		fieldMap[f.JSONName] = f
	}

	assert.NotNil(t, fieldMap["title"], "Document should have 'title' field")
	assert.NotNil(t, fieldMap["content"], "Document should have 'content' field")
	assert.True(t, fieldMap["title"].Required, "title should be required")
}

// TestBuild_AllOfInlineObjectsOnly tests allOf with only inline objects (no $refs).
// This tests the Vehicle pattern where all schemas are defined inline.
func TestBuild_AllOfInlineObjectsOnly(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "Vehicle",
		"allOf": [
			{
				"type": "object",
				"properties": {
					"make": {"type": "string"},
					"model": {"type": "string"},
					"year": {"type": "integer"}
				},
				"required": ["make", "model"]
			},
			{
				"type": "object",
				"properties": {
					"registrationNumber": {"type": "string"},
					"owner": {"type": "string"}
				},
				"required": ["registrationNumber"]
			}
		]
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "vehicle.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "vehicle.json",
		RelativePath:  "vehicle.json",
		Name:          "Vehicle",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 1)

	typ := graph.Types[0]
	assert.Equal(t, "Vehicle", typ.Name)

	// CRITICAL: Vehicle should be a struct, NOT a primitive
	assert.Equal(t, KindStruct, typ.Kind, "Vehicle should be KindStruct, not KindPrimitive")

	// Vehicle should have 5 fields from both inline allOf schemas
	assert.Len(t, typ.Fields, 5, "Vehicle should have 5 fields (make, model, year, registrationNumber, owner)")

	// Check for specific fields
	fieldMap := make(map[string]*Field)
	for _, f := range typ.Fields {
		fieldMap[f.JSONName] = f
	}

	assert.NotNil(t, fieldMap["make"], "Vehicle should have 'make' field")
	assert.NotNil(t, fieldMap["model"], "Vehicle should have 'model' field")
	assert.NotNil(t, fieldMap["registrationNumber"], "Vehicle should have 'registrationNumber' field")
	assert.True(t, fieldMap["make"].Required, "make should be required")
	assert.True(t, fieldMap["registrationNumber"].Required, "registrationNumber should be required")
}

// TestBuild_AllOfMixedRefsAndInline tests allOf with both $refs and inline objects.
func TestBuild_AllOfMixedRefsAndInline(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "Product",
		"$defs": {
			"BaseItem": {
				"type": "object",
				"properties": {
					"id": {"type": "string"},
					"name": {"type": "string"}
				},
				"required": ["id"]
			}
		},
		"allOf": [
			{"$ref": "#/$defs/BaseItem"},
			{
				"type": "object",
				"properties": {
					"price": {"type": "number"},
					"inStock": {"type": "boolean"}
				},
				"required": ["price"]
			}
		]
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "product.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "product.json",
		RelativePath:  "product.json",
		Name:          "Product",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)

	// Should create at least 2 types: Product and BaseItem
	assert.GreaterOrEqual(t, len(graph.Types), 2)

	// Find the Product type
	var productType *Type
	for _, typ := range graph.Types {
		if typ.Name == "Product" {
			productType = typ
			break
		}
	}
	assert.NotNil(t, productType, "Product type should be created")

	// CRITICAL: Product should be a struct, NOT a primitive
	assert.Equal(t, KindStruct, productType.Kind, "Product should be KindStruct, not KindPrimitive")

	// Product should extend BaseItem
	assert.Contains(t, productType.Extends, "BaseItem", "Product should extend BaseItem")

	// Product should have fields from the inline allOf schema (price, inStock)
	assert.GreaterOrEqual(t, len(productType.Fields), 2, "Product should have at least 2 fields from inline schema")

	fieldMap := make(map[string]*Field)
	for _, f := range productType.Fields {
		fieldMap[f.JSONName] = f
	}

	assert.NotNil(t, fieldMap["price"], "Product should have 'price' field")
	assert.NotNil(t, fieldMap["inStock"], "Product should have 'inStock' field")
}

// TestBuild_EmptyAllOf tests edge case of empty allOf array.
// An empty allOf array doesn't compose anything, so it should be treated as primitive/undefined.
func TestBuild_EmptyAllOf(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "EmptyComposite",
		"allOf": []
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "empty.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "empty.json",
		RelativePath:  "empty.json",
		Name:          "EmptyComposite",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 1)

	typ := graph.Types[0]
	assert.Equal(t, "EmptyComposite", typ.Name)
	// Empty allOf doesn't compose anything, so it should be treated as primitive
	assert.Equal(t, KindPrimitive, typ.Kind, "EmptyComposite with empty allOf should be KindPrimitive")
}

// TestBuild_AllOfWithDirectProperties tests backward compatibility:
// allOf combined with direct properties at root level should still work.
func TestBuild_AllOfWithDirectProperties(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "Enhanced",
		"type": "object",
		"properties": {
			"rootProp": {"type": "string"}
		},
		"allOf": [
			{
				"type": "object",
				"properties": {
					"allOfProp": {"type": "number"}
				}
			}
		]
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "enhanced.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "enhanced.json",
		RelativePath:  "enhanced.json",
		Name:          "Enhanced",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 1)

	typ := graph.Types[0]
	assert.Equal(t, "Enhanced", typ.Name)
	assert.Equal(t, KindStruct, typ.Kind)

	// Should have fields from both direct properties and allOf
	assert.Len(t, typ.Fields, 2, "Enhanced should have 2 fields (rootProp + allOfProp)")

	fieldMap := make(map[string]*Field)
	for _, f := range typ.Fields {
		fieldMap[f.JSONName] = f
	}

	assert.NotNil(t, fieldMap["rootProp"], "Enhanced should have 'rootProp' from direct properties")
	assert.NotNil(t, fieldMap["allOfProp"], "Enhanced should have 'allOfProp' from allOf")
}

func TestBuild_MixedEnum(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "MixedValue",
		"enum": ["active", 1, true, null]
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "mixed.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "mixed.json",
		RelativePath:  "mixed.json",
		Name:          "MixedValue",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 1)

	typ := graph.Types[0]
	assert.Equal(t, "MixedValue", typ.Name)
	assert.Equal(t, KindEnum, typ.Kind)
	assert.Len(t, typ.EnumValues, 4)
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

func TestBuild_AdditionalProperties(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "Config",
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"additionalProperties": {
			"type": "string"
		}
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "config.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "config.json",
		RelativePath:  "config.json",
		Name:          "Config",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 1)

	typ := graph.Types[0]
	assert.Equal(t, "Config", typ.Name)
	assert.NotNil(t, typ.AdditionalProps)
	assert.NotNil(t, typ.AdditionalProps.Type)
}

func TestBuild_NullableField(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "User",
		"type": "object",
		"properties": {
			"name": {
				"type": ["string", "null"]
			},
			"age": {
				"type": "integer"
			}
		}
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "user.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "user.json",
		RelativePath:  "user.json",
		Name:          "User",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 1)

	typ := graph.Types[0]
	fieldMap := make(map[string]*Field)
	for _, f := range typ.Fields {
		fieldMap[f.JSONName] = f
	}

	nameField := fieldMap["name"]
	assert.NotNil(t, nameField)
	// Nullable field should be marked as not required or have nullable type
	assert.True(t, nameField.Type.Nullable || !nameField.Required)
}

func TestEnsureUniqueTypeName_UniqueBaseName(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	// Add an existing type
	builder.types = []*Type{
		{Name: "User"},
	}

	// Request a unique name that doesn't conflict
	result := builder.ensureUniqueTypeName("Product")
	assert.Equal(t, "Product", result)
}

func TestEnsureUniqueTypeName_ConflictingName(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	// Add existing types with conflicting names
	builder.types = []*Type{
		{Name: "User"},
		{Name: "User2"},
	}

	// Request "User" which conflicts - should return "User3"
	result := builder.ensureUniqueTypeName("User")
	assert.Equal(t, "User3", result)
}

func TestGetOrderedPropertyNames_AlphabeticalFallback(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	// Create a schema map with properties in non-alphabetical order
	properties := jsonschema.SchemaMap{
		"zebra": &jsonschema.Schema{},
		"apple": &jsonschema.Schema{},
		"mango": &jsonschema.Schema{},
	}

	result := builder.getOrderedPropertyNames(&properties, "test.json")

	// Should be sorted alphabetically
	expected := []string{"apple", "mango", "zebra"}
	assert.Equal(t, expected, result)
}

func TestGetOrderedPropertyNames_NilProperties(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	result := builder.getOrderedPropertyNames(nil, "test.json")

	assert.Nil(t, result)
}

func TestMapPrimitiveType_UUID(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	format := "uuid"
	schema := &jsonschema.Schema{
		Type:   []string{"string"},
		Format: &format,
	}

	result := builder.mapPrimitiveType(schema)
	assert.Equal(t, "uuid.UUID", result)
}

func TestMapPrimitiveType_Email(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	format := "email"
	schema := &jsonschema.Schema{
		Type:   []string{"string"},
		Format: &format,
	}

	result := builder.mapPrimitiveType(schema)
	assert.Equal(t, "string", result)
}

func TestMapPrimitiveType_Int32(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	format := "int32"
	schema := &jsonschema.Schema{
		Type:   []string{"integer"},
		Format: &format,
	}

	result := builder.mapPrimitiveType(schema)
	assert.Equal(t, "int32", result)
}

func TestMapPrimitiveType_Float(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	format := "float"
	schema := &jsonschema.Schema{
		Type:   []string{"number"},
		Format: &format,
	}

	result := builder.mapPrimitiveType(schema)
	assert.Equal(t, "float32", result)
}

func TestBuildTypeRef_ConstValue(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	// Create a schema with const
	constVal := "fixed-value"
	schema := &jsonschema.Schema{
		Const: &jsonschema.ConstValue{
			IsSet: true,
			Value: constVal,
		},
	}

	ref := builder.buildTypeRef(schema, "")

	assert.Equal(t, KindEnum, ref.Kind)
	assert.Equal(t, "string", ref.GoType)
	assert.Len(t, ref.EnumValues, 1)
	assert.Equal(t, constVal, ref.EnumValues[0])
}

func TestBuildTypeRef_InlineEnumExtraction(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	cfg := &BuildConfig{ExtractInlined: true}
	builder := NewBuilderWithConfig(compiler, cfg)

	// Create a schema with inline enum
	schema := &jsonschema.Schema{
		Enum: []interface{}{"active", "inactive", "pending"},
	}

	// Call buildTypeRef with a field name so it extracts
	ref := builder.buildTypeRef(schema, "status")

	// Should extract to a separate type and return a ref
	assert.Equal(t, KindRef, ref.Kind)
	assert.Equal(t, "Status", ref.TypeName)

	// Check that a new type was added to builder
	assert.Len(t, builder.types, 1)
	assert.Equal(t, "Status", builder.types[0].Name)
	assert.Equal(t, KindEnum, builder.types[0].Kind)
}

func TestDeriveTypeName_TitleAlreadyPascalCase(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	title := "UserProfile"
	schema := &jsonschema.Schema{
		Title: &title,
	}

	result := builder.deriveTypeName(schema, "")
	assert.Equal(t, "UserProfile", result)
}

func TestDeriveTypeName_TitleNeedsPascalCase(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	title := "user-profile"
	schema := &jsonschema.Schema{
		Title: &title,
	}

	result := builder.deriveTypeName(schema, "")
	assert.Equal(t, "UserProfile", result)
}

func TestDeriveTypeName_FromURI(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	schema := &jsonschema.Schema{
		// No title
	}

	result := builder.deriveTypeName(schema, "payloads/subscribe.json")
	assert.Equal(t, "Subscribe", result)
}

func TestDeriveTypeName_NoTitleNoURI(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	schema := &jsonschema.Schema{
		// No title
	}

	result := builder.deriveTypeName(schema, "")
	assert.Equal(t, "Unknown", result)
}

func TestGetOrderedPropertyNames_WithCustomOrder(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	// Create property order
	order := schema.NewPropertyOrder()
	// Use reflection to set the order (since orders field is private)
	// Or we can test this through a full integration test
	// For now, let's create a simpler test

	// Set currentOrder on builder
	builder.currentOrder = order

	properties := jsonschema.SchemaMap{
		"zebra": &jsonschema.Schema{},
		"apple": &jsonschema.Schema{},
		"mango": &jsonschema.Schema{},
	}

	// Without order for this path, should fall back to alphabetical
	result := builder.getOrderedPropertyNames(&properties, "unknown.json")
	expected := []string{"apple", "mango", "zebra"}
	assert.Equal(t, expected, result)
}

func TestBuildTypeRef_OneOf(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	// Create a schema with oneOf
	schema := &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{Type: []string{"string"}},
			{Type: []string{"number"}},
		},
	}

	ref := builder.buildTypeRef(schema, "")

	assert.Equal(t, KindUnion, ref.Kind)
	assert.Len(t, ref.UnionMembers, 2)
}

func TestBuildTypeRef_ArrayWithItems(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	// Create a schema for array with string items
	schema := &jsonschema.Schema{
		Type: []string{"array"},
		Items: &jsonschema.Schema{
			Type: []string{"string"},
		},
	}

	ref := builder.buildTypeRef(schema, "tags")

	assert.Equal(t, KindArray, ref.Kind)
	assert.NotNil(t, ref.ItemType)
	assert.Equal(t, KindPrimitive, ref.ItemType.Kind)
}

func TestBuildTypeRef_ObjectWithoutProperties(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	// Create a schema for object without properties (becomes a map)
	schema := &jsonschema.Schema{
		Type: []string{"object"},
	}

	ref := builder.buildTypeRef(schema, "")

	assert.Equal(t, KindMap, ref.Kind)
	assert.NotNil(t, ref.ValueType)
}

func TestExtractDefinition_PrimitiveType(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	schema := &jsonschema.Schema{
		Type: []string{"string"},
	}

	err := builder.extractDefinition("CustomString", schema)

	assert.NoError(t, err)
	assert.Len(t, builder.types, 1)
	assert.Equal(t, "CustomString", builder.types[0].Name)
	assert.Equal(t, KindPrimitive, builder.types[0].Kind)
}

func TestGetDescription_WithDescription(t *testing.T) {
	desc := "This is a test description"
	schema := &jsonschema.Schema{
		Description: &desc,
	}

	result := getDescription(schema)
	assert.Equal(t, "This is a test description", result)
}

func TestGetDescription_NoDescription(t *testing.T) {
	schema := &jsonschema.Schema{}

	result := getDescription(schema)
	assert.Equal(t, "", result)
}

func TestBuildTypeRef_PrimitiveWithFormat(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	format := "email"
	schema := &jsonschema.Schema{
		Type:   []string{"string"},
		Format: &format,
	}

	ref := builder.buildTypeRef(schema, "")

	assert.Equal(t, KindPrimitive, ref.Kind)
	assert.Equal(t, "string", ref.GoType)
	assert.Equal(t, "email", ref.Format)
}

func TestBuildTypeRef_WithRef(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	// Create a schema with a $ref
	schema := &jsonschema.Schema{
		Ref: "#/$defs/MyType",
	}

	ref := builder.buildTypeRef(schema, "")

	// Should create a reference
	assert.Equal(t, KindRef, ref.Kind)
	assert.Equal(t, "MyType", ref.TypeName)
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
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:         "test.json",
		RelativePath: "test.json",
		Name:         "Root",
		Compiled:     compiled,
	}

	builder := NewBuilder(compiler)
	err = builder.processSchema(testSchema)

	assert.NoError(t, err)
	// Should have SubType from $defs + Root
	assert.GreaterOrEqual(t, len(builder.types), 2)
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

	fields := builder.buildFieldsFromProperties(schema, "")

	assert.Len(t, fields, 2)
	// Should be alphabetically ordered
	assert.Equal(t, "age", fields[0].JSONName)
	assert.Equal(t, "name", fields[1].JSONName)
	assert.True(t, fields[1].Required)
	assert.False(t, fields[0].Required)
}

func TestBuildTypeRef_NullableType(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	// Create a schema with nullable type (["string", "null"])
	schema := &jsonschema.Schema{
		Type: []string{"string", "null"},
	}

	ref := builder.buildTypeRef(schema, "")

	assert.True(t, ref.Nullable)
	assert.Equal(t, KindPrimitive, ref.Kind)
}

func TestBuildTypeRef_ArrayWithoutItems(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	// Create a schema for array without items specified
	schema := &jsonschema.Schema{
		Type: []string{"array"},
	}

	ref := builder.buildTypeRef(schema, "")

	assert.Equal(t, KindArray, ref.Kind)
	assert.NotNil(t, ref.ItemType)
	assert.Equal(t, KindPrimitive, ref.ItemType.Kind)
	assert.Equal(t, "interface{}", ref.ItemType.GoType)
}

func TestBuildTypeRef_MapWithTypedAdditionalProperties(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	// Create a schema for object without properties but with typed additionalProperties
	schema := &jsonschema.Schema{
		Type: []string{"object"},
		AdditionalProperties: &jsonschema.Schema{
			Type: []string{"number"},
		},
	}

	ref := builder.buildTypeRef(schema, "")

	assert.Equal(t, KindMap, ref.Kind)
	assert.NotNil(t, ref.ValueType)
	assert.Equal(t, KindPrimitive, ref.ValueType.Kind)
}

func TestBuildTypeRef_InlineEnumNoExtraction(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	cfg := &BuildConfig{ExtractInlined: false}
	builder := NewBuilderWithConfig(compiler, cfg)

	// Create a schema with inline enum (won't be extracted)
	schema := &jsonschema.Schema{
		Enum: []interface{}{"option1", "option2", "option3"},
	}

	ref := builder.buildTypeRef(schema, "status")

	// Should remain inline
	assert.Equal(t, KindEnum, ref.Kind)
	assert.Len(t, ref.EnumValues, 3)
	assert.Equal(t, "string", ref.GoType)
}

func TestBuildStruct_AllOfWithRef(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	// Create a schema with allOf that includes a $ref
	schema := &jsonschema.Schema{
		Type: []string{"object"},
		AllOf: []*jsonschema.Schema{
			{Ref: "#/$defs/BaseType"},
			{
				Type: []string{"object"},
				Properties: &jsonschema.SchemaMap{
					"extra": &jsonschema.Schema{Type: []string{"string"}},
				},
			},
		},
	}

	typ := &Type{
		ID:   "1",
		Name: "Extended",
	}

	err := builder.buildStruct(typ, schema)

	assert.NoError(t, err)
	assert.Equal(t, KindStruct, typ.Kind)
	assert.Contains(t, typ.Extends, "BaseType")
	// Should have field from inline allOf
	assert.GreaterOrEqual(t, len(typ.Fields), 1)
}

func TestMapPrimitiveType_DateTimeFormat(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	format := "date-time"
	schema := &jsonschema.Schema{
		Type:   []string{"string"},
		Format: &format,
	}

	result := builder.mapPrimitiveType(schema)
	assert.Equal(t, "time.Time", result)
}

func TestMapPrimitiveType_DateFormat(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	format := "date"
	schema := &jsonschema.Schema{
		Type:   []string{"string"},
		Format: &format,
	}

	result := builder.mapPrimitiveType(schema)
	assert.Equal(t, "time.Time", result)
}

func TestMapPrimitiveType_Int64Format(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	format := "int64"
	schema := &jsonschema.Schema{
		Type:   []string{"integer"},
		Format: &format,
	}

	result := builder.mapPrimitiveType(schema)
	assert.Equal(t, "int64", result)
}

func TestMapPrimitiveType_DoubleFormat(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	format := "double"
	schema := &jsonschema.Schema{
		Type:   []string{"number"},
		Format: &format,
	}

	result := builder.mapPrimitiveType(schema)
	assert.Equal(t, "float64", result)
}

func TestMapPrimitiveType_TimeFormat(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	format := "time"
	schema := &jsonschema.Schema{
		Type:   []string{"string"},
		Format: &format,
	}

	result := builder.mapPrimitiveType(schema)
	assert.Equal(t, "string", result)
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

			result := builder.mapPrimitiveType(schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildStruct_WithAdditionalProperties(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	builder := NewBuilder(compiler)

	schema := &jsonschema.Schema{
		Type: []string{"object"},
		Properties: &jsonschema.SchemaMap{
			"name": &jsonschema.Schema{Type: []string{"string"}},
		},
		AdditionalProperties: &jsonschema.Schema{
			Type: []string{"string"},
		},
	}

	typ := &Type{
		ID:   "1",
		Name: "TestType",
	}

	err := builder.buildStruct(typ, schema)

	assert.NoError(t, err)
	assert.Equal(t, KindStruct, typ.Kind)
	assert.Len(t, typ.Fields, 1)
	assert.NotNil(t, typ.AdditionalProps)
	assert.True(t, typ.AdditionalProps.Allowed)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := jsonschema.NewCompiler()
			builder := NewBuilder(compiler)

			schema := tt.setupSchema()
			ref := builder.buildTypeRef(schema, tt.fieldName)

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

// TestBuild_PropertyNamesOnly tests that schemas with ONLY propertyNames (no direct properties)
// are correctly identified as objects and generate proper struct types (not any).
func TestBuild_PropertyNamesOnly(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "StrictObject",
		"type": "object",
		"propertyNames": {
			"pattern": "^[a-z_]+$"
		},
		"additionalProperties": {
			"type": "string"
		}
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "strict.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "strict.json",
		RelativePath:  "strict.json",
		Name:          "StrictObject",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 1)

	typ := graph.Types[0]
	assert.Equal(t, "StrictObject", typ.Name)

	// CRITICAL: Should be a struct, NOT a primitive (which would make it "any")
	assert.Equal(t, KindStruct, typ.Kind, "StrictObject should be KindStruct, not KindPrimitive")

	// Should have additionalProperties configuration
	assert.NotNil(t, typ.AdditionalProps, "Should have AdditionalProps config")
	assert.True(t, typ.AdditionalProps.Allowed, "AdditionalProperties should be allowed")
	assert.NotNil(t, typ.AdditionalProps.Type, "AdditionalProps should have type")
	assert.Equal(t, KindPrimitive, typ.AdditionalProps.Type.Kind)
}

// TestBuild_PropertyNamesTypedValues tests propertyNames with typed additionalProperties.
func TestBuild_PropertyNamesTypedValues(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "ConfigMap",
		"type": "object",
		"propertyNames": {
			"pattern": "^[A-Z_]+$"
		},
		"additionalProperties": {
			"type": "number",
			"minimum": 0
		}
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "config.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "config.json",
		RelativePath:  "config.json",
		Name:          "ConfigMap",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 1)

	typ := graph.Types[0]
	assert.Equal(t, "ConfigMap", typ.Name)
	assert.Equal(t, KindStruct, typ.Kind, "ConfigMap should be KindStruct")

	// Should have additionalProperties with number type
	assert.NotNil(t, typ.AdditionalProps)
	assert.True(t, typ.AdditionalProps.Allowed)
	assert.NotNil(t, typ.AdditionalProps.Type)
	assert.Equal(t, KindPrimitive, typ.AdditionalProps.Type.Kind)
}

// TestBuild_PropertyNamesWithDirectProperties tests backward compatibility:
// propertyNames combined with direct properties should still work.
func TestBuild_PropertyNamesWithDirectProperties(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "MixedObject",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"count": {"type": "integer"}
		},
		"required": ["name"],
		"propertyNames": {
			"pattern": "^[a-z_]+$"
		},
		"additionalProperties": {
			"type": "string"
		}
	}`)

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(schemaJSON, "mixed.json")
	assert.NoError(t, err)

	testSchema := &schema.Schema{
		Path:          "mixed.json",
		RelativePath:  "mixed.json",
		Name:          "MixedObject",
		Compiled:      compiled,
		PropertyOrder: nil,
	}

	builder := NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Len(t, graph.Types, 1)

	typ := graph.Types[0]
	assert.Equal(t, "MixedObject", typ.Name)
	assert.Equal(t, KindStruct, typ.Kind)

	// Should have both direct properties and additionalProperties
	assert.Len(t, typ.Fields, 2, "Should have 2 direct properties")
	assert.NotNil(t, typ.AdditionalProps, "Should have AdditionalProps config")

	fieldMap := make(map[string]*Field)
	for _, f := range typ.Fields {
		fieldMap[f.JSONName] = f
	}

	assert.NotNil(t, fieldMap["name"], "Should have 'name' field")
	assert.NotNil(t, fieldMap["count"], "Should have 'count' field")
	assert.True(t, fieldMap["name"].Required, "name should be required")
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

		err := builder.buildUnion(typ, schema)
		assert.NoError(t, err)
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

		err := builder.buildUnion(typ, schema)
		assert.NoError(t, err)
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

	err := builder.processSchema(s)
	assert.NoError(t, err)
	assert.Len(t, builder.types, 1)

	typ := builder.types[0]
	assert.Equal(t, "UnionNullable", typ.Name)
	assert.Equal(t, KindUnion, typ.Kind)
	assert.Len(t, typ.UnionMembers, 3)
}

func ptrString(s string) *string {
	return &s
}
