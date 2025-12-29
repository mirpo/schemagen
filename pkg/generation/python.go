package generation

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/lang/py"
	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

// pythonGenerator wraps the Python generator from pkg/lang/py
type pythonGenerator struct {
	generator *py.Generator
}

// newPythonGenerator creates a new Python generator with the given config
func newPythonGenerator(graph *typegraph.Graph, cfg *Config) Generator {
	return &pythonGenerator{
		generator: py.NewGeneratorWithConfig(graph, &py.Config{
			DisableHeaders:   cfg.DisableHeaders,
			DisableTimestamp: cfg.DisableTimestamp,
			SnakeCaseField:   cfg.Python.SnakeCaseField,
			AllowExtraFields: cfg.Python.AdditionalProperties,
		}),
	}
}

// Generate generates Python code for the given types and imports
func (g *pythonGenerator) Generate(types []*typegraph.Type, imports interface{}) (string, error) {
	pyImports, ok := imports.([]typegraph.ImportSpec)
	if !ok {
		return "", fmt.Errorf("invalid imports type: expected []typegraph.ImportSpec, got %T", imports)
	}
	return g.generator.GenerateFile(types, pyImports)
}

// ConvertImports converts generic imports to Python-specific format
// Python uses dots for relative imports, not slashes
//   - "./module" → ".module"
//   - "../module" → "..module"
//   - "./dir/module" → ".dir.module"
func (g *pythonGenerator) ConvertImports(imports []output.ImportSpec) interface{} {
	result := make([]typegraph.ImportSpec, len(imports))
	for i, imp := range imports {
		pyPath := imp.ImportPath

		if strings.HasPrefix(pyPath, "./") {
			pyPath = strings.Replace(pyPath, "./", ".", 1)
			pyPath = strings.ReplaceAll(pyPath, "/", ".")
		} else if strings.HasPrefix(pyPath, "../") {
			pyPath = strings.ReplaceAll(pyPath, "../", "..")
			pyPath = strings.ReplaceAll(pyPath, "/", ".")
		}

		result[i] = typegraph.ImportSpec{
			ImportPath: pyPath,
			TypeNames:  imp.TypeNames,
		}
	}
	return result
}
