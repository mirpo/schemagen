package generation

import (
	"testing"

	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoGenerator_Generate_InvalidImports(t *testing.T) {
	gen := newGoGenerator(&typegraph.Graph{}, &Config{
		Language: LanguageGo,
		Go: &GoConfig{
			PackageName: "models",
		},
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

func TestGoGenerator_Generate_ValidImports(t *testing.T) {
	gen := newGoGenerator(&typegraph.Graph{}, &Config{
		Language: LanguageGo,
		Go: &GoConfig{
			PackageName: "models",
		},
	})

	out, err := gen.Generate(nil, []typegraph.ImportSpec{})
	require.NoError(t, err)
	assert.NotEmpty(t, out)
}

func TestGoGenerator_ConvertImports(t *testing.T) {
	gen := newGoGenerator(&typegraph.Graph{}, &Config{
		Language: LanguageGo,
		Go: &GoConfig{
			PackageName: "models",
			ModulePath:  "github.com/test/project",
		},
	}).(*goGenerator)

	result := gen.ConvertImports([]output.ImportSpec{
		{
			FromPath:   "events/event.go",
			ImportPath: "./header",
			TypeNames:  []string{"Header"},
		},
		{
			ImportPath: "github.com/pkg/errors",
		},
	}).([]typegraph.ImportSpec)

	require.Len(t, result, 2)
	assert.Equal(t, "github.com/test/project/events/header", result[0].ImportPath)
	assert.Equal(t, "github.com/pkg/errors", result[1].ImportPath)
}

func TestRelativeToAbsoluteImport(t *testing.T) {
	tests := []struct {
		name       string
		relPath    string
		fromFile   string
		modulePath string
		expected   string
	}{
		{
			"same dir",
			"./header",
			"events/event.go",
			"github.com/test/project",
			"github.com/test/project/events/header",
		},
		{
			"parent dir",
			"../common/types",
			"events/event.go",
			"github.com/test/project",
			"github.com/test/project/common/types",
		},
		{
			"no module path",
			"./header",
			"events/event.go",
			"",
			"events/header",
		},
		{
			"external unchanged",
			"github.com/pkg/errors",
			"events/event.go",
			"github.com/test/project",
			"github.com/pkg/errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(
				t,
				tt.expected,
				relativeToAbsoluteImport(tt.relPath, tt.fromFile, tt.modulePath),
			)
		})
	}
}
