package generation

import (
	"testing"

	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerators_InvalidImports(t *testing.T) {
	generators := map[string]Generator{
		"go":         newGoGenerator(&typegraph.Graph{}, &Config{Language: LanguageGo, Go: &GoConfig{PackageName: "models"}}),
		"python":     newPythonGenerator(&typegraph.Graph{}, &Config{Language: LanguagePython, Python: &PythonConfig{}}),
		"typescript": newTypeScriptGenerator(&typegraph.Graph{}, &Config{Language: LanguageTypeScript, TypeScript: &TypeScriptConfig{}}),
	}

	invalidImports := []struct {
		name    string
		imports interface{}
	}{
		{"nil", nil},
		{"string", "bad"},
		{"slice of string", []string{"a"}},
		{"map", map[string]string{}},
	}

	for genName, gen := range generators {
		for _, tt := range invalidImports {
			t.Run(genName+"/"+tt.name, func(t *testing.T) {
				_, err := gen.Generate(nil, tt.imports)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid imports type")
			})
		}
	}
}

func TestGenerators_ValidImports(t *testing.T) {
	tests := []struct {
		name string
		gen  Generator
	}{
		{"go", newGoGenerator(&typegraph.Graph{}, &Config{Language: LanguageGo, Go: &GoConfig{PackageName: "models"}})},
		{"python", newPythonGenerator(&typegraph.Graph{}, &Config{Language: LanguagePython, Python: &PythonConfig{}})},
		{"typescript", newTypeScriptGenerator(&typegraph.Graph{}, &Config{Language: LanguageTypeScript, TypeScript: &TypeScriptConfig{}})},
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
		{"python", newPythonGenerator(&typegraph.Graph{}, &Config{Language: LanguagePython, Python: &PythonConfig{}})},
		{"typescript", newTypeScriptGenerator(&typegraph.Graph{}, &Config{Language: LanguageTypeScript, TypeScript: &TypeScriptConfig{}})},
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

func TestGoGenerator_ConvertImports(t *testing.T) {
	gen := newGoGenerator(&typegraph.Graph{}, &Config{
		Language: LanguageGo,
		Go:       &GoConfig{PackageName: "models", ModulePath: "github.com/test/project"},
	}).(*goGenerator)

	result := gen.ConvertImports([]output.ImportSpec{
		{FromPath: "events/event.go", ImportPath: "./header", TypeNames: []string{"Header"}},
		{ImportPath: "github.com/pkg/errors"},
	}).([]typegraph.ImportSpec)

	require.Len(t, result, 2)
	assert.Equal(t, "github.com/test/project/events/header", result[0].ImportPath)
	assert.Equal(t, "github.com/pkg/errors", result[1].ImportPath)
}

func TestRelativeToAbsoluteImport(t *testing.T) {
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
			assert.Equal(t, tt.expected, relativeToAbsoluteImport(tt.relPath, tt.fromFile, tt.modulePath))
		})
	}
}

func TestPythonGenerator_ConvertImports(t *testing.T) {
	gen := newPythonGenerator(&typegraph.Graph{}, &Config{Language: LanguagePython, Python: &PythonConfig{}}).(*pythonGenerator)

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
			result := gen.ConvertImports([]output.ImportSpec{{ImportPath: tt.input}}).([]typegraph.ImportSpec)
			require.Len(t, result, 1)
			assert.Equal(t, tt.expected, result[0].ImportPath)
		})
	}
}

func TestTypeScriptGenerator_ConvertImports(t *testing.T) {
	gen := newTypeScriptGenerator(&typegraph.Graph{}, &Config{
		Language: LanguageTypeScript, TypeScript: &TypeScriptConfig{},
	}).(*typeScriptGenerator)

	result := gen.ConvertImports([]output.ImportSpec{{ImportPath: "./types", TypeNames: []string{"User"}}}).([]typegraph.ImportSpec)

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
			gen, err := createGenerator(&typegraph.Graph{}, &Config{
				Language: tt.language,
			})

			require.Error(t, err)
			assert.Nil(t, gen)
			assert.Contains(t, err.Error(), "unsupported language")
		})
	}
}
