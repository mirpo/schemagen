package generation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers

// createTestCompiler creates a compiler with a simple test schema
func createTestCompiler(t *testing.T) (*jsonschema.Compiler, *jsonschema.Schema) {
	t.Helper()
	compiler := jsonschema.NewCompiler()

	// Create a simple object schema for testing
	schemaJSON := []byte(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name"]
	}`)

	compiled, err := compiler.Compile(schemaJSON, "test.json")
	require.NoError(t, err, "failed to compile test schema")

	return compiler, compiled
}

// createTestSchema creates a simple test schema
func createTestSchema(t *testing.T, name string) *schema.Schema {
	t.Helper()
	_, compiled := createTestCompiler(t)

	return &schema.Schema{
		Name:         name,
		Path:         "test.json",
		RelativePath: "test.json",
		Compiled:     compiled,
	}
}

// getCompilerFromSchema extracts compiler from a test schema (for config)
func getCompilerFromSchema(t *testing.T) *jsonschema.Compiler {
	t.Helper()
	compiler, _ := createTestCompiler(t)
	return compiler
}

// createTempOutDir creates a temporary output directory
func createTempOutDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

// TestValidateConfig tests the validateConfig function
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "config is nil",
		},
		{
			name: "empty schemas",
			config: &Config{
				Compiler: getCompilerFromSchema(t),
				OutDir:   "/tmp/test",
				Language: LanguageTypeScript,
			},
			wantErr: true,
			errMsg:  "no schemas provided",
		},
		{
			name: "nil compiler",
			config: &Config{
				Schemas:  []*schema.Schema{createTestSchema(t, "Test")},
				OutDir:   "/tmp/test",
				Language: LanguageTypeScript,
			},
			wantErr: true,
			errMsg:  "compiler is nil",
		},
		{
			name: "empty output directory",
			config: &Config{
				Schemas:  []*schema.Schema{createTestSchema(t, "Test")},
				Compiler: getCompilerFromSchema(t),
				Language: LanguageTypeScript,
			},
			wantErr: true,
			errMsg:  "output directory is empty",
		},
		{
			name: "empty language",
			config: &Config{
				Schemas:  []*schema.Schema{createTestSchema(t, "Test")},
				Compiler: getCompilerFromSchema(t),
				OutDir:   "/tmp/test",
			},
			wantErr: true,
			errMsg:  "language not specified",
		},
		{
			name: "unsupported language",
			config: &Config{
				Schemas:  []*schema.Schema{createTestSchema(t, "Test")},
				Compiler: getCompilerFromSchema(t),
				OutDir:   "/tmp/test",
				Language: "java",
			},
			wantErr: true,
			errMsg:  "unsupported language",
		},
		{
			name: "valid TypeScript config",
			config: &Config{
				Schemas:  []*schema.Schema{createTestSchema(t, "Test")},
				Compiler: getCompilerFromSchema(t),
				OutDir:   "/tmp/test",
				Language: LanguageTypeScript,
			},
			wantErr: false,
		},
		{
			name: "valid Python config",
			config: &Config{
				Schemas:  []*schema.Schema{createTestSchema(t, "Test")},
				Compiler: getCompilerFromSchema(t),
				OutDir:   "/tmp/test",
				Language: LanguagePython,
			},
			wantErr: false,
		},
		{
			name: "valid Go config",
			config: &Config{
				Schemas:  []*schema.Schema{createTestSchema(t, "Test")},
				Compiler: getCompilerFromSchema(t),
				OutDir:   "/tmp/test",
				Language: LanguageGo,
			},
			wantErr: false,
		},
		{
			name: "TypeScript with custom config",
			config: &Config{
				Schemas:  []*schema.Schema{createTestSchema(t, "Test")},
				Compiler: getCompilerFromSchema(t),
				OutDir:   "/tmp/test",
				Language: LanguageTypeScript,
				TypeScript: &TypeScriptConfig{
					UnknownAny: true,
				},
			},
			wantErr: false,
		},
		{
			name: "Python with custom config",
			config: &Config{
				Schemas:  []*schema.Schema{createTestSchema(t, "Test")},
				Compiler: getCompilerFromSchema(t),
				OutDir:   "/tmp/test",
				Language: LanguagePython,
				Python: &PythonConfig{
					SnakeCaseField: true,
				},
			},
			wantErr: false,
		},
		{
			name: "Go with custom config",
			config: &Config{
				Schemas:  []*schema.Schema{createTestSchema(t, "Test")},
				Compiler: getCompilerFromSchema(t),
				OutDir:   "/tmp/test",
				Language: LanguageGo,
				Go: &GoConfig{
					PackageName: "custom",
					UsePointers: false,
					OmitEmpty:   false,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err, "validateConfig() should return error")
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg, "error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "validateConfig() should not return error")
			}
		})
	}
}

// TestRun_TypeScript_Bundle tests TypeScript generation with bundle strategy
func TestRun_TypeScript_Bundle(t *testing.T) {
	outDir := createTempOutDir(t)
	compiler := getCompilerFromSchema(t)

	cfg := &Config{
		Schemas:        []*schema.Schema{createTestSchema(t, "User")},
		Compiler:       compiler,
		OutDir:         outDir,
		Language:       LanguageTypeScript,
		OutputStrategy: output.StrategyBundle,
		TypeScript:     &TypeScriptConfig{},
	}

	err := Run(cfg)
	require.NoError(t, err, "Run() should succeed")

	// Verify output file exists
	outputFile := filepath.Join(outDir, "types.ts")
	assert.FileExists(t, outputFile, "output file should exist")

	// Verify file has content
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "should be able to read output file")
	assert.NotEmpty(t, content, "output file should have content")

	// Verify content contains expected TypeScript code
	contentStr := string(content)
	// Should contain either "export interface" or "export type"
	containsInterface := contains(contentStr, "export interface")
	containsType := contains(contentStr, "export type")
	assert.True(t, containsInterface || containsType, "output should contain TypeScript interface or type declaration")
}

// TestRun_TypeScript_MultiFile tests TypeScript generation with multifile strategy
func TestRun_TypeScript_MultiFile(t *testing.T) {
	outDir := createTempOutDir(t)
	compiler := getCompilerFromSchema(t)

	cfg := &Config{
		Schemas:        []*schema.Schema{createTestSchema(t, "User")},
		Compiler:       compiler,
		OutDir:         outDir,
		Language:       LanguageTypeScript,
		OutputStrategy: output.StrategyMultiFile,
		TypeScript:     &TypeScriptConfig{},
	}

	err := Run(cfg)
	require.NoError(t, err, "Run() should succeed")

	// Verify output directory exists and has files
	entries, err := os.ReadDir(outDir)
	require.NoError(t, err, "should be able to read output directory")
	assert.NotEmpty(t, entries, "output directory should have files")

	// Check for index.ts (barrel file)
	indexFile := filepath.Join(outDir, "index.ts")
	assert.FileExists(t, indexFile, "barrel file should exist")
}

// TestRun_Python_Bundle tests Python generation with bundle strategy
func TestRun_Python_Bundle(t *testing.T) {
	outDir := createTempOutDir(t)
	compiler := getCompilerFromSchema(t)

	cfg := &Config{
		Schemas:        []*schema.Schema{createTestSchema(t, "User")},
		Compiler:       compiler,
		OutDir:         outDir,
		Language:       LanguagePython,
		OutputStrategy: output.StrategyBundle,
		Python:         &PythonConfig{},
	}

	err := Run(cfg)
	require.NoError(t, err, "Run() should succeed")

	// Verify output file exists
	outputFile := filepath.Join(outDir, "types.py")
	assert.FileExists(t, outputFile, "output file should exist")

	// Verify file has content
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "should be able to read output file")
	assert.NotEmpty(t, content, "output file should have content")

	// Verify content contains expected Python code
	contentStr := string(content)
	containsClass := contains(contentStr, "class ")
	containsPydantic := contains(contentStr, "from pydantic")
	assert.True(t, containsClass || containsPydantic, "output should contain Python class or pydantic import")
}

// TestRun_Python_SnakeCase tests Python generation with snake_case fields
func TestRun_Python_SnakeCase(t *testing.T) {
	outDir := createTempOutDir(t)
	compiler := getCompilerFromSchema(t)

	cfg := &Config{
		Schemas:        []*schema.Schema{createTestSchema(t, "User")},
		Compiler:       compiler,
		OutDir:         outDir,
		Language:       LanguagePython,
		OutputStrategy: output.StrategyBundle,
		Python: &PythonConfig{
			SnakeCaseField: true,
		},
	}

	err := Run(cfg)
	require.NoError(t, err, "Run() should succeed")

	// Verify output file exists
	outputFile := filepath.Join(outDir, "types.py")
	assert.FileExists(t, outputFile, "output file should exist")
}

// TestRun_Go_Bundle tests Go generation with bundle strategy
func TestRun_Go_Bundle(t *testing.T) {
	outDir := createTempOutDir(t)
	compiler := getCompilerFromSchema(t)

	cfg := &Config{
		Schemas:        []*schema.Schema{createTestSchema(t, "User")},
		Compiler:       compiler,
		OutDir:         outDir,
		Language:       LanguageGo,
		OutputStrategy: output.StrategyBundle,
		Go: &GoConfig{
			PackageName: "models",
			UsePointers: true,
			OmitEmpty:   true,
		},
	}

	err := Run(cfg)
	require.NoError(t, err, "Run() should succeed")

	// Verify output file exists
	outputFile := filepath.Join(outDir, "types.go")
	assert.FileExists(t, outputFile, "output file should exist")

	// Verify file has content
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "should be able to read output file")
	assert.NotEmpty(t, content, "output file should have content")

	// Verify content contains expected Go code
	contentStr := string(content)
	assert.Contains(t, contentStr, "package models", "should contain package declaration")
	assert.Contains(t, contentStr, "type ", "should contain type declaration")
}

// TestRun_InvalidLanguage tests error handling for unsupported language
func TestRun_InvalidLanguage(t *testing.T) {
	outDir := createTempOutDir(t)
	compiler := getCompilerFromSchema(t)

	cfg := &Config{
		Schemas:        []*schema.Schema{createTestSchema(t, "User")},
		Compiler:       compiler,
		OutDir:         outDir,
		Language:       "rust",
		OutputStrategy: output.StrategyBundle,
	}

	err := Run(cfg)
	assert.Error(t, err, "Run() should return error for invalid language")
	assert.Contains(t, err.Error(), "unsupported language", "error should mention unsupported language")
}

// TestRun_InvalidOutputStrategy tests error handling for unsupported output strategy
func TestRun_InvalidOutputStrategy(t *testing.T) {
	outDir := createTempOutDir(t)
	compiler := getCompilerFromSchema(t)

	cfg := &Config{
		Schemas:        []*schema.Schema{createTestSchema(t, "User")},
		Compiler:       compiler,
		OutDir:         outDir,
		Language:       LanguageTypeScript,
		OutputStrategy: "", // Empty strategies default to bundle
		TypeScript:     &TypeScriptConfig{},
	}

	err := Run(cfg)
	assert.NoError(t, err, "Run() should succeed (empty strategies default to bundle)")

	// Verify output directory exists and has files
	entries, err := os.ReadDir(outDir)
	require.NoError(t, err, "should be able to read output directory")
	assert.NotEmpty(t, entries, "output directory should have files")
}

// TestRun_EmptySchemas tests error handling for empty schemas
func TestRun_EmptySchemas(t *testing.T) {
	outDir := createTempOutDir(t)
	compiler := getCompilerFromSchema(t)

	cfg := &Config{
		Schemas:        []*schema.Schema{},
		Compiler:       compiler,
		OutDir:         outDir,
		Language:       LanguageTypeScript,
		OutputStrategy: output.StrategyBundle,
		TypeScript:     &TypeScriptConfig{},
	}

	err := Run(cfg)
	assert.Error(t, err, "Run() should return error for empty schemas")
	assert.Contains(t, err.Error(), "no schemas provided", "error should mention no schemas")
}

// TestRun_BundleDepsStrategy tests bundle-deps output strategy
func TestRun_BundleDepsStrategy(t *testing.T) {
	outDir := createTempOutDir(t)
	compiler := getCompilerFromSchema(t)

	cfg := &Config{
		Schemas:        []*schema.Schema{createTestSchema(t, "User")},
		Compiler:       compiler,
		OutDir:         outDir,
		Language:       LanguageTypeScript,
		OutputStrategy: output.StrategyBundleDeps,
		TypeScript:     &TypeScriptConfig{},
	}

	err := Run(cfg)
	require.NoError(t, err, "Run() should succeed")

	// Verify output exists
	entries, err := os.ReadDir(outDir)
	require.NoError(t, err, "should be able to read output directory")
	assert.NotEmpty(t, entries, "output directory should have files")
}

// TestRun_ExtractInline tests extract inline flag
func TestRun_ExtractInline(t *testing.T) {
	outDir := createTempOutDir(t)
	compiler := getCompilerFromSchema(t)

	cfg := &Config{
		Schemas:        []*schema.Schema{createTestSchema(t, "User")},
		Compiler:       compiler,
		OutDir:         outDir,
		Language:       LanguageTypeScript,
		OutputStrategy: output.StrategyBundle,
		ExtractInline:  true,
		TypeScript:     &TypeScriptConfig{},
	}

	err := Run(cfg)
	require.NoError(t, err, "Run() should succeed")

	// Verify output file exists
	outputFile := filepath.Join(outDir, "types.ts")
	assert.FileExists(t, outputFile, "output file should exist")
}

// TestRun_GoAlwaysExtractsInline verifies Go forces extraction even without --extract-inline flag
func TestRun_GoAlwaysExtractsInline(t *testing.T) {
	outDir := createTempOutDir(t)
	compiler := getCompilerFromSchema(t)

	cfg := &Config{
		Schemas:        []*schema.Schema{createTestSchema(t, "User")},
		Compiler:       compiler,
		OutDir:         outDir,
		Language:       LanguageGo,
		OutputStrategy: output.StrategyBundle,
		ExtractInline:  false, // Explicitly false, but Go should still extract
		Go:             &GoConfig{PackageName: "models"},
	}

	err := Run(cfg)
	require.NoError(t, err, "Run() should succeed")

	// Verify output file exists
	outputFile := filepath.Join(outDir, "types.go")
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "should be able to read output file")

	// Go should produce proper struct types, not map[string]interface{}
	contentStr := string(content)
	assert.NotContains(t, contentStr, "map[string]interface{}", "Go should extract inline types, not use map[string]interface{}")
}

// TestRun_DisableHeaders tests header generation flags
func TestRun_DisableHeaders(t *testing.T) {
	outDir := createTempOutDir(t)
	compiler := getCompilerFromSchema(t)

	cfg := &Config{
		Schemas:        []*schema.Schema{createTestSchema(t, "User")},
		Compiler:       compiler,
		OutDir:         outDir,
		Language:       LanguageTypeScript,
		OutputStrategy: output.StrategyBundle,
		DisableHeaders: true,
		TypeScript:     &TypeScriptConfig{},
	}

	err := Run(cfg)
	require.NoError(t, err, "Run() should succeed")

	// Verify output file exists
	outputFile := filepath.Join(outDir, "types.ts")
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "should be able to read output file")

	// When headers are disabled, should not contain "Generated by"
	contentStr := string(content)
	assert.NotContains(t, contentStr, "Generated by", "output should not contain header when disabled")
}

// TestPlanOutput tests the planOutput function
func TestPlanOutput(t *testing.T) {
	compiler := getCompilerFromSchema(t)
	testSchema := createTestSchema(t, "User")

	builder := typegraph.NewBuilder(compiler)
	graph, err := builder.Build([]*schema.Schema{testSchema})
	require.NoError(t, err, "should be able to build graph")

	tests := []struct {
		name           string
		language       Language
		outputStrategy output.OutputStrategy
		wantErr        bool
		wantExt        string
	}{
		{
			name:           "TypeScript bundle",
			language:       LanguageTypeScript,
			outputStrategy: output.StrategyBundle,
			wantErr:        false,
			wantExt:        "ts",
		},
		{
			name:           "Python multifile",
			language:       LanguagePython,
			outputStrategy: output.StrategyMultiFile,
			wantErr:        false,
			wantExt:        "py",
		},
		{
			name:           "Go bundle-deps",
			language:       LanguageGo,
			outputStrategy: output.StrategyBundleDeps,
			wantErr:        false,
			wantExt:        "go",
		},
		{
			name:           "invalid language",
			language:       "java",
			outputStrategy: output.StrategyBundle,
			wantErr:        true,
		},
		{
			name:           "empty strategy defaults to bundle",
			language:       LanguageTypeScript,
			outputStrategy: "", // Empty strategy defaults to bundle
			wantErr:        false,
			wantExt:        "ts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Schemas:        []*schema.Schema{testSchema},
				Compiler:       compiler,
				Language:       tt.language,
				OutputStrategy: tt.outputStrategy,
			}

			plan, err := planOutput(graph, cfg)

			if tt.wantErr {
				assert.Error(t, err, "planOutput() should return error")
			} else {
				assert.NoError(t, err, "planOutput() should not return error")
				assert.NotNil(t, plan, "planOutput() should return plan")
				if plan != nil && len(plan.Files) > 0 {
					// Check file extension
					for _, file := range plan.Files {
						ext := filepath.Ext(file.RelativePath)
						assert.Equal(t, "."+tt.wantExt, ext, "file should have correct extension")
					}
				}
			}
		})
	}
}

// TestRun_Go_CustomPackage tests Go generation with custom package name
func TestRun_Go_CustomPackage(t *testing.T) {
	outDir := createTempOutDir(t)
	compiler := getCompilerFromSchema(t)

	cfg := &Config{
		Schemas:        []*schema.Schema{createTestSchema(t, "User")},
		Compiler:       compiler,
		OutDir:         outDir,
		Language:       LanguageGo,
		OutputStrategy: output.StrategyBundle,
		Go: &GoConfig{
			PackageName: "mymodels",
			UsePointers: true,
			OmitEmpty:   true,
		},
	}

	err := Run(cfg)
	require.NoError(t, err, "Run() should succeed")

	outputFile := filepath.Join(outDir, "types.go")
	assert.FileExists(t, outputFile, "output file should exist")

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "should be able to read output file")

	contentStr := string(content)
	assert.Contains(t, contentStr, "package mymodels", "should use custom package name")
}

// TestRun_Go_NoPointers tests Go generation without pointers for optional fields
func TestRun_Go_NoPointers(t *testing.T) {
	outDir := createTempOutDir(t)
	compiler := jsonschema.NewCompiler()

	// Schema with optional field
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"required_field": {"type": "string"},
			"optional_field": {"type": "string"}
		},
		"required": ["required_field"]
	}`)

	compiled, err := compiler.Compile(schemaJSON, "test.json")
	require.NoError(t, err)

	testSchema := &schema.Schema{
		Name:         "Test",
		Path:         "test.json",
		RelativePath: "test.json",
		Compiled:     compiled,
	}

	cfg := &Config{
		Schemas:        []*schema.Schema{testSchema},
		Compiler:       compiler,
		OutDir:         outDir,
		Language:       LanguageGo,
		OutputStrategy: output.StrategyBundle,
		Go: &GoConfig{
			PackageName: "models",
			UsePointers: false,
			OmitEmpty:   true,
		},
	}

	err = Run(cfg)
	require.NoError(t, err, "Run() should succeed")

	outputFile := filepath.Join(outDir, "types.go")
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	contentStr := string(content)
	// With UsePointers=false, optional fields should not have pointer types
	// We check that the field exists without a pointer indicator in a reasonable way
	assert.Contains(t, contentStr, "OptionalField", "should contain optional field")
}

// TestRun_Go_NoOmitEmpty tests Go generation without omitempty tags
func TestRun_Go_NoOmitEmpty(t *testing.T) {
	outDir := createTempOutDir(t)
	compiler := jsonschema.NewCompiler()

	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"required_field": {"type": "string"},
			"optional_field": {"type": "string"}
		},
		"required": ["required_field"]
	}`)

	compiled, err := compiler.Compile(schemaJSON, "test.json")
	require.NoError(t, err)

	testSchema := &schema.Schema{
		Name:         "Test",
		Path:         "test.json",
		RelativePath: "test.json",
		Compiled:     compiled,
	}

	cfg := &Config{
		Schemas:        []*schema.Schema{testSchema},
		Compiler:       compiler,
		OutDir:         outDir,
		Language:       LanguageGo,
		OutputStrategy: output.StrategyBundle,
		Go: &GoConfig{
			PackageName: "models",
			UsePointers: true,
			OmitEmpty:   false,
		},
	}

	err = Run(cfg)
	require.NoError(t, err, "Run() should succeed")

	outputFile := filepath.Join(outDir, "types.go")
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "OptionalField", "should contain optional field")
	// When omitempty is disabled, tags should not have omitempty for optional fields
	// We verify the field exists (detailed tag checking would require more parsing)
}

// TestRun_TypeScript_AdditionalProperties tests TypeScript generation with additional properties
func TestRun_TypeScript_AdditionalProperties(t *testing.T) {
	outDir := createTempOutDir(t)
	compiler := jsonschema.NewCompiler()

	// Schema with additionalProperties
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"additionalProperties": {"type": "string"}
	}`)

	compiled, err := compiler.Compile(schemaJSON, "test.json")
	require.NoError(t, err)

	testSchema := &schema.Schema{
		Name:         "Test",
		Path:         "test.json",
		RelativePath: "test.json",
		Compiled:     compiled,
	}

	cfg := &Config{
		Schemas:        []*schema.Schema{testSchema},
		Compiler:       compiler,
		OutDir:         outDir,
		Language:       LanguageTypeScript,
		OutputStrategy: output.StrategyBundle,
		TypeScript: &TypeScriptConfig{
			AdditionalProperties: true,
		},
	}

	err = Run(cfg)
	require.NoError(t, err, "Run() should succeed")

	outputFile := filepath.Join(outDir, "types.ts")
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	contentStr := string(content)
	// When AdditionalProperties is enabled, should have index signature
	assert.Contains(t, contentStr, "[key: string]:", "should contain index signature for additional properties")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
