package generation

import (
	"testing"

	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPythonGenerator_Generate_InvalidImports(t *testing.T) {
	gen := newPythonGenerator(&typegraph.Graph{}, &Config{
		Language: LanguagePython,
		Python:   &PythonConfig{},
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
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid imports type")
		})
	}
}

func TestPythonGenerator_Generate_ValidImports(t *testing.T) {
	gen := newPythonGenerator(&typegraph.Graph{}, &Config{
		Language: LanguagePython,
		Python:   &PythonConfig{},
	})

	out, err := gen.Generate(nil, []typegraph.ImportSpec{})
	require.NoError(t, err)
	assert.NotEmpty(t, out)
}

func TestPythonGenerator_Generate_TypedNilImports(t *testing.T) {
	gen := newPythonGenerator(&typegraph.Graph{}, &Config{
		Language: LanguagePython,
		Python:   &PythonConfig{},
	})

	var imports []typegraph.ImportSpec
	out, err := gen.Generate(nil, imports)

	require.NoError(t, err)
	assert.NotEmpty(t, out)
}

func TestPythonGenerator_ConvertImports(t *testing.T) {
	gen := newPythonGenerator(&typegraph.Graph{}, &Config{
		Language: LanguagePython,
		Python:   &PythonConfig{},
	}).(*pythonGenerator)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"same dir", "./module", ".module"},
		{"sub dir", "./dir/module", ".dir.module"},
		{"parent dir", "../common", "..common"},
		{"nested parent", "../../types", "...types"},
		{"absolute untouched", "pydantic", "pydantic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.ConvertImports([]output.ImportSpec{
				{ImportPath: tt.input},
			}).([]typegraph.ImportSpec)

			require.Len(t, result, 1)
			assert.Equal(t, tt.expected, result[0].ImportPath)
		})
	}
}
