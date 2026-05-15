package generation

import (
	"testing"

	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerators_ValidImports(t *testing.T) {
	tests := []struct {
		name string
		gen  Generator
	}{
		{"go", newGoGenerator(&Config{Language: LanguageGo, Go: &GoConfig{PackageName: "models"}})},
		{"python", newPythonGenerator(&Config{Language: LanguagePython, Python: &PythonConfig{}})},
		{"typescript", newTypeScriptGenerator(&Config{Language: LanguageTypeScript, TypeScript: &TypeScriptConfig{}})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := tt.gen.Generate(nil, []typegraph.ImportSpec{})
			require.NoError(t, err)
			assert.NotEmpty(t, out)
		})
	}
}

func TestGenerators_TypedNilImports(t *testing.T) {
	tests := []struct {
		name string
		gen  Generator
	}{
		{"python", newPythonGenerator(&Config{Language: LanguagePython, Python: &PythonConfig{}})},
		{"typescript", newTypeScriptGenerator(&Config{Language: LanguageTypeScript, TypeScript: &TypeScriptConfig{}})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var imports []typegraph.ImportSpec
			out, err := tt.gen.Generate(nil, imports)
			require.NoError(t, err)
			assert.NotEmpty(t, out)
		})
	}
}

func TestGenerators_NilImports(t *testing.T) {
	tests := []struct {
		name string
		gen  Generator
	}{
		{"go", newGoGenerator(&Config{Language: LanguageGo, Go: &GoConfig{PackageName: "models"}})},
		{"python", newPythonGenerator(&Config{Language: LanguagePython, Python: &PythonConfig{}})},
		{"typescript", newTypeScriptGenerator(&Config{Language: LanguageTypeScript, TypeScript: &TypeScriptConfig{}})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := tt.gen.Generate(nil, nil)
			require.NoError(t, err)
			assert.NotEmpty(t, out)
		})
	}
}

func TestGoImportConverter_Convert(t *testing.T) {
	converter := &GoImportConverter{ModulePath: "github.com/test/project"}

	result := converter.Convert([]typegraph.ImportSpec{
		{FromPath: "events/event.go", ImportPath: "./header", TypeNames: []string{"Header"}},
		{ImportPath: "github.com/pkg/errors"},
	})

	require.Len(t, result, 2)
	assert.Equal(t, "github.com/test/project/events/header", result[0].ImportPath)
	assert.Equal(t, "github.com/pkg/errors", result[1].ImportPath)
}

func TestGoImportConverter_ResolveRelative(t *testing.T) {
	tests := []struct {
		name, relPath, fromFile, modulePath, expected string
	}{
		{"same dir", "./header", "events/event.go", "github.com/test/project", "github.com/test/project/events/header"},
		{"parent dir", "../common/types", "events/event.go", "github.com/test/project", "github.com/test/project/common/types"},
		{"no module path", "./header", "events/event.go", "", "events/header"},
		{"external unchanged", "github.com/pkg/errors", "events/event.go", "github.com/test/project", "github.com/pkg/errors"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := &GoImportConverter{ModulePath: tt.modulePath}
			assert.Equal(t, tt.expected, converter.resolveRelative(tt.relPath, tt.fromFile))
		})
	}
}

func TestPythonImportConverter_Convert(t *testing.T) {
	converter := &PythonImportConverter{}

	tests := []struct {
		name, input, expected string
	}{
		{"same dir", "./module", ".module"},
		{"sub dir", "./dir/module", ".dir.module"},
		{"parent dir", "../common", "..common"},
		{"nested parent", "../../types", "...types"},
		{"absolute untouched", "pydantic", "pydantic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert([]typegraph.ImportSpec{{ImportPath: tt.input}})
			require.Len(t, result, 1)
			assert.Equal(t, tt.expected, result[0].ImportPath)
		})
	}
}

func TestPassthroughConverter_Convert(t *testing.T) {
	converter := &PassthroughConverter{}

	result := converter.Convert([]typegraph.ImportSpec{{ImportPath: "./types", TypeNames: []string{"User"}}})

	require.Len(t, result, 1)
	assert.Equal(t, "./types", result[0].ImportPath)
	assert.Equal(t, []string{"User"}, result[0].TypeNames)
}

func TestCreateGenerator_UnsupportedLanguages(t *testing.T) {
	tests := []struct {
		name     string
		language Language
	}{
		{"empty", ""},
		{"rust", "rust"},
		{"java", "java"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := createGenerator(&Config{
				Language: tt.language,
			})

			require.Error(t, err)
			assert.Nil(t, gen)
			assert.Contains(t, err.Error(), "unsupported language")
		})
	}
}
