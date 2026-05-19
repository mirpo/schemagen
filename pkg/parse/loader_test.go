package parse

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFileJSON(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "schemas", "complex", "ecommerce_order.json")
	ns, err := ParseFile(path)
	require.NoError(t, err)

	assert.Equal(t, "EcommerceOrder", ns.Name)
	assert.Equal(t, "ecommerce_order.json", ns.Path)
	assert.NotNil(t, ns.Schema)
	assert.True(t, ns.Schema.Type.Has("object"))
}

func TestParseFileYAML(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "schemas", "basic", "simple.yaml")
	ns, err := ParseFile(path)
	require.NoError(t, err)

	assert.Equal(t, "Simple", ns.Name)
	assert.Equal(t, "simple.yaml", ns.Path)
	assert.NotNil(t, ns.Schema)
	assert.True(t, ns.Schema.Type.Has("object"))
}

func TestParseFileNameFromTitle(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "schemas", "anyof", "notification.json")
	ns, err := ParseFile(path)
	require.NoError(t, err)
	assert.Equal(t, "Notification", ns.Name)
}

func TestParseFileNameFromFilename(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "my_schema.json")
	err := os.WriteFile(schemaPath, []byte(`{"type": "object"}`), 0o644)
	require.NoError(t, err)

	ns, err := ParseFile(schemaPath)
	require.NoError(t, err)
	assert.Equal(t, "my_schema", ns.Name)
}

func TestParseFileUnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.txt")
	err := os.WriteFile(path, []byte(`{"type": "object"}`), 0o644)
	require.NoError(t, err)

	_, err = ParseFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported file extension")
}

func TestParseFileNonexistent(t *testing.T) {
	_, err := ParseFile("/nonexistent/path/schema.json")
	require.Error(t, err)
}

func TestParseDirRecursive(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "schemas", "events")
	schemas, err := ParseDir(dir)
	require.NoError(t, err)

	require.Len(t, schemas, 7)

	var names []string
	for _, s := range schemas {
		names = append(names, s.Name)
	}

	assert.Contains(t, names, "Event")
	assert.Contains(t, names, "EventHeader")
	assert.Contains(t, names, "EventMetadata")
	assert.Contains(t, names, "PingPayload")
	assert.Contains(t, names, "PongPayload")
	assert.Contains(t, names, "SubscribePayload")
	assert.Contains(t, names, "UnsubscribePayload")
}

func TestParseDirRelativePaths(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "schemas", "events")
	schemas, err := ParseDir(dir)
	require.NoError(t, err)

	for _, s := range schemas {
		assert.False(t, filepath.IsAbs(s.Path), "path should be relative: %s", s.Path)
	}
}

func TestParseDirSkipsNonSchemaFiles(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "valid.json"), []byte(`{"type": "object"}`), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# Readme"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "data.txt"), []byte("text"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("type: object\nproperties:\n  x:\n    type: string"), 0o644)
	require.NoError(t, err)

	schemas, err := ParseDir(dir)
	require.NoError(t, err)
	assert.Len(t, schemas, 2)

	var names []string
	for _, s := range schemas {
		names = append(names, s.Name)
	}
	assert.Contains(t, names, "valid")
	assert.Contains(t, names, "config")
}

func TestParseDirEmpty(t *testing.T) {
	dir := t.TempDir()
	schemas, err := ParseDir(dir)
	require.NoError(t, err)
	assert.Empty(t, schemas)
}

func TestParseDirNestedRelativePaths(t *testing.T) {
	dir := t.TempDir()

	subdir := filepath.Join(dir, "sub", "deep")
	err := os.MkdirAll(subdir, 0o755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "root.json"), []byte(`{"type": "object", "title": "Root"}`), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "sub", "mid.json"), []byte(`{"type": "string", "title": "Mid"}`), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subdir, "leaf.json"), []byte(`{"type": "integer", "title": "Leaf"}`), 0o644)
	require.NoError(t, err)

	schemas, err := ParseDir(dir)
	require.NoError(t, err)
	require.Len(t, schemas, 3)

	pathMap := make(map[string]string)
	for _, s := range schemas {
		pathMap[s.Name] = s.Path
	}

	assert.Equal(t, "root.json", pathMap["Root"])
	assert.Equal(t, filepath.Join("sub", "mid.json"), pathMap["Mid"])
	assert.Equal(t, filepath.Join("sub", "deep", "leaf.json"), pathMap["Leaf"])

	for _, s := range schemas {
		assert.False(t, filepath.IsAbs(s.Path), "path should be relative: %s", s.Path)
	}
}

func TestParseDirAllSchemas(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "schemas")
	schemas, err := ParseDir(dir)
	require.NoError(t, err)
	assert.Len(t, schemas, 24)
}
