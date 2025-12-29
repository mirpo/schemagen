package generation

import (
	"fmt"

	"github.com/mirpo/schemagen/pkg/lang/ts"
	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

// typeScriptGenerator wraps the TypeScript generator from pkg/lang/ts
type typeScriptGenerator struct {
	generator *ts.Generator
}

// newTypeScriptGenerator creates a new TypeScript generator with the given config
func newTypeScriptGenerator(graph *typegraph.Graph, cfg *Config) Generator {
	return &typeScriptGenerator{
		generator: ts.NewGeneratorWithConfig(graph, &ts.Config{
			DisableHeaders:       cfg.DisableHeaders,
			DisableTimestamp:     cfg.DisableTimestamp,
			UnknownAny:           cfg.TypeScript.UnknownAny,
			AdditionalProperties: cfg.TypeScript.AdditionalProperties,
		}),
	}
}

// Generate generates TypeScript code for the given types and imports
func (g *typeScriptGenerator) Generate(types []*typegraph.Type, imports interface{}) (string, error) {
	tsImports, ok := imports.([]typegraph.ImportSpec)
	if !ok {
		return "", fmt.Errorf("invalid imports type: expected []typegraph.ImportSpec, got %T", imports)
	}
	return g.generator.GenerateFile(types, tsImports)
}

// ConvertImports converts generic imports to TypeScript-specific format
func (g *typeScriptGenerator) ConvertImports(imports []output.ImportSpec) interface{} {
	result := make([]typegraph.ImportSpec, len(imports))
	for i, imp := range imports {
		result[i] = typegraph.ImportSpec{
			ImportPath: imp.ImportPath,
			TypeNames:  imp.TypeNames,
		}
	}
	return result
}
