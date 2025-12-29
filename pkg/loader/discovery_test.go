package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverSchemas_SingleFile(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "user.json")
	require.NoError(t, os.WriteFile(file, []byte(`{}`), 0o644))

	schemas, err := DiscoverSchemas(DiscoveryOptions{Input: file})
	require.NoError(t, err)
	require.Len(t, schemas, 1)

	assert.Equal(t, "user.json", schemas[0].RelativePath)
	assert.Equal(t, tmp, schemas[0].SchemaRoot)
	assert.Equal(t, file, schemas[0].AbsolutePath)
}

func TestDiscoverSchemas_SingleFile_InvalidExtension(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "user.txt")
	require.NoError(t, os.WriteFile(file, []byte(`{}`), 0o644))

	_, err := DiscoverSchemas(DiscoveryOptions{Input: file})
	assert.Error(t, err)
}

func TestDiscoverSchemas_DirectoryRecursive(t *testing.T) {
	tmp := t.TempDir()

	files := []string{
		"a.json",
		"nested/b.yaml",
		"nested/deep/c.yml",
	}

	for _, f := range files {
		full := filepath.Join(tmp, f)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(`{}`), 0o644))
	}

	schemas, err := DiscoverSchemas(DiscoveryOptions{Input: tmp})
	require.NoError(t, err)
	require.Len(t, schemas, 3)

	assert.Equal(t, []string{
		"a.json",
		"nested/b.yaml",
		"nested/deep/c.yml",
	}, extractRelativePaths(schemas))
}

func TestDiscoverSchemas_DeterministicOrder(t *testing.T) {
	tmp := t.TempDir()

	files := []string{"c.json", "a.json", "b.yaml"}
	for _, f := range files {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, f), []byte(`{}`), 0o644))
	}

	s1, err := DiscoverSchemas(DiscoveryOptions{Input: tmp})
	require.NoError(t, err)

	s2, err := DiscoverSchemas(DiscoveryOptions{Input: tmp})
	require.NoError(t, err)

	assert.Equal(t, extractRelativePaths(s1), extractRelativePaths(s2))
}

func TestDiscoverSchemas_CustomExtensions(t *testing.T) {
	tmp := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmp, "a.schema"), []byte(`{}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "b.json"), []byte(`{}`), 0o644))

	schemas, err := DiscoverSchemas(DiscoveryOptions{
		Input:      tmp,
		Extensions: []string{".schema"},
	})
	require.NoError(t, err)
	require.Len(t, schemas, 1)

	assert.Equal(t, "a.schema", schemas[0].RelativePath)
}

func TestDiscoverSchemas_EmptyDirectory(t *testing.T) {
	tmp := t.TempDir()

	schemas, err := DiscoverSchemas(DiscoveryOptions{Input: tmp})
	require.NoError(t, err)
	assert.Empty(t, schemas)
}

func TestDiscoverSchemas_NonExistentPath(t *testing.T) {
	_, err := DiscoverSchemas(DiscoveryOptions{
		Input: "/no/such/path",
	})
	assert.Error(t, err)
}

func extractRelativePaths(s []DiscoveredSchema) []string {
	out := make([]string, len(s))
	for i, v := range s {
		out[i] = v.RelativePath
	}
	return out
}
