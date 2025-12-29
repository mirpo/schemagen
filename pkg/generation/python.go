package generation

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/lang/py"
	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

type pythonGenerator struct {
	generator *py.Generator
}

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

func (g *pythonGenerator) Generate(types []*typegraph.Type, imports interface{}) (string, error) {
	pyImports, ok := imports.([]typegraph.ImportSpec)
	if !ok {
		return "", fmt.Errorf(
			"invalid imports type: expected []typegraph.ImportSpec, got %T",
			imports,
		)
	}
	return g.generator.GenerateFile(types, pyImports)
}

// ConvertImports converts generic imports to Python-style module paths.
func (g *pythonGenerator) ConvertImports(imports []output.ImportSpec) interface{} {
	result := make([]typegraph.ImportSpec, len(imports))

	for i, imp := range imports {
		path := imp.ImportPath

		if strings.HasPrefix(path, ".") {
			dots := 0

			for strings.HasPrefix(path, "../") {
				dots++
				path = strings.TrimPrefix(path, "../")
			}

			path = strings.TrimPrefix(path, "./")

			path = strings.ReplaceAll(path, "/", ".")
			path = strings.Repeat(".", dots+1) + path
		}

		result[i] = typegraph.ImportSpec{
			ImportPath: path,
			TypeNames:  imp.TypeNames,
		}
	}

	return result
}
