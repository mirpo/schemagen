package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kaptinlin/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_New(t *testing.T) {
	loader := NewLoader()
	require.NotNil(t, loader)
	assert.NotNil(t, loader.Compiler())
}

func TestLoader_Load_SingleJSON(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "user.json", `{
		"type": "object",
		"title": "User"
	}`)

	loader := NewLoader()
	schemas, err := loader.Load(filepath.Join(tmp, "user.json"))

	require.NoError(t, err)
	require.Len(t, schemas, 1)

	s := schemas[0]
	assert.Equal(t, "User", s.Name)
	assert.Equal(t, "user.json", s.RelativePath)
	assert.NotNil(t, s.Compiled)
}

func TestLoader_Load_DirectoryRecursive(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "root.json", `{"title":"Root"}`)
	writeFile(t, filepath.Join(tmp, "events"), "event.json", `{"title":"Event"}`)
	writeFile(t, filepath.Join(tmp, "events/payloads"), "sub.json", `{"title":"Sub"}`)

	loader := NewLoader()
	schemas, err := loader.Load(tmp)

	require.NoError(t, err)
	require.Len(t, schemas, 3)

	paths := collectRelativePaths(schemas)
	assert.ElementsMatch(t, []string{
		"root.json",
		"events/event.json",
		"events/payloads/sub.json",
	}, paths)
}

func TestLoader_Skips_NonSchemaFiles(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "valid.json", `{"title":"Valid"}`)
	writeFile(t, tmp, "readme.txt", "ignore me")
	writeFile(t, tmp, "config.toml", "ignore me")

	loader := NewLoader()
	schemas, err := loader.Load(tmp)

	require.NoError(t, err)
	require.Len(t, schemas, 1)
	assert.Equal(t, "Valid", schemas[0].Name)
}

func TestLoader_Load_JSON_YAML_YML(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "a.json", `{"title":"A"}`)
	writeFile(t, tmp, "b.yaml", "title: B")
	writeFile(t, tmp, "c.yml", "title: C")

	loader := NewLoader()
	schemas, err := loader.Load(tmp)

	require.NoError(t, err)
	require.Len(t, schemas, 3)

	names := collectNames(schemas)
	assert.ElementsMatch(t, []string{"A", "B", "C"}, names)
}

func TestLoader_Refs_Work(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "address.json", `{"title":"Address"}`)
	writeFile(t, tmp, "user.json", `{
		"title":"User",
		"properties": {
			"addr": {"$ref":"address.json"}
		}
	}`)

	loader := NewLoader()
	schemas, err := loader.Load(tmp)

	require.NoError(t, err)
	require.Len(t, schemas, 2)

	compiler := loader.Compiler()
	_, err = compiler.GetSchema("address.json")
	assert.NoError(t, err)
}

func TestLoader_InvalidInput_Errors(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "broken.json", `{ invalid json `)

	loader := NewLoader()
	_, err := loader.Load(filepath.Join(tmp, "broken.json"))

	require.Error(t, err)
}

func TestDeriveName(t *testing.T) {
	tests := []struct {
		path  string
		title *string
		want  string
	}{
		{"/x/user.json", str("User"), "User"},
		{"/x/user.json", str("user profile"), "UserProfile"},
		{"/x/user-profile.json", nil, "UserProfile"},
	}

	for _, tt := range tests {
		s := &jsonschema.Schema{Title: tt.title}
		assert.Equal(t, tt.want, deriveName(tt.path, s))
	}
}

/* ---------------- helpers ---------------- */

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func collectNames(schemas []*Schema) []string {
	var out []string
	for _, s := range schemas {
		out = append(out, s.Name)
	}
	return out
}

func collectRelativePaths(schemas []*Schema) []string {
	var out []string
	for _, s := range schemas {
		out = append(out, s.RelativePath)
	}
	return out
}

func str(s string) *string { return &s }
