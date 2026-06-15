package compare

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mirpo/schemagen/pkg/parse"
	"github.com/mirpo/schemagen/pkg/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestOutput(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
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

func TestCompareFiles(t *testing.T) {
	t.Run("no drift", func(t *testing.T) {
		dir := t.TempDir()
		createTestOutput(t, dir, "user.ts", "export interface User { name: string; }")

		generated := map[string][]byte{
			"user.ts": []byte("export interface User { name: string; }"),
		}
		diffs, err := compareFiles(generated, dir)
		require.NoError(t, err)
		assert.Empty(t, diffs)
	})

	t.Run("modified", func(t *testing.T) {
		dir := t.TempDir()
		createTestOutput(t, dir, "user.ts", "export interface User { name: string; }")

		generated := map[string][]byte{
			"user.ts": []byte("export interface User { name: string; age: number; }"),
		}
		diffs, err := compareFiles(generated, dir)
		require.NoError(t, err)
		require.Len(t, diffs, 1)
		assert.Equal(t, StatusModified, diffs[0].Status)
	})

	t.Run("new file", func(t *testing.T) {
		dir := t.TempDir()

		generated := map[string][]byte{
			"user.ts": []byte("export interface User {}"),
		}
		diffs, err := compareFiles(generated, dir)
		require.NoError(t, err)
		require.Len(t, diffs, 1)
		assert.Equal(t, StatusNew, diffs[0].Status)
	})

	t.Run("deleted file", func(t *testing.T) {
		dir := t.TempDir()
		createTestOutput(t, dir, "user.ts", "export interface User { name: string; }")

		generated := map[string][]byte{}
		diffs, err := compareFiles(generated, dir)
		require.NoError(t, err)
		require.Len(t, diffs, 1)
		assert.Equal(t, StatusDeleted, diffs[0].Status)
	})

	t.Run("non existent dir treats all as new", func(t *testing.T) {
		generated := map[string][]byte{
			"user.ts": []byte("content"),
			"role.ts": []byte("content"),
		}
		diffs, err := compareFiles(generated, "/nonexistent/dir")
		require.NoError(t, err)
		require.Len(t, diffs, 2)
		for _, diff := range diffs {
			assert.Equal(t, StatusNew, diff.Status)
		}
	})

	t.Run("line ending differences ignored", func(t *testing.T) {
		dir := t.TempDir()
		createTestOutput(t, dir, "user.ts", "line1\r\nline2")

		generated := map[string][]byte{
			"user.ts": []byte("line1\nline2"),
		}
		diffs, err := compareFiles(generated, dir)
		require.NoError(t, err)
		assert.Empty(t, diffs)
	})

	t.Run("nested paths", func(t *testing.T) {
		dir := t.TempDir()
		createTestOutput(t, dir, "models/user.ts", "old")

		generated := map[string][]byte{
			"models/user.ts": []byte("new"),
		}
		diffs, err := compareFiles(generated, dir)
		require.NoError(t, err)
		require.Len(t, diffs, 1)
		assert.Equal(t, "models/user.ts", diffs[0].Path)
		assert.Equal(t, StatusModified, diffs[0].Status)
	})

	t.Run("mixed drift collects every status", func(t *testing.T) {
		dir := t.TempDir()
		createTestOutput(t, dir, "same.ts", "identical")
		createTestOutput(t, dir, "changed.ts", "old")
		createTestOutput(t, dir, "gone.ts", "removed")

		generated := map[string][]byte{
			"same.ts":    []byte("identical"),
			"changed.ts": []byte("new"),
			"added.ts":   []byte("brand new"),
		}
		diffs, err := compareFiles(generated, dir)
		require.NoError(t, err)

		// Diff order is non-deterministic (map iteration), so assert as a set.
		got := make(map[string]FileStatus, len(diffs))
		for _, d := range diffs {
			got[d.Path] = d.Status
		}
		assert.Equal(t, map[string]FileStatus{
			"added.ts":   StatusNew,
			"changed.ts": StatusModified,
			"gone.ts":    StatusDeleted,
		}, got)
	})
}

func TestGenerateAndDiffTarget_WrapsErrorWithTargetContext(t *testing.T) {
	// Empty schemas make pipeline.Run fail validation; the surfaced error
	// must carry the target's dir and lang so the failing target is known.
	target := pipeline.GenerationTarget{Dir: "out/ts", Lang: pipeline.LanguageTypeScript}
	_, err := generateAndDiffTarget(nil, target, &pipeline.GenerationFlags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out/ts")
	assert.Contains(t, err.Error(), string(pipeline.LanguageTypeScript))
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

		schemas, _ := parse.Load(schemaPath)
		_ = pipeline.Run(&pipeline.Config{
			Schemas: schemas, OutDir: outputDir,
			Language:         pipeline.LanguageTypeScript,
			DisableTimestamp: true,
			TypeScript:       &pipeline.TypeScriptConfig{},
		})

		result, err := Run(&Config{
			Input: schemaPath,
			Flags: &pipeline.GenerationFlags{OutTS: outputDir, DisableTimestamp: true},
		})
		require.NoError(t, err)
		assert.Empty(t, result.Diffs)
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

		result, err := Run(&Config{
			Input: schemaPath,
			Flags: &pipeline.GenerationFlags{OutTS: outputDir},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, result.Diffs)
	})

	t.Run("loads schemas automatically", func(t *testing.T) {
		tmpDir := t.TempDir()
		schemaDir := filepath.Join(tmpDir, "schemas")
		_ = os.MkdirAll(schemaDir, 0o755)

		schemaPath := filepath.Join(schemaDir, "user.json")
		_ = os.WriteFile(schemaPath, []byte(`{"type":"object","title":"User","properties":{"name":{"type":"string"}}}`), 0o644)

		result, err := Run(&Config{
			Input: schemaPath, Flags: &pipeline.GenerationFlags{OutTS: filepath.Join(tmpDir, "output")},
		})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("invalid schema path", func(t *testing.T) {
		_, err := Run(&Config{
			Input: "/nonexistent/schema.json", Flags: &pipeline.GenerationFlags{OutTS: "/tmp/output"},
		})
		require.Error(t, err)
	})
}
