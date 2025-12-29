package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kaptinlin/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLoader tests creating a new loader
func TestNewLoader(t *testing.T) {
	loader := NewLoader()

	require.NotNil(t, loader, "NewLoader should not return nil")
	assert.NotNil(t, loader.compiler, "compiler should not be nil")
}

// TestLoader_Compiler tests getting the compiler
func TestLoader_Compiler(t *testing.T) {
	loader := NewLoader()

	compiler := loader.Compiler()

	require.NotNil(t, compiler, "Compiler should not return nil")
	assert.IsType(t, &jsonschema.Compiler{}, compiler, "should return jsonschema.Compiler")
}

// TestLoader_LoadSingleFile tests loading a single JSON schema file
func TestLoader_LoadSingleFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple schema file
	schemaContent := `{
		"type": "object",
		"title": "User",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		}
	}`
	schemaPath := filepath.Join(tmpDir, "user.json")
	err := os.WriteFile(schemaPath, []byte(schemaContent), 0o644)
	require.NoError(t, err, "should write schema file")

	loader := NewLoader()
	schemas, err := loader.Load(schemaPath)

	require.NoError(t, err, "Load should not return error")
	require.Len(t, schemas, 1, "should load 1 schema")

	schema := schemas[0]
	assert.Equal(t, schemaPath, schema.Path, "Path should match")
	assert.Equal(t, "user.json", schema.RelativePath, "RelativePath should match")
	assert.Equal(t, "User", schema.Name, "Name should be derived from title")
	require.NotNil(t, schema.Compiled, "Compiled should not be nil")
}

// TestLoader_LoadSingleFile_WithoutTitle tests loading a file without title
func TestLoader_LoadSingleFile_WithoutTitle(t *testing.T) {
	tmpDir := t.TempDir()

	schemaContent := `{
		"type": "object",
		"properties": {
			"id": {"type": "string"}
		}
	}`
	schemaPath := filepath.Join(tmpDir, "product-item.json")
	err := os.WriteFile(schemaPath, []byte(schemaContent), 0o644)
	require.NoError(t, err, "should write schema file")

	loader := NewLoader()
	schemas, err := loader.Load(schemaPath)

	require.NoError(t, err, "Load should not return error")
	require.Len(t, schemas, 1, "should load 1 schema")

	schema := schemas[0]
	assert.Equal(t, "ProductItem", schema.Name, "Name should be derived from filename")
}

// TestLoader_LoadDirectory tests loading schemas from a directory
func TestLoader_LoadDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple schema files
	files := map[string]string{
		"user.json": `{"type": "object", "title": "User"}`,
		"role.json": `{"type": "object", "title": "Role"}`,
		"team.json": `{"type": "object", "title": "Team"}`,
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		err := os.WriteFile(path, []byte(content), 0o644)
		require.NoError(t, err, "should write %s", name)
	}

	loader := NewLoader()
	schemas, err := loader.Load(tmpDir)

	require.NoError(t, err, "Load should not return error")
	require.Len(t, schemas, 3, "should load 3 schemas")

	// Verify all schemas were loaded
	names := make(map[string]bool)
	for _, schema := range schemas {
		names[schema.Name] = true
	}

	assert.True(t, names["User"], "should load User")
	assert.True(t, names["Role"], "should load Role")
	assert.True(t, names["Team"], "should load Team")
}

// TestLoader_LoadDirectory_Nested tests loading schemas from nested directories
func TestLoader_LoadDirectory_Nested(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested directory structure
	eventsDir := filepath.Join(tmpDir, "events")
	payloadsDir := filepath.Join(eventsDir, "payloads")
	err := os.MkdirAll(payloadsDir, 0o755)
	require.NoError(t, err, "should create nested directories")

	// Create files at different levels
	files := map[string]string{
		filepath.Join(tmpDir, "root.json"):           `{"type": "object", "title": "Root"}`,
		filepath.Join(eventsDir, "event.json"):       `{"type": "object", "title": "Event"}`,
		filepath.Join(payloadsDir, "subscribe.json"): `{"type": "object", "title": "Subscribe"}`,
	}

	for path, content := range files {
		err := os.WriteFile(path, []byte(content), 0o644)
		require.NoError(t, err, "should write file")
	}

	loader := NewLoader()
	schemas, err := loader.Load(tmpDir)

	require.NoError(t, err, "Load should not return error")
	require.Len(t, schemas, 3, "should load 3 schemas")

	// Verify relative paths are correct
	relativePaths := make(map[string]bool)
	for _, schema := range schemas {
		relativePaths[schema.RelativePath] = true
	}

	assert.True(t, relativePaths["root.json"], "should have root.json")
	assert.True(t, relativePaths["events/event.json"], "should have events/event.json")
	assert.True(t, relativePaths["events/payloads/subscribe.json"], "should have events/payloads/subscribe.json")
}

// TestLoader_LoadDirectory_SkipNonSchema tests that non-schema files are skipped
func TestLoader_LoadDirectory_SkipNonSchema(t *testing.T) {
	tmpDir := t.TempDir()

	// Create schema and non-schema files
	files := map[string]string{
		"user.json":   `{"type": "object", "title": "User"}`,
		"readme.txt":  "This is a readme",
		"config.toml": "key = value",
		"data.csv":    "name,age\nJohn,30",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		err := os.WriteFile(path, []byte(content), 0o644)
		require.NoError(t, err, "should write %s", name)
	}

	loader := NewLoader()
	schemas, err := loader.Load(tmpDir)

	require.NoError(t, err, "Load should not return error")
	require.Len(t, schemas, 1, "should load only 1 schema")
	assert.Equal(t, "User", schemas[0].Name, "should load user.json")
}

// TestLoader_LoadWithRefs tests loading schemas with $ref
func TestLoader_LoadWithRefs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create referenced schema
	addressSchema := `{
		"type": "object",
		"title": "Address",
		"properties": {
			"street": {"type": "string"},
			"city": {"type": "string"}
		}
	}`
	addressPath := filepath.Join(tmpDir, "address.json")
	err := os.WriteFile(addressPath, []byte(addressSchema), 0o644)
	require.NoError(t, err, "should write address.json")

	// Create schema with $ref
	userSchema := `{
		"type": "object",
		"title": "User",
		"properties": {
			"name": {"type": "string"},
			"address": {"$ref": "address.json"}
		}
	}`
	userPath := filepath.Join(tmpDir, "user.json")
	err = os.WriteFile(userPath, []byte(userSchema), 0o644)
	require.NoError(t, err, "should write user.json")

	loader := NewLoader()
	schemas, err := loader.Load(tmpDir)

	require.NoError(t, err, "Load should not return error")
	require.Len(t, schemas, 2, "should load 2 schemas")

	// Verify compiler has both schemas registered
	compiler := loader.Compiler()
	assert.NotNil(t, compiler, "compiler should not be nil")

	// Both schemas should be registered in the compiler
	addressCompiled, err := compiler.GetSchema("address.json")
	require.NoError(t, err, "should get address.json schema")
	assert.NotNil(t, addressCompiled, "address.json should be registered")

	userCompiled, err := compiler.GetSchema("user.json")
	require.NoError(t, err, "should get user.json schema")
	assert.NotNil(t, userCompiled, "user.json should be registered")
}

// TestLoader_InvalidSchema tests loading an invalid schema
func TestLoader_InvalidSchema(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid JSON schema
	invalidContent := `{
		"type": "invalid-type",
		"properties": {
			"name": "not-an-object"
		}
	}`
	schemaPath := filepath.Join(tmpDir, "invalid.json")
	err := os.WriteFile(schemaPath, []byte(invalidContent), 0o644)
	require.NoError(t, err, "should write invalid schema")

	loader := NewLoader()
	_, err = loader.Load(schemaPath)

	assert.Error(t, err, "Load should return error for invalid schema")
}

// TestLoader_InvalidJSON tests loading a file with invalid JSON
func TestLoader_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	invalidContent := `{invalid json`
	schemaPath := filepath.Join(tmpDir, "invalid.json")
	err := os.WriteFile(schemaPath, []byte(invalidContent), 0o644)
	require.NoError(t, err, "should write invalid JSON")

	loader := NewLoader()
	_, err = loader.Load(schemaPath)

	assert.Error(t, err, "Load should return error for invalid JSON")
}

// TestLoader_CompilerReuse tests that the same compiler instance is reused
func TestLoader_CompilerReuse(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two schema files
	schema1 := `{"type": "object", "title": "Schema1"}`
	path1 := filepath.Join(tmpDir, "schema1.json")
	err := os.WriteFile(path1, []byte(schema1), 0o644)
	require.NoError(t, err, "should write schema1.json")

	schema2 := `{"type": "object", "title": "Schema2"}`
	path2 := filepath.Join(tmpDir, "schema2.json")
	err = os.WriteFile(path2, []byte(schema2), 0o644)
	require.NoError(t, err, "should write schema2.json")

	loader := NewLoader()
	compilerBefore := loader.Compiler()

	// Load first file
	_, err = loader.Load(path1)
	require.NoError(t, err, "Load should not return error")

	// Verify compiler is the same instance
	compilerAfter1 := loader.Compiler()
	assert.Same(t, compilerBefore, compilerAfter1, "compiler should be the same instance")

	// Load second file
	_, err = loader.Load(path2)
	require.NoError(t, err, "Load should not return error")

	// Verify compiler is still the same instance
	compilerAfter2 := loader.Compiler()
	assert.Same(t, compilerBefore, compilerAfter2, "compiler should be the same instance")

	// Both schemas should be registered in the same compiler
	schema1Compiled, errGet1 := compilerAfter2.GetSchema("schema1.json")
	require.NoError(t, errGet1, "should get schema1")
	assert.NotNil(t, schema1Compiled, "schema1 should be registered")

	schema2Compiled, errGet2 := compilerAfter2.GetSchema("schema2.json")
	require.NoError(t, errGet2, "should get schema2")
	assert.NotNil(t, schema2Compiled, "schema2 should be registered")
}

// TestLoader_LoadNonExistentPath tests loading from non-existent path
func TestLoader_LoadNonExistentPath(t *testing.T) {
	loader := NewLoader()
	_, err := loader.Load("/nonexistent/path/schema.json")

	assert.Error(t, err, "Load should return error for non-existent path")
}

// TestLoader_LoadYAMLFile tests loading a YAML schema file
func TestLoader_LoadYAMLFile(t *testing.T) {
	tmpDir := t.TempDir()

	yamlContent := `
type: object
title: Config
properties:
  name:
    type: string
  port:
    type: integer
`
	schemaPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(schemaPath, []byte(yamlContent), 0o644)
	require.NoError(t, err, "should write YAML file")

	loader := NewLoader()
	schemas, err := loader.Load(schemaPath)

	require.NoError(t, err, "Load should not return error for YAML")
	require.Len(t, schemas, 1, "should load 1 schema")

	schema := schemas[0]
	assert.Equal(t, "Config", schema.Name, "Name should be derived from title")
	assert.NotNil(t, schema.Compiled, "Compiled should not be nil")
}

// TestLoader_LoadYMLFile tests loading a .yml schema file
func TestLoader_LoadYMLFile(t *testing.T) {
	tmpDir := t.TempDir()

	ymlContent := `
type: object
title: Settings
properties:
  enabled:
    type: boolean
`
	schemaPath := filepath.Join(tmpDir, "settings.yml")
	err := os.WriteFile(schemaPath, []byte(ymlContent), 0o644)
	require.NoError(t, err, "should write YML file")

	loader := NewLoader()
	schemas, err := loader.Load(schemaPath)

	require.NoError(t, err, "Load should not return error for YML")
	require.Len(t, schemas, 1, "should load 1 schema")

	schema := schemas[0]
	assert.Equal(t, "Settings", schema.Name, "Name should be derived from title")
}

// TestLoader_LoadDirectory_MixedFormats tests loading JSON and YAML together
func TestLoader_LoadDirectory_MixedFormats(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files in different formats
	files := map[string]string{
		"user.json":   `{"type": "object", "title": "User"}`,
		"config.yaml": "type: object\ntitle: Config\n",
		"data.yml":    "type: object\ntitle: Data\n",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		err := os.WriteFile(path, []byte(content), 0o644)
		require.NoError(t, err, "should write %s", name)
	}

	loader := NewLoader()
	schemas, err := loader.Load(tmpDir)

	require.NoError(t, err, "Load should not return error")
	require.Len(t, schemas, 3, "should load 3 schemas")

	names := make(map[string]bool)
	for _, schema := range schemas {
		names[schema.Name] = true
	}

	assert.True(t, names["User"], "should load User from JSON")
	assert.True(t, names["Config"], "should load Config from YAML")
	assert.True(t, names["Data"], "should load Data from YML")
}

// TestDeriveName tests the deriveName function
func TestDeriveName(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		schemaTitle  *string
		expectedName string
	}{
		{
			name:         "with PascalCase title",
			path:         "/tmp/user.json",
			schemaTitle:  stringPtr("UserProfile"),
			expectedName: "UserProfile",
		},
		{
			name:         "with lowercase title",
			path:         "/tmp/user.json",
			schemaTitle:  stringPtr("user profile"),
			expectedName: "UserProfile",
		},
		{
			name:         "without title",
			path:         "/tmp/user-profile.json",
			schemaTitle:  nil,
			expectedName: "UserProfile",
		},
		{
			name:         "with empty title",
			path:         "/tmp/product.json",
			schemaTitle:  stringPtr(""),
			expectedName: "Product",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &jsonschema.Schema{
				Title: tt.schemaTitle,
			}

			result := deriveName(tt.path, schema)
			assert.Equal(t, tt.expectedName, result, "deriveName should return correct name")
		})
	}
}

// TestIsYAMLFile tests the isYAMLFile function
func TestIsYAMLFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/tmp/schema.yaml", true},
		{"/tmp/schema.yml", true},
		{"/tmp/schema.YAML", true},
		{"/tmp/schema.YML", true},
		{"/tmp/schema.json", false},
		{"/tmp/schema.txt", false},
		{"/tmp/schema", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isYAMLFile(tt.path)
			assert.Equal(t, tt.expected, result, "isYAMLFile should return %v for %s", tt.expected, tt.path)
		})
	}
}

// TestConvertYAMLToJSON tests YAML to JSON conversion
func TestConvertYAMLToJSON(t *testing.T) {
	yamlData := []byte(`
name: John Doe
age: 30
active: true
tags:
  - developer
  - golang
`)

	jsonData, err := convertYAMLToJSON(yamlData)

	require.NoError(t, err, "convertYAMLToJSON should not return error")
	assert.Contains(t, string(jsonData), "John Doe", "should contain name")
	assert.Contains(t, string(jsonData), "30", "should contain age")
	assert.Contains(t, string(jsonData), "true", "should contain active")
	assert.Contains(t, string(jsonData), "developer", "should contain tags")
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
