package generation

import (
	"strings"
	"testing"

	"github.com/mirpo/schemagen/pkg/typegraph"
)

func TestPythonGenerator_Generate_InvalidImportsType(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: LanguagePython,
		Python:   &PythonConfig{},
	}
	gen := newPythonGenerator(graph, cfg)

	testCases := []struct {
		name        string
		imports     interface{}
		expectPanic bool
	}{
		{
			name:        "nil imports",
			imports:     nil,
			expectPanic: true,
		},
		{
			name:        "wrong slice type - []string",
			imports:     []string{"import1", "import2"},
			expectPanic: true,
		},
		{
			name:        "wrong concrete type - string",
			imports:     "invalid",
			expectPanic: true,
		},
		{
			name:        "wrong concrete type - int",
			imports:     42,
			expectPanic: true,
		},
		{
			name:        "wrong map type",
			imports:     map[string]string{"key": "value"},
			expectPanic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			types := []*typegraph.Type{}

			_, err := gen.Generate(types, tc.imports)

			if err == nil {
				t.Errorf("expected error for invalid imports type, got nil")
			}

			if err != nil {
				errMsg := err.Error()
				if !strings.Contains(errMsg, "invalid imports type") {
					t.Errorf("error message should contain 'invalid imports type', got: %s", errMsg)
				}
			}
		})
	}
}

func TestPythonGenerator_Generate_ValidImports(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: LanguagePython,
		Python:   &PythonConfig{},
	}
	gen := newPythonGenerator(graph, cfg)

	testCases := []struct {
		name    string
		imports []typegraph.ImportSpec
	}{
		{
			name:    "empty imports slice",
			imports: []typegraph.ImportSpec{},
		},
		{
			name: "single import",
			imports: []typegraph.ImportSpec{
				{
					ImportPath: ".types",
					TypeNames:  []string{"User"},
				},
			},
		},
		{
			name: "multiple imports",
			imports: []typegraph.ImportSpec{
				{
					ImportPath: ".types",
					TypeNames:  []string{"User", "Post"},
				},
				{
					ImportPath: ".common",
					TypeNames:  []string{"Base"},
				},
			},
		},
		{
			name: "import with empty type names",
			imports: []typegraph.ImportSpec{
				{
					ImportPath: ".types",
					TypeNames:  []string{},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			types := []*typegraph.Type{}

			output, err := gen.Generate(types, tc.imports)
			if err != nil {
				t.Errorf("expected no error for valid imports, got: %v", err)
			}

			if output == "" {
				t.Error("expected non-empty output")
			}
		})
	}
}

func TestPythonGenerator_Generate_TypedNilImports(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: LanguagePython,
		Python:   &PythonConfig{},
	}
	gen := newPythonGenerator(graph, cfg)

	var typedNilImports []typegraph.ImportSpec = nil
	types := []*typegraph.Type{}

	output, err := gen.Generate(types, typedNilImports)
	if err != nil {
		t.Errorf("typed nil should pass type assertion, got error: %v", err)
	}

	if output == "" {
		t.Error("expected non-empty output")
	}
}
