package generation

import (
	"testing"

	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeScriptGenerator_Generate_InvalidImports(t *testing.T) {
	gen := newTypeScriptGenerator(&typegraph.Graph{}, &Config{
		Language:   LanguageTypeScript,
		TypeScript: &TypeScriptConfig{},
	})

	tests := []struct {
		name    string
		imports interface{}
	}{
		{"nil", nil},
		{"string", "bad"},
		{"slice of string", []string{"a"}},
		{"map", map[string]string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := gen.Generate(nil, tt.imports)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid imports type")
		})
	}
}

func TestTypeScriptGenerator_Generate_ValidImports(t *testing.T) {
	gen := newTypeScriptGenerator(&typegraph.Graph{}, &Config{
		Language:   LanguageTypeScript,
		TypeScript: &TypeScriptConfig{},
	})

	out, err := gen.Generate(nil, []typegraph.ImportSpec{})
	require.NoError(t, err)
	assert.NotEmpty(t, out)
}

func TestTypeScriptGenerator_Generate_TypedNilImports(t *testing.T) {
	gen := newTypeScriptGenerator(&typegraph.Graph{}, &Config{
		Language:   LanguageTypeScript,
		TypeScript: &TypeScriptConfig{},
	})

	var imports []typegraph.ImportSpec
	out, err := gen.Generate(nil, imports)

	require.NoError(t, err)
	assert.NotEmpty(t, out)
}

func TestTypeScriptGenerator_ConvertImports(t *testing.T) {
	gen := newTypeScriptGenerator(&typegraph.Graph{}, &Config{
		Language:   LanguageTypeScript,
		TypeScript: &TypeScriptConfig{},
	}).(*typeScriptGenerator)

	input := []output.ImportSpec{
		{
			ImportPath: "./types",
			TypeNames:  []string{"User"},
		},
	}

	result := gen.ConvertImports(input).([]typegraph.ImportSpec)

	require.Len(t, result, 1)
	assert.Equal(t, "./types", result[0].ImportPath)
	assert.Equal(t, []string{"User"}, result[0].TypeNames)
}
