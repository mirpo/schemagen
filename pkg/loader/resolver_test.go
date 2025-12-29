package loader

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractExternalRefs_Sibling(t *testing.T) {
	tmpDir := t.TempDir()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"header": map[string]interface{}{
				"$ref": "header.json",
			},
		},
	}

	schemaPath := filepath.Join(tmpDir, "event.json")
	writeJSON(t, schemaPath, schema)

	refs, err := ExtractExternalRefs(schemaPath)
	require.NoError(t, err, "ExtractExternalRefs() should not return error")
	require.Len(t, refs, 1, "should extract exactly 1 ref")

	expectedPath := filepath.Join(tmpDir, "header.json")
	assert.Equal(t, expectedPath, refs[0].FilePath, "FilePath should match")
}

func TestExtractExternalRefs_Subdirectory(t *testing.T) {
	tmpDir := t.TempDir()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"payload": map[string]interface{}{
				"$ref": "payloads/subscribe.json",
			},
		},
	}

	schemaPath := filepath.Join(tmpDir, "event.json")
	writeJSON(t, schemaPath, schema)

	refs, err := ExtractExternalRefs(schemaPath)
	require.NoError(t, err, "ExtractExternalRefs() should not return error")
	require.Len(t, refs, 1, "should extract exactly 1 ref")

	expectedPath := filepath.Join(tmpDir, "payloads", "subscribe.json")
	assert.Equal(t, expectedPath, refs[0].FilePath, "FilePath should match")
}

func TestExtractExternalRefs_ParentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested")
	err := os.MkdirAll(nestedDir, 0o755)
	require.NoError(t, err, "should create nested directory")

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"common": map[string]interface{}{
				"$ref": "../types.json",
			},
		},
	}

	schemaPath := filepath.Join(nestedDir, "user.json")
	writeJSON(t, schemaPath, schema)

	refs, err := ExtractExternalRefs(schemaPath)
	require.NoError(t, err, "ExtractExternalRefs() should not return error")
	require.Len(t, refs, 1, "should extract exactly 1 ref")

	expectedPath := filepath.Join(tmpDir, "types.json")
	assert.Equal(t, expectedPath, refs[0].FilePath, "FilePath should match")
}

func TestExtractExternalRefs_InternalRef_Skipped(t *testing.T) {
	tmpDir := t.TempDir()

	schema := map[string]interface{}{
		"$defs": map[string]interface{}{
			"Address": map[string]interface{}{
				"type": "object",
			},
		},
		"properties": map[string]interface{}{
			"address": map[string]interface{}{
				"$ref": "#/$defs/Address",
			},
		},
	}

	schemaPath := filepath.Join(tmpDir, "user.json")
	writeJSON(t, schemaPath, schema)

	refs, err := ExtractExternalRefs(schemaPath)
	require.NoError(t, err, "ExtractExternalRefs() should not return error")
	assert.Empty(t, refs, "internal refs should be skipped")
}

func TestExtractExternalRefs_WithFragment(t *testing.T) {
	tmpDir := t.TempDir()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"$ref": "common.json#/$defs/EventType",
			},
		},
	}

	schemaPath := filepath.Join(tmpDir, "event.json")
	writeJSON(t, schemaPath, schema)

	refs, err := ExtractExternalRefs(schemaPath)
	require.NoError(t, err, "ExtractExternalRefs() should not return error")
	require.Len(t, refs, 1, "should extract exactly 1 ref")

	expectedPath := filepath.Join(tmpDir, "common.json")
	assert.Equal(t, expectedPath, refs[0].FilePath, "FilePath should match")
	assert.Equal(t, "/$defs/EventType", refs[0].Fragment, "Fragment should match")
}

func TestExtractExternalRefs_MultipleRefs(t *testing.T) {
	tmpDir := t.TempDir()

	schema := map[string]interface{}{
		"allOf": []interface{}{
			map[string]interface{}{"$ref": "base.json"},
			map[string]interface{}{"$ref": "timestamps.json"},
		},
		"properties": map[string]interface{}{
			"owner": map[string]interface{}{
				"$ref": "person.json",
			},
			"address": map[string]interface{}{
				"$ref": "common/address.json",
			},
		},
	}

	schemaPath := filepath.Join(tmpDir, "user.json")
	writeJSON(t, schemaPath, schema)

	refs, err := ExtractExternalRefs(schemaPath)
	require.NoError(t, err, "ExtractExternalRefs() should not return error")
	require.Len(t, refs, 4, "should extract exactly 4 refs")

	refPaths := make(map[string]bool)
	for _, ref := range refs {
		refPaths[filepath.Base(ref.FilePath)] = true
	}

	expectedFiles := []string{"base.json", "timestamps.json", "person.json", "address.json"}
	for _, expected := range expectedFiles {
		assert.True(t, refPaths[expected], "should find ref to %s", expected)
	}
}

func TestExtractExternalRefs_NestedInAllOf(t *testing.T) {
	tmpDir := t.TempDir()

	schema := map[string]interface{}{
		"allOf": []interface{}{
			map[string]interface{}{
				"properties": map[string]interface{}{
					"nested": map[string]interface{}{
						"$ref": "nested.json",
					},
				},
			},
		},
	}

	schemaPath := filepath.Join(tmpDir, "test.json")
	writeJSON(t, schemaPath, schema)

	refs, err := ExtractExternalRefs(schemaPath)
	require.NoError(t, err, "ExtractExternalRefs() should not return error")
	require.Len(t, refs, 1, "should extract exactly 1 ref")

	expectedPath := filepath.Join(tmpDir, "nested.json")
	assert.Equal(t, expectedPath, refs[0].FilePath, "FilePath should match")
}

func TestExtractExternalRefs_Deduplication(t *testing.T) {
	tmpDir := t.TempDir()

	schema := map[string]interface{}{
		"properties": map[string]interface{}{
			"field1": map[string]interface{}{
				"$ref": "common.json",
			},
			"field2": map[string]interface{}{
				"$ref": "common.json",
			},
			"field3": map[string]interface{}{
				"$ref": "common.json#/$defs/Type",
			},
		},
	}

	schemaPath := filepath.Join(tmpDir, "test.json")
	writeJSON(t, schemaPath, schema)

	refs, err := ExtractExternalRefs(schemaPath)
	require.NoError(t, err, "ExtractExternalRefs() should not return error")
	require.Len(t, refs, 2, "should deduplicate refs but keep different fragments")
}

func TestLoadSchemasRecursive_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()

	headerSchema := map[string]interface{}{"type": "object"}
	writeJSON(t, filepath.Join(tmpDir, "header.json"), headerSchema)

	eventSchema := map[string]interface{}{
		"properties": map[string]interface{}{
			"header": map[string]interface{}{"$ref": "header.json"},
		},
	}
	eventPath := filepath.Join(tmpDir, "event.json")
	writeJSON(t, eventPath, eventSchema)

	schemas, err := LoadSchemasRecursive(eventPath)
	require.NoError(t, err, "LoadSchemasRecursive() should not return error")
	require.Len(t, schemas, 2, "should load exactly 2 schemas (event + header)")

	foundEvent := false
	foundHeader := false
	for _, schema := range schemas {
		if filepath.Base(schema.AbsolutePath) == "event.json" {
			foundEvent = true
		}
		if filepath.Base(schema.AbsolutePath) == "header.json" {
			foundHeader = true
		}
	}

	assert.True(t, foundEvent, "should find event.json")
	assert.True(t, foundHeader, "should find header.json")
}

func TestLoadSchemasRecursive_ChainedDeps(t *testing.T) {
	tmpDir := t.TempDir()

	cSchema := map[string]interface{}{"type": "string"}
	writeJSON(t, filepath.Join(tmpDir, "c.json"), cSchema)

	bSchema := map[string]interface{}{
		"properties": map[string]interface{}{
			"c": map[string]interface{}{"$ref": "c.json"},
		},
	}
	writeJSON(t, filepath.Join(tmpDir, "b.json"), bSchema)

	aSchema := map[string]interface{}{
		"properties": map[string]interface{}{
			"b": map[string]interface{}{"$ref": "b.json"},
		},
	}
	aPath := filepath.Join(tmpDir, "a.json")
	writeJSON(t, aPath, aSchema)

	schemas, err := LoadSchemasRecursive(aPath)
	require.NoError(t, err, "LoadSchemasRecursive() should not return error")
	require.Len(t, schemas, 3, "should load exactly 3 schemas (a → b → c)")

	foundFiles := make(map[string]bool)
	for _, schema := range schemas {
		foundFiles[filepath.Base(schema.AbsolutePath)] = true
	}

	for _, expected := range []string{"a.json", "b.json", "c.json"} {
		assert.True(t, foundFiles[expected], "should find %s", expected)
	}
}

func TestLoadSchemasRecursive_CircularRef(t *testing.T) {
	tmpDir := t.TempDir()

	aSchema := map[string]interface{}{
		"properties": map[string]interface{}{
			"b": map[string]interface{}{"$ref": "b.json"},
		},
	}
	aPath := filepath.Join(tmpDir, "a.json")
	writeJSON(t, aPath, aSchema)

	bSchema := map[string]interface{}{
		"properties": map[string]interface{}{
			"a": map[string]interface{}{"$ref": "a.json"},
		},
	}
	writeJSON(t, filepath.Join(tmpDir, "b.json"), bSchema)

	schemas, err := LoadSchemasRecursive(aPath)
	require.NoError(t, err, "LoadSchemasRecursive() should not return error")
	require.Len(t, schemas, 2, "circular ref should not cause infinite loop")
}

func TestLoadSchemasRecursive_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	writeJSON(t, filepath.Join(tmpDir, "a.json"), map[string]interface{}{"type": "object"})
	writeJSON(t, filepath.Join(tmpDir, "b.json"), map[string]interface{}{"type": "object"})

	schemas, err := LoadSchemasRecursive(tmpDir)
	require.NoError(t, err, "LoadSchemasRecursive() should not return error")
	require.Len(t, schemas, 2, "should load exactly 2 schemas")
}

func writeJSON(t *testing.T, path string, data interface{}) {
	t.Helper()

	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0o755)
	require.NoError(t, err, "should create directory")

	jsonData, err := json.Marshal(data)
	require.NoError(t, err, "should marshal JSON")

	err = os.WriteFile(path, jsonData, 0o644)
	require.NoError(t, err, "should write JSON file")
}
