package loader

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Helpers
*/

func writeJSON(t *testing.T, path string, data any) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	b, err := json.Marshal(data)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, b, 0o644))
}

/*
ExtractExternalRefs
*/

func TestExtractExternalRefs_Sibling(t *testing.T) {
	tmp := t.TempDir()

	writeJSON(t, filepath.Join(tmp, "event.json"), map[string]any{
		"properties": map[string]any{
			"header": map[string]any{
				"$ref": "header.json",
			},
		},
	})

	refs, err := ExtractExternalRefs(filepath.Join(tmp, "event.json"))
	require.NoError(t, err)
	require.Len(t, refs, 1)

	assert.Equal(t,
		filepath.Join(tmp, "header.json"),
		refs[0].FilePath,
	)
	assert.Empty(t, refs[0].Fragment)
}

func TestExtractExternalRefs_WithFragment(t *testing.T) {
	tmp := t.TempDir()

	writeJSON(t, filepath.Join(tmp, "event.json"), map[string]any{
		"properties": map[string]any{
			"type": map[string]any{
				"$ref": "common.json#/$defs/Type",
			},
		},
	})

	refs, err := ExtractExternalRefs(filepath.Join(tmp, "event.json"))
	require.NoError(t, err)
	require.Len(t, refs, 1)

	assert.Equal(t,
		filepath.Join(tmp, "common.json"),
		refs[0].FilePath,
	)
	assert.Equal(t, "/$defs/Type", refs[0].Fragment)
}

func TestExtractExternalRefs_InternalRefsIgnored(t *testing.T) {
	tmp := t.TempDir()

	writeJSON(t, filepath.Join(tmp, "schema.json"), map[string]any{
		"$defs": map[string]any{
			"A": map[string]any{"type": "string"},
		},
		"properties": map[string]any{
			"a": map[string]any{
				"$ref": "#/$defs/A",
			},
		},
	})

	refs, err := ExtractExternalRefs(filepath.Join(tmp, "schema.json"))
	require.NoError(t, err)
	assert.Empty(t, refs)
}

func TestExtractExternalRefs_Deduplication(t *testing.T) {
	tmp := t.TempDir()

	writeJSON(t, filepath.Join(tmp, "schema.json"), map[string]any{
		"properties": map[string]any{
			"a": map[string]any{"$ref": "common.json"},
			"b": map[string]any{"$ref": "common.json"},
			"c": map[string]any{"$ref": "common.json#/$defs/X"},
		},
	})

	refs, err := ExtractExternalRefs(filepath.Join(tmp, "schema.json"))
	require.NoError(t, err)
	require.Len(t, refs, 2)
}

/*
LoadSchemasRecursive
*/

func TestLoadSchemasRecursive_SingleFileWithDeps(t *testing.T) {
	tmp := t.TempDir()

	writeJSON(t, filepath.Join(tmp, "header.json"), map[string]any{"type": "object"})
	writeJSON(t, filepath.Join(tmp, "event.json"), map[string]any{
		"properties": map[string]any{
			"h": map[string]any{"$ref": "header.json"},
		},
	})

	schemas, err := LoadSchemasRecursive(filepath.Join(tmp, "event.json"))
	require.NoError(t, err)
	require.Len(t, schemas, 2)

	found := map[string]bool{}
	for _, s := range schemas {
		found[filepath.Base(s.AbsolutePath)] = true
	}

	assert.True(t, found["event.json"])
	assert.True(t, found["header.json"])
}

func TestLoadSchemasRecursive_ChainedDeps(t *testing.T) {
	tmp := t.TempDir()

	writeJSON(t, filepath.Join(tmp, "c.json"), map[string]any{"type": "string"})
	writeJSON(t, filepath.Join(tmp, "b.json"), map[string]any{
		"$ref": "c.json",
	})
	writeJSON(t, filepath.Join(tmp, "a.json"), map[string]any{
		"$ref": "b.json",
	})

	schemas, err := LoadSchemasRecursive(filepath.Join(tmp, "a.json"))
	require.NoError(t, err)
	require.Len(t, schemas, 3)
}

func TestLoadSchemasRecursive_CircularRefs(t *testing.T) {
	tmp := t.TempDir()

	writeJSON(t, filepath.Join(tmp, "a.json"), map[string]any{
		"$ref": "b.json",
	})
	writeJSON(t, filepath.Join(tmp, "b.json"), map[string]any{
		"$ref": "a.json",
	})

	schemas, err := LoadSchemasRecursive(filepath.Join(tmp, "a.json"))
	require.NoError(t, err)
	require.Len(t, schemas, 2)
}

func TestLoadSchemasRecursive_DirectoryDelegatesToDiscover(t *testing.T) {
	tmp := t.TempDir()

	writeJSON(t, filepath.Join(tmp, "a.json"), map[string]any{"type": "object"})
	writeJSON(t, filepath.Join(tmp, "b.json"), map[string]any{"type": "object"})

	schemas, err := LoadSchemasRecursive(tmp)
	require.NoError(t, err)
	require.Len(t, schemas, 2)
}
