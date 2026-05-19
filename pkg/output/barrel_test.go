package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateNestedBarrels_Minimal(t *testing.T) {
	files := []OutputFile{
		{RelativePath: "root.ts"},
		{RelativePath: "a/file1.ts"},
		{RelativePath: "a/b/file2.ts"},
	}

	barrels := GenerateNestedBarrels(files, LanguageTypeScript)

	require.Len(t, barrels, 3)

	found := map[string][]string{}
	for _, b := range barrels {
		found[b.Path] = b.Exports
	}

	assert.ElementsMatch(t, []string{"root"}, found["index.ts"])
	assert.ElementsMatch(t, []string{"file1"}, found["a/index.ts"])
	assert.ElementsMatch(t, []string{"file2"}, found["a/b/index.ts"])
}

func TestGenerateNestedBarrels_SkipExistingBarrels(t *testing.T) {
	files := []OutputFile{
		{RelativePath: "index.ts"},
		{RelativePath: "a/index.ts"},
		{RelativePath: "a/file.ts"},
	}

	barrels := GenerateNestedBarrels(files, LanguageTypeScript)

	require.Len(t, barrels, 1)
	assert.ElementsMatch(
		t,
		[]string{"file"},
		barrels[0].Exports,
	)
	assert.Equal(t, "a/index.ts", barrels[0].Path)
}

func TestGenerateBarrelContent_TypeScript(t *testing.T) {
	barrel := BarrelFile{
		Path:    "index.ts",
		Exports: []string{"a", "b"},
	}

	out := GenerateBarrelContent(barrel, LanguageTypeScript)

	assert.Contains(t, out, "export * from './a';")
	assert.Contains(t, out, "export * from './b';")
}

func TestGenerateBarrelContent_Python(t *testing.T) {
	barrel := BarrelFile{
		Path:    "__init__.py",
		Exports: []string{"user_model", "product"},
	}

	out := GenerateBarrelContent(barrel, LanguagePython)

	assert.Contains(t, out, "from .user_model import *")
	assert.Contains(t, out, "from .product import *")
	assert.Contains(t, out, "#")
}

func TestGenerateBarrelContent_UnsupportedLanguage(t *testing.T) {
	barrel := BarrelFile{
		Path:    "barrel.go",
		Exports: []string{"a"},
	}

	out := GenerateBarrelContent(barrel, LanguageGo)
	assert.Empty(t, out)
}

func TestGenerateNestedBarrels_Python(t *testing.T) {
	files := []OutputFile{
		{RelativePath: "user.py"},
		{RelativePath: "models/product.py"},
	}

	barrels := GenerateNestedBarrels(files, LanguagePython)
	require.NotEmpty(t, barrels)

	found := map[string][]string{}
	for _, b := range barrels {
		found[b.Path] = b.Exports
	}

	assert.ElementsMatch(t, []string{"user"}, found["__init__.py"])
	assert.ElementsMatch(t, []string{"product"}, found["models/__init__.py"])
}

func TestGenerateNestedBarrels_UnsupportedLanguage(t *testing.T) {
	files := []OutputFile{
		{RelativePath: "models.go"},
	}

	barrels := GenerateNestedBarrels(files, LanguageGo)
	assert.Nil(t, barrels)
}
