package generation

import (
	"strings"
	"testing"

	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

func TestGoGenerator_Generate_InvalidImportsType(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: LanguageGo,
		Go: &GoConfig{
			PackageName: "models",
		},
	}
	gen := newGoGenerator(graph, cfg)

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

func TestGoGenerator_Generate_ValidImports(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: LanguageGo,
		Go: &GoConfig{
			PackageName: "models",
		},
	}
	gen := newGoGenerator(graph, cfg)

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
					ImportPath: "github.com/example/types",
					TypeNames:  []string{"User"},
				},
			},
		},
		{
			name: "multiple imports",
			imports: []typegraph.ImportSpec{
				{
					ImportPath: "github.com/example/types",
					TypeNames:  []string{"User", "Post"},
				},
				{
					ImportPath: "github.com/example/common",
					TypeNames:  []string{"Base"},
				},
			},
		},
		{
			name: "import with empty type names",
			imports: []typegraph.ImportSpec{
				{
					ImportPath: "github.com/example/types",
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

func TestGoGenerator_Generate_TypedNilImports(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: LanguageGo,
		Go: &GoConfig{
			PackageName: "models",
		},
	}
	gen := newGoGenerator(graph, cfg)

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

func TestGoGenerator_ConvertImports_AbsolutePaths(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: LanguageGo,
		Go: &GoConfig{
			PackageName: "models",
			ModulePath:  "github.com/test/project",
		},
	}

	gen := newGoGenerator(graph, cfg).(*goGenerator)

	// Simulate relative import specs from output layer
	outputImports := []output.ImportSpec{
		{
			FromPath:   "events/event.go",
			ToPath:     "events/header.go",
			ImportPath: "./header", // Relative path from output layer
			TypeNames:  []string{"EventHeader"},
		},
		{
			FromPath:   "events/event.go",
			ToPath:     "events/payloads/ping.go",
			ImportPath: "./payloads/ping",
			TypeNames:  []string{"PingPayload"},
		},
	}

	result := gen.ConvertImports(outputImports)
	imports := result.([]typegraph.ImportSpec)

	if len(imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(imports))
	}

	// Should convert to absolute module paths
	if imports[0].ImportPath != "github.com/test/project/events/header" {
		t.Errorf("expected 'github.com/test/project/events/header', got '%s'", imports[0].ImportPath)
	}
	if len(imports[0].TypeNames) != 1 || imports[0].TypeNames[0] != "EventHeader" {
		t.Errorf("expected TypeNames ['EventHeader'], got %v", imports[0].TypeNames)
	}

	if imports[1].ImportPath != "github.com/test/project/events/payloads/ping" {
		t.Errorf("expected 'github.com/test/project/events/payloads/ping', got '%s'", imports[1].ImportPath)
	}
	if len(imports[1].TypeNames) != 1 || imports[1].TypeNames[0] != "PingPayload" {
		t.Errorf("expected TypeNames ['PingPayload'], got %v", imports[1].TypeNames)
	}
}

func TestGoGenerator_ConvertImports_NoModulePath(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: LanguageGo,
		Go: &GoConfig{
			PackageName: "models",
			// ModulePath not set - fallback to relative
		},
	}

	gen := newGoGenerator(graph, cfg).(*goGenerator)

	outputImports := []output.ImportSpec{
		{
			FromPath:   "event.go",
			ToPath:     "header.go",
			ImportPath: "./header",
			TypeNames:  []string{"EventHeader"},
		},
	}

	result := gen.ConvertImports(outputImports)
	imports := result.([]typegraph.ImportSpec)

	if len(imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(imports))
	}

	// Without ModulePath, should keep relative (fallback)
	if imports[0].ImportPath != "./header" {
		t.Errorf("expected './header', got '%s'", imports[0].ImportPath)
	}
}

func TestGoGenerator_ConvertImports_ExternalPackage(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: LanguageGo,
		Go: &GoConfig{
			PackageName: "models",
			ModulePath:  "github.com/test/project",
		},
	}

	gen := newGoGenerator(graph, cfg).(*goGenerator)

	outputImports := []output.ImportSpec{
		{
			ImportPath: "github.com/go-playground/validator/v10", // Already absolute
			TypeNames:  []string{},
		},
	}

	result := gen.ConvertImports(outputImports)
	imports := result.([]typegraph.ImportSpec)

	if len(imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(imports))
	}

	// External packages should remain unchanged
	if imports[0].ImportPath != "github.com/go-playground/validator/v10" {
		t.Errorf("expected 'github.com/go-playground/validator/v10', got '%s'", imports[0].ImportPath)
	}
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
			name:       "same directory",
			relPath:    "./header",
			fromFile:   "events/event.go",
			modulePath: "github.com/test/project",
			expected:   "github.com/test/project/events/header",
		},
		{
			name:       "subdirectory",
			relPath:    "./payloads/ping",
			fromFile:   "events/event.go",
			modulePath: "github.com/test/project",
			expected:   "github.com/test/project/events/payloads/ping",
		},
		{
			name:       "parent directory",
			relPath:    "../common/types",
			fromFile:   "events/event.go",
			modulePath: "github.com/test/project",
			expected:   "github.com/test/project/common/types",
		},
		{
			name:       "external package unchanged",
			relPath:    "github.com/go-playground/validator/v10",
			fromFile:   "events/event.go",
			modulePath: "github.com/test/project",
			expected:   "github.com/go-playground/validator/v10",
		},
		{
			name:       "no module path keeps relative",
			relPath:    "./header",
			fromFile:   "events/event.go",
			modulePath: "",
			expected:   "events/header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := relativeToAbsoluteImport(tt.relPath, tt.fromFile, tt.modulePath)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
