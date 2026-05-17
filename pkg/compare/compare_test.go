package compare

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mirpo/schemagen/pkg/generation"
	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestOutput(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func setupDirs(t *testing.T) (tmpDir, generatedDir, existingDir string) {
	t.Helper()
	tmpDir = t.TempDir()
	generatedDir = filepath.Join(tmpDir, "generated")
	existingDir = filepath.Join(tmpDir, "existing")
	_ = os.MkdirAll(generatedDir, 0o755)
	_ = os.MkdirAll(existingDir, 0o755)
	return tmpDir, generatedDir, existingDir
}

func TestNormalizeLineEndings(t *testing.T) {
	tests := []struct {
		name, input, expected string
	}{
		{"CRLF to LF", "line1\r\nline2\r\nline3", "line1\nline2\nline3"},
		{"LF unchanged", "line1\nline2\nline3", "line1\nline2\nline3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeLineEndings(tt.input))
		})
	}
}

func TestCompareLangDir(t *testing.T) {
	t.Run("no drift", func(t *testing.T) {
		_, gen, exist := setupDirs(t)
		createTestOutput(t, gen, "user.ts", "export interface User { name: string; }")
		createTestOutput(t, exist, "user.ts", "export interface User { name: string; }")
		diffs, err := compareLangDir(gen, exist)
		require.NoError(t, err)
		assert.Empty(t, diffs)
	})

	t.Run("modified", func(t *testing.T) {
		_, gen, exist := setupDirs(t)
		createTestOutput(t, gen, "user.ts", "export interface User { name: string; age: number; }")
		createTestOutput(t, exist, "user.ts", "export interface User { name: string; }")
		diffs, err := compareLangDir(gen, exist)
		require.NoError(t, err)
		require.Len(t, diffs, 1)
		assert.Equal(t, StatusModified, diffs[0].Status)
	})

	t.Run("new", func(t *testing.T) {
		_, gen, exist := setupDirs(t)
		createTestOutput(t, gen, "user.ts", "export interface User { name: string; }")
		diffs, err := compareLangDir(gen, exist)
		require.NoError(t, err)
		require.Len(t, diffs, 1)
		assert.Equal(t, StatusNew, diffs[0].Status)
	})

	t.Run("deleted", func(t *testing.T) {
		_, gen, exist := setupDirs(t)
		createTestOutput(t, exist, "user.ts", "export interface User { name: string; }")
		diffs, err := compareLangDir(gen, exist)
		require.NoError(t, err)
		require.Len(t, diffs, 1)
		assert.Equal(t, StatusDeleted, diffs[0].Status)
	})

	t.Run("non existent", func(t *testing.T) {
		tmpDir := t.TempDir()
		gen := filepath.Join(tmpDir, "generated")
		_ = os.MkdirAll(gen, 0o755)
		createTestOutput(t, gen, "user.ts", "content")
		createTestOutput(t, gen, "role.ts", "content")
		diffs, err := compareLangDir(gen, filepath.Join(tmpDir, "nonexistent"))
		require.NoError(t, err)
		require.Len(t, diffs, 2)
		for _, diff := range diffs {
			assert.Equal(t, StatusNew, diff.Status)
		}
	})

	t.Run("nested directories", func(t *testing.T) {
		_, gen, exist := setupDirs(t)
		createTestOutput(t, gen, "models/user.ts", "content")
		createTestOutput(t, gen, "models/auth/role.ts", "new")
		createTestOutput(t, exist, "models/user.ts", "content")
		createTestOutput(t, exist, "models/auth/role.ts", "old")

		diffs, err := compareLangDir(gen, exist)
		require.NoError(t, err)
		require.Len(t, diffs, 1)
		assert.Equal(t, filepath.Join("models", "auth", "role.ts"), diffs[0].Path)
	})

	t.Run("line ending differences ignored", func(t *testing.T) {
		_, gen, exist := setupDirs(t)
		createTestOutput(t, gen, "user.ts", "line1\nline2")
		createTestOutput(t, exist, "user.ts", "line1\r\nline2")
		diffs, err := compareLangDir(gen, exist)
		require.NoError(t, err)
		assert.Empty(t, diffs)
	})
}

func TestRun(t *testing.T) {
	t.Run("no drift", func(t *testing.T) {
		tmpDir := t.TempDir()
		schemaDir := filepath.Join(tmpDir, "schemas")
		outputDir := filepath.Join(tmpDir, "output")
		_ = os.MkdirAll(schemaDir, 0o755)
		_ = os.MkdirAll(outputDir, 0o755)

		schemaPath := filepath.Join(schemaDir, "user.json")
		_ = os.WriteFile(schemaPath, []byte(`{"type":"object","title":"User","properties":{"name":{"type":"string"}}}`), 0o644)

		loader := schema.NewLoader()
		schemas, _ := loader.Load(schemaPath)
		_ = generation.Run(&generation.Config{
			Schemas: schemas, Compiler: loader.Compiler(), OutDir: outputDir,
			Language:         generation.LanguageTypeScript,
			DisableTimestamp: true,
			TypeScript:       &generation.TypeScriptConfig{},
		})

		result, err := Run(&Config{
			Input: schemaPath, Schemas: schemas, Loader: loader,
			Flags: &generation.GenerationFlags{OutTS: outputDir, DisableTimestamp: true},
		})
		require.NoError(t, err)
		assert.False(t, result.HasDrift)
	})

	t.Run("with drift", func(t *testing.T) {
		tmpDir := t.TempDir()
		schemaDir := filepath.Join(tmpDir, "schemas")
		outputDir := filepath.Join(tmpDir, "output")
		_ = os.MkdirAll(schemaDir, 0o755)
		_ = os.MkdirAll(outputDir, 0o755)

		schemaPath := filepath.Join(schemaDir, "user.json")
		_ = os.WriteFile(schemaPath, []byte(`{"type":"object","title":"User","properties":{"name":{"type":"string"}}}`), 0o644)
		_ = os.WriteFile(filepath.Join(outputDir, "user.ts"), []byte("// old content"), 0o644)

		loader := schema.NewLoader()
		schemas, _ := loader.Load(schemaPath)
		result, err := Run(&Config{
			Input: schemaPath, Schemas: schemas, Loader: loader,
			Flags: &generation.GenerationFlags{OutTS: outputDir},
		})
		require.NoError(t, err)
		assert.True(t, result.HasDrift)
	})

	t.Run("loads schemas automatically", func(t *testing.T) {
		tmpDir := t.TempDir()
		schemaDir := filepath.Join(tmpDir, "schemas")
		_ = os.MkdirAll(schemaDir, 0o755)

		schemaPath := filepath.Join(schemaDir, "user.json")
		_ = os.WriteFile(schemaPath, []byte(`{"type":"object","title":"User","properties":{"name":{"type":"string"}}}`), 0o644)

		result, err := Run(&Config{
			Input: schemaPath, Flags: &generation.GenerationFlags{OutTS: filepath.Join(tmpDir, "output")},
		})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("invalid schema path", func(t *testing.T) {
		_, err := Run(&Config{
			Input: "/nonexistent/schema.json", Flags: &generation.GenerationFlags{OutTS: "/tmp/output"},
		})
		require.Error(t, err)
	})
}

func TestWalkExistingFiles_StatErrorPropagated(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod 0o000 does not restrict access on Windows")
	}

	_, generatedDir, existingDir := setupDirs(t)
	createTestOutput(t, existingDir, "file.ts", "content")

	unreadableDir := filepath.Join(generatedDir, "noperm")
	require.NoError(t, os.MkdirAll(unreadableDir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(unreadableDir, 0o755) })

	readFile := func(p string) (string, error) {
		b, err := os.ReadFile(p)
		return string(b), err
	}

	createTestOutput(t, existingDir, "noperm/inner.ts", "content")

	_, err := walkExistingFiles(generatedDir, existingDir, readFile)
	assert.Error(t, err, "non-IsNotExist stat errors should propagate")
}
