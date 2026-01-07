package compare

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mirpo/schemagen/pkg/config"
	"github.com/mirpo/schemagen/pkg/generation"
	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers

func createTestSchema(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err, "should write schema file")
	return path
}

func createTestOutput(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	require.NoError(t, err, "should create directory")
	err = os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err, "should write output file")
}

// TestNormalizeLineEndings tests line ending normalization
func TestNormalizeLineEndings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "CRLF to LF",
			input:    "line1\r\nline2\r\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "LF unchanged",
			input:    "line1\nline2\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "mixed line endings",
			input:    "line1\r\nline2\nline3\r\n",
			expected: "line1\nline2\nline3\n",
		},
		{
			name:     "no line endings",
			input:    "single line",
			expected: "single line",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeLineEndings(tt.input)
			assert.Equal(t, tt.expected, result, "normalized content should match")
		})
	}
}

// TestCompareLangDir_NoDrift tests comparing directories with no differences
func TestCompareLangDir_NoDrift(t *testing.T) {
	tmpDir := t.TempDir()
	generatedDir := filepath.Join(tmpDir, "generated")
	existingDir := filepath.Join(tmpDir, "existing")

	err := os.MkdirAll(generatedDir, 0o755)
	require.NoError(t, err, "should create generated dir")
	err = os.MkdirAll(existingDir, 0o755)
	require.NoError(t, err, "should create existing dir")

	// Create identical files in both directories
	content := "export interface User { name: string; }"
	createTestOutput(t, generatedDir, "user.ts", content)
	createTestOutput(t, existingDir, "user.ts", content)

	diffs, err := compareLangDir(generatedDir, existingDir)

	require.NoError(t, err, "compareLangDir should not return error")
	assert.Empty(t, diffs, "should have no diffs")
}

// TestCompareLangDir_Modified tests detecting modified files
func TestCompareLangDir_Modified(t *testing.T) {
	tmpDir := t.TempDir()
	generatedDir := filepath.Join(tmpDir, "generated")
	existingDir := filepath.Join(tmpDir, "existing")

	err := os.MkdirAll(generatedDir, 0o755)
	require.NoError(t, err, "should create generated dir")
	err = os.MkdirAll(existingDir, 0o755)
	require.NoError(t, err, "should create existing dir")

	// Create different content in the two directories
	oldContent := "export interface User { name: string; }"
	newContent := "export interface User { name: string; age: number; }"
	createTestOutput(t, generatedDir, "user.ts", newContent)
	createTestOutput(t, existingDir, "user.ts", oldContent)

	diffs, err := compareLangDir(generatedDir, existingDir)

	require.NoError(t, err, "compareLangDir should not return error")
	require.Len(t, diffs, 1, "should have 1 diff")

	diff := diffs[0]
	assert.Equal(t, "user.ts", diff.Path, "path should match")
	assert.Equal(t, StatusModified, diff.Status, "status should be modified")
	assert.Equal(t, oldContent, diff.OldContent, "old content should match")
	assert.Equal(t, newContent, diff.NewContent, "new content should match")
}

// TestCompareLangDir_New tests detecting new files
func TestCompareLangDir_New(t *testing.T) {
	tmpDir := t.TempDir()
	generatedDir := filepath.Join(tmpDir, "generated")
	existingDir := filepath.Join(tmpDir, "existing")

	err := os.MkdirAll(generatedDir, 0o755)
	require.NoError(t, err, "should create generated dir")
	err = os.MkdirAll(existingDir, 0o755)
	require.NoError(t, err, "should create existing dir")

	// Create file only in generated directory
	newContent := "export interface User { name: string; }"
	createTestOutput(t, generatedDir, "user.ts", newContent)

	diffs, err := compareLangDir(generatedDir, existingDir)

	require.NoError(t, err, "compareLangDir should not return error")
	require.Len(t, diffs, 1, "should have 1 diff")

	diff := diffs[0]
	assert.Equal(t, "user.ts", diff.Path, "path should match")
	assert.Equal(t, StatusNew, diff.Status, "status should be new")
	assert.Empty(t, diff.OldContent, "old content should be empty")
	assert.Equal(t, newContent, diff.NewContent, "new content should match")
}

// TestCompareLangDir_Deleted tests detecting deleted files
func TestCompareLangDir_Deleted(t *testing.T) {
	tmpDir := t.TempDir()
	generatedDir := filepath.Join(tmpDir, "generated")
	existingDir := filepath.Join(tmpDir, "existing")

	err := os.MkdirAll(generatedDir, 0o755)
	require.NoError(t, err, "should create generated dir")
	err = os.MkdirAll(existingDir, 0o755)
	require.NoError(t, err, "should create existing dir")

	// Create file only in existing directory
	oldContent := "export interface User { name: string; }"
	createTestOutput(t, existingDir, "user.ts", oldContent)

	diffs, err := compareLangDir(generatedDir, existingDir)

	require.NoError(t, err, "compareLangDir should not return error")
	require.Len(t, diffs, 1, "should have 1 diff")

	diff := diffs[0]
	assert.Equal(t, "user.ts", diff.Path, "path should match")
	assert.Equal(t, StatusDeleted, diff.Status, "status should be deleted")
	assert.Equal(t, oldContent, diff.OldContent, "old content should match")
	assert.Empty(t, diff.NewContent, "new content should be empty")
}

// TestCompareLangDir_NonExistent tests when existing directory doesn't exist
func TestCompareLangDir_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	generatedDir := filepath.Join(tmpDir, "generated")
	existingDir := filepath.Join(tmpDir, "nonexistent")

	err := os.MkdirAll(generatedDir, 0o755)
	require.NoError(t, err, "should create generated dir")

	// Create files in generated directory
	content1 := "export interface User { name: string; }"
	content2 := "export interface Role { id: string; }"
	createTestOutput(t, generatedDir, "user.ts", content1)
	createTestOutput(t, generatedDir, "role.ts", content2)

	diffs, err := compareLangDir(generatedDir, existingDir)

	require.NoError(t, err, "compareLangDir should not return error")
	require.Len(t, diffs, 2, "should have 2 diffs")

	// All files should be marked as new
	for _, diff := range diffs {
		assert.Equal(t, StatusNew, diff.Status, "status should be new")
		assert.Empty(t, diff.OldContent, "old content should be empty")
		assert.NotEmpty(t, diff.NewContent, "new content should not be empty")
	}
}

// TestCompareLangDir_MultipleFiles tests comparing multiple files
func TestCompareLangDir_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	generatedDir := filepath.Join(tmpDir, "generated")
	existingDir := filepath.Join(tmpDir, "existing")

	err := os.MkdirAll(generatedDir, 0o755)
	require.NoError(t, err, "should create generated dir")
	err = os.MkdirAll(existingDir, 0o755)
	require.NoError(t, err, "should create existing dir")

	// Unchanged file
	unchangedContent := "export interface User { name: string; }"
	createTestOutput(t, generatedDir, "user.ts", unchangedContent)
	createTestOutput(t, existingDir, "user.ts", unchangedContent)

	// Modified file
	oldRoleContent := "export interface Role { id: string; }"
	newRoleContent := "export interface Role { id: string; name: string; }"
	createTestOutput(t, generatedDir, "role.ts", newRoleContent)
	createTestOutput(t, existingDir, "role.ts", oldRoleContent)

	// New file
	newTeamContent := "export interface Team { name: string; }"
	createTestOutput(t, generatedDir, "team.ts", newTeamContent)

	// Deleted file
	deletedContent := "export interface Project { id: string; }"
	createTestOutput(t, existingDir, "project.ts", deletedContent)

	diffs, err := compareLangDir(generatedDir, existingDir)

	require.NoError(t, err, "compareLangDir should not return error")
	require.Len(t, diffs, 3, "should have 3 diffs")

	// Verify each diff type
	statusCounts := make(map[FileStatus]int)
	for _, diff := range diffs {
		statusCounts[diff.Status]++
	}

	assert.Equal(t, 1, statusCounts[StatusModified], "should have 1 modified file")
	assert.Equal(t, 1, statusCounts[StatusNew], "should have 1 new file")
	assert.Equal(t, 1, statusCounts[StatusDeleted], "should have 1 deleted file")
}

// TestCompareLangDir_NestedDirectories tests comparing nested directory structures
func TestCompareLangDir_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	generatedDir := filepath.Join(tmpDir, "generated")
	existingDir := filepath.Join(tmpDir, "existing")

	err := os.MkdirAll(generatedDir, 0o755)
	require.NoError(t, err, "should create generated dir")
	err = os.MkdirAll(existingDir, 0o755)
	require.NoError(t, err, "should create existing dir")

	// Create nested files
	content1 := "export interface User { name: string; }"
	content2 := "export interface Role { id: string; }"

	createTestOutput(t, generatedDir, "models/user.ts", content1)
	createTestOutput(t, generatedDir, "models/auth/role.ts", content2)

	createTestOutput(t, existingDir, "models/user.ts", content1)
	createTestOutput(t, existingDir, "models/auth/role.ts", "old content")

	diffs, err := compareLangDir(generatedDir, existingDir)

	require.NoError(t, err, "compareLangDir should not return error")
	require.Len(t, diffs, 1, "should have 1 diff")

	diff := diffs[0]
	assert.Equal(t, filepath.Join("models", "auth", "role.ts"), diff.Path, "path should include nested structure")
	assert.Equal(t, StatusModified, diff.Status, "status should be modified")
}

// TestCompareLangDir_LineEndingDifferences tests that line ending differences are ignored
func TestCompareLangDir_LineEndingDifferences(t *testing.T) {
	tmpDir := t.TempDir()
	generatedDir := filepath.Join(tmpDir, "generated")
	existingDir := filepath.Join(tmpDir, "existing")

	err := os.MkdirAll(generatedDir, 0o755)
	require.NoError(t, err, "should create generated dir")
	err = os.MkdirAll(existingDir, 0o755)
	require.NoError(t, err, "should create existing dir")

	// Create files with different line endings but same content
	contentLF := "line1\nline2\nline3"
	contentCRLF := "line1\r\nline2\r\nline3"

	createTestOutput(t, generatedDir, "user.ts", contentLF)
	createTestOutput(t, existingDir, "user.ts", contentCRLF)

	diffs, err := compareLangDir(generatedDir, existingDir)

	require.NoError(t, err, "compareLangDir should not return error")
	assert.Empty(t, diffs, "should have no diffs (line endings normalized)")
}

// TestRun_NoDrift tests Run with no differences
func TestRun_NoDrift(t *testing.T) {
	tmpDir := t.TempDir()
	schemaDir := filepath.Join(tmpDir, "schemas")
	outputDir := filepath.Join(tmpDir, "output")

	err := os.MkdirAll(schemaDir, 0o755)
	require.NoError(t, err, "should create schema dir")
	err = os.MkdirAll(outputDir, 0o755)
	require.NoError(t, err, "should create output dir")

	// Create a simple schema
	schemaPath := createTestSchema(t, schemaDir, "user.json", `{
		"type": "object",
		"title": "User",
		"properties": {
			"name": {"type": "string"}
		}
	}`)

	// Load the schema
	loader := schema.NewLoader()
	schemas, err := loader.Load(schemaPath)
	require.NoError(t, err, "should load schema")

	// Generate the output first time
	genCfg := &generation.Config{
		Schemas:  schemas,
		Compiler: loader.Compiler(),
		OutDir:   outputDir,
		Language: generation.LanguageTypeScript,
		TypeScript: &generation.TypeScriptConfig{
			UnknownAny:           false,
			AdditionalProperties: false,
		},
	}
	err = generation.Run(genCfg)
	require.NoError(t, err, "should generate output")

	// Now compare - should have no drift
	cfg := &Config{
		Input:   schemaPath,
		Schemas: schemas,
		Loader:  loader,
		Flags: &config.GenerationFlags{
			OutTS: outputDir,
		},
		ExistingDir: tmpDir,
	}

	result, err := Run(cfg)

	require.NoError(t, err, "Run should not return error")
	require.NotNil(t, result, "result should not be nil")
	assert.False(t, result.HasDrift, "should have no drift")
	assert.Empty(t, result.Diffs, "should have no diffs")
}

// TestRun_WithDrift tests Run detecting drift
func TestRun_WithDrift(t *testing.T) {
	tmpDir := t.TempDir()
	schemaDir := filepath.Join(tmpDir, "schemas")
	outputDir := filepath.Join(tmpDir, "output")

	err := os.MkdirAll(schemaDir, 0o755)
	require.NoError(t, err, "should create schema dir")
	err = os.MkdirAll(outputDir, 0o755)
	require.NoError(t, err, "should create output dir")

	// Create a schema
	schemaPath := createTestSchema(t, schemaDir, "user.json", `{
		"type": "object",
		"title": "User",
		"properties": {
			"name": {"type": "string"}
		}
	}`)

	// Create outdated output
	createTestOutput(t, outputDir, "user.ts", "// old content")

	// Load schema
	loader := schema.NewLoader()
	schemas, err := loader.Load(schemaPath)
	require.NoError(t, err, "should load schema")

	// Compare
	cfg := &Config{
		Input:   schemaPath,
		Schemas: schemas,
		Loader:  loader,
		Flags: &config.GenerationFlags{
			OutTS: outputDir,
		},
		ExistingDir: tmpDir,
	}

	result, err := Run(cfg)

	require.NoError(t, err, "Run should not return error")
	require.NotNil(t, result, "result should not be nil")
	assert.True(t, result.HasDrift, "should have drift")
	assert.NotEmpty(t, result.Diffs, "should have diffs")
}

// TestRun_LoadSchemasAutomatically tests that Run loads schemas if not provided
func TestRun_LoadSchemasAutomatically(t *testing.T) {
	tmpDir := t.TempDir()
	schemaDir := filepath.Join(tmpDir, "schemas")

	err := os.MkdirAll(schemaDir, 0o755)
	require.NoError(t, err, "should create schema dir")

	// Create a schema
	schemaPath := createTestSchema(t, schemaDir, "user.json", `{
		"type": "object",
		"title": "User",
		"properties": {
			"name": {"type": "string"}
		}
	}`)

	// Run without pre-loaded schemas
	cfg := &Config{
		Input: schemaPath,
		Flags: &config.GenerationFlags{
			OutTS: filepath.Join(tmpDir, "output"),
		},
		ExistingDir: tmpDir,
	}

	result, err := Run(cfg)

	require.NoError(t, err, "Run should load schemas automatically")
	require.NotNil(t, result, "result should not be nil")
}

// TestRun_InvalidSchemaPath tests Run with invalid schema path
func TestRun_InvalidSchemaPath(t *testing.T) {
	cfg := &Config{
		Input: "/nonexistent/schema.json",
		Flags: &config.GenerationFlags{
			OutTS: "/tmp/output",
		},
		ExistingDir: "/tmp/existing",
	}

	_, err := Run(cfg)

	require.Error(t, err, "Run should return error for invalid schema path")
}

// TestResult_HasDrift tests the Result struct
func TestResult_HasDrift(t *testing.T) {
	tests := []struct {
		name        string
		diffs       []FileDiff
		expectDrift bool
	}{
		{
			name:        "no diffs",
			diffs:       []FileDiff{},
			expectDrift: false,
		},
		{
			name: "with diffs",
			diffs: []FileDiff{
				{Path: "user.ts", Status: StatusModified},
			},
			expectDrift: true,
		},
		{
			name: "multiple diffs",
			diffs: []FileDiff{
				{Path: "user.ts", Status: StatusModified},
				{Path: "role.ts", Status: StatusNew},
			},
			expectDrift: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{
				HasDrift: len(tt.diffs) > 0,
				Diffs:    tt.diffs,
			}

			assert.Equal(t, tt.expectDrift, result.HasDrift, "HasDrift should match expected")
			assert.Len(t, result.Diffs, len(tt.diffs), "Diffs length should match")
		})
	}
}

// TestFileStatus_Constants tests FileStatus constants
func TestFileStatus_Constants(t *testing.T) {
	assert.Equal(t, StatusModified, FileStatus("modified"), "StatusModified should match")
	assert.Equal(t, StatusNew, FileStatus("new"), "StatusNew should match")
	assert.Equal(t, StatusDeleted, FileStatus("deleted"), "StatusDeleted should match")
}
