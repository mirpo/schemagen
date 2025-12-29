package output

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateNestedBarrels_Minimal(t *testing.T) {
	files := []OutputFile{
		{RelativePath: "root.ts"},
		{RelativePath: filepath.Join("a", "file1.ts")},
		{RelativePath: filepath.Join("a", "b", "file2.ts")},
	}

	barrels := GenerateNestedBarrels(files, "ts")

	require.Len(t, barrels, 3)

	found := map[string][]string{}
	for _, b := range barrels {
		found[b.Path] = b.Exports
	}

	assert.ElementsMatch(t, []string{"root"}, found["index.ts"])
	assert.ElementsMatch(t, []string{"file1"}, found[filepath.Join("a", "index.ts")])
	assert.ElementsMatch(t, []string{"file2"}, found[filepath.Join("a", "b", "index.ts")])
}

func TestGenerateNestedBarrels_SkipExistingBarrels(t *testing.T) {
	files := []OutputFile{
		{RelativePath: "index.ts"},
		{RelativePath: "a/index.ts"},
		{RelativePath: "a/file.ts"},
	}

	barrels := GenerateNestedBarrels(files, "ts")

	require.Len(t, barrels, 1)
	assert.ElementsMatch(
		t,
		[]string{"file"},
		barrels[0].Exports,
	)
	assert.Equal(t, filepath.Join("a", "index.ts"), barrels[0].Path)
}

func TestGenerateBarrelContent_TypeScript(t *testing.T) {
	barrel := BarrelFile{
		Path:    "index.ts",
		Exports: []string{"a", "b"},
	}

	out := GenerateBarrelContent(barrel, "ts")

	assert.Contains(t, out, "export * from './a';")
	assert.Contains(t, out, "export * from './b';")
}
