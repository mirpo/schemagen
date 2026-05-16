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

func TestPassthroughConverter_Convert(t *testing.T) {
	converter := &PassthroughConverter{}

	result := converter.Convert([]typegraph.ImportSpec{{ImportPath: "./types", TypeNames: []string{"User"}}})

	require.Len(t, result, 1)
	assert.Equal(t, "./types", result[0].ImportPath)
	assert.Equal(t, []string{"User"}, result[0].TypeNames)
}

func TestCreateGenerator_UnsupportedLanguage(t *testing.T) {
	gen, err := createGenerator(&Config{Language: ""})
	require.Error(t, err)
	assert.Nil(t, gen)
	assert.Contains(t, err.Error(), "unsupported language")
}
