package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverSchemas_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	schemaFile := filepath.Join(tmpDir, "user.json")
	err := os.WriteFile(schemaFile, []byte(`{"type": "object"}`), 0o644)
	require.NoError(t, err, "should create test file")

	opts := DiscoveryOptions{
		Input: schemaFile,
	}

	schemas, err := DiscoverSchemas(opts)
	require.NoError(t, err, "DiscoverSchemas() should not return error")
	require.Len(t, schemas, 1, "should discover exactly 1 schema")

	assert.Equal(t, "user.json", schemas[0].RelativePath, "RelativePath should match")
	assert.Equal(t, tmpDir, schemas[0].SchemaRoot, "SchemaRoot should match")
}

func TestDiscoverSchemas_SingleFile_InvalidExtension(t *testing.T) {
	tmpDir := t.TempDir()
	txtFile := filepath.Join(tmpDir, "user.txt")
	err := os.WriteFile(txtFile, []byte("text"), 0o644)
	require.NoError(t, err, "should create test file")

	opts := DiscoveryOptions{
		Input: txtFile,
	}

	_, err = DiscoverSchemas(opts)
	assert.Error(t, err, "should return error for invalid extension")
}

func TestDiscoverSchemas_DirectoryFlat(t *testing.T) {
	tmpDir := t.TempDir()

	files := []string{"user.json", "role.json", "team.yaml", "project.yml"}
	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		err := os.WriteFile(path, []byte(`{"type": "object"}`), 0o644)
		require.NoError(t, err, "should create test file")
	}

	txtFile := filepath.Join(tmpDir, "readme.txt")
	err := os.WriteFile(txtFile, []byte("text"), 0o644)
	require.NoError(t, err, "should create test file")

	opts := DiscoveryOptions{
		Input: tmpDir,
	}

	schemas, err := DiscoverSchemas(opts)
	require.NoError(t, err, "DiscoverSchemas() should not return error")
	require.Len(t, schemas, 4, "should discover exactly 4 schemas")

	expectedPaths := []string{"project.yml", "role.json", "team.yaml", "user.json"}
	for i, expected := range expectedPaths {
		assert.Equal(t, expected, schemas[i].RelativePath, "schema RelativePath should match")
	}
}

func TestDiscoverSchemas_DirectoryNested(t *testing.T) {
	tmpDir := t.TempDir()

	eventDir := filepath.Join(tmpDir, "events")
	payloadsDir := filepath.Join(eventDir, "payloads")
	v1Dir := filepath.Join(payloadsDir, "v1")
	v2Dir := filepath.Join(payloadsDir, "v2")

	for _, dir := range []string{eventDir, payloadsDir, v1Dir, v2Dir} {
		err := os.MkdirAll(dir, 0o755)
		require.NoError(t, err, "should create test directory")
	}

	files := map[string]string{
		filepath.Join(eventDir, "event.json"):   `{"type": "object"}`,
		filepath.Join(eventDir, "header.json"):  `{"type": "object"}`,
		filepath.Join(payloadsDir, "base.json"): `{"type": "object"}`,
		filepath.Join(v1Dir, "subscribe.json"):  `{"type": "object"}`,
		filepath.Join(v1Dir, "ping.json"):       `{"type": "object"}`,
		filepath.Join(v2Dir, "subscribe.json"):  `{"type": "object"}`,
		filepath.Join(v2Dir, "ping.json"):       `{"type": "object"}`,
	}

	for path, content := range files {
		err := os.WriteFile(path, []byte(content), 0o644)
		require.NoError(t, err, "should create test file")
	}

	opts := DiscoveryOptions{
		Input: tmpDir,
	}

	schemas, err := DiscoverSchemas(opts)
	require.NoError(t, err, "DiscoverSchemas() should not return error")
	require.Len(t, schemas, 7, "should discover exactly 7 schemas")

	for i := 1; i < len(schemas); i++ {
		assert.Less(t, schemas[i-1].RelativePath, schemas[i].RelativePath, "schemas should be sorted")
	}

	for _, schema := range schemas {
		assert.Equal(t, tmpDir, schema.SchemaRoot, "SchemaRoot should match")
	}
}

func TestDiscoverSchemas_NonExistentPath(t *testing.T) {
	opts := DiscoveryOptions{
		Input: "/nonexistent/path/to/schema.json",
	}

	_, err := DiscoverSchemas(opts)
	assert.Error(t, err, "should return error for non-existent path")
}

func TestDiscoverSchemas_CustomExtensions(t *testing.T) {
	tmpDir := t.TempDir()

	files := []string{"user.json", "role.schema", "team.txt"}
	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		err := os.WriteFile(path, []byte(`{"type": "object"}`), 0o644)
		require.NoError(t, err, "should create test file")
	}

	opts := DiscoveryOptions{
		Input:      tmpDir,
		Extensions: []string{".json", ".schema"},
	}

	schemas, err := DiscoverSchemas(opts)
	require.NoError(t, err, "DiscoverSchemas() should not return error")
	require.Len(t, schemas, 2, "should discover exactly 2 schemas")

	expectedPaths := []string{"role.schema", "user.json"}
	for i, expected := range expectedPaths {
		assert.Equal(t, expected, schemas[i].RelativePath, "schema RelativePath should match")
	}
}

func TestDiscoverSchemas_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	opts := DiscoveryOptions{
		Input: tmpDir,
	}

	schemas, err := DiscoverSchemas(opts)
	require.NoError(t, err, "DiscoverSchemas() should not return error")
	assert.Empty(t, schemas, "should discover 0 schemas in empty directory")
}

func TestDiscoverSchemas_DeterministicOrdering(t *testing.T) {
	tmpDir := t.TempDir()

	files := []string{"c.json", "a.json", "b.yaml", "d.yml"}
	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		err := os.WriteFile(path, []byte(`{"type": "object"}`), 0o644)
		require.NoError(t, err, "should create test file")
	}

	opts := DiscoveryOptions{
		Input: tmpDir,
	}

	schemas1, err := DiscoverSchemas(opts)
	require.NoError(t, err, "DiscoverSchemas() should not return error")

	schemas2, err := DiscoverSchemas(opts)
	require.NoError(t, err, "DiscoverSchemas() should not return error")

	require.Equal(t, len(schemas1), len(schemas2), "schema count should be consistent")

	for i := range schemas1 {
		assert.Equal(t, schemas1[i].RelativePath, schemas2[i].RelativePath,
			"ordering should be consistent across calls")
	}

	expectedOrder := []string{"a.json", "b.yaml", "c.json", "d.yml"}
	for i, expected := range expectedOrder {
		assert.Equal(t, expected, schemas1[i].RelativePath, "schema should match expected order")
	}
}
