package generation

import (
	"fmt"

	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

// LanguageGenerator is the interface that language-specific generators implement
type LanguageGenerator interface {
	GenerateFile(types []*typegraph.Type, imports []typegraph.ImportSpec) (string, error)
}

// generatorWrapper adapts a LanguageGenerator to the Generator interface
type generatorWrapper struct {
	generator LanguageGenerator
	converter ImportConverter
}

func newGeneratorWrapper(gen LanguageGenerator, conv ImportConverter) *generatorWrapper {
	return &generatorWrapper{
		generator: gen,
		converter: conv,
	}
}

func (w *generatorWrapper) Generate(types []*typegraph.Type, imports interface{}) (string, error) {
	if imports == nil {
		return w.generator.GenerateFile(types, nil)
	}

	// Handle both import types
	switch typedImports := imports.(type) {
	case []typegraph.ImportSpec:
		return w.generator.GenerateFile(types, typedImports)
	case []output.ImportSpec:
		return w.generator.GenerateFile(types, w.converter.Convert(typedImports))
	default:
		return "", fmt.Errorf("invalid imports type: expected []typegraph.ImportSpec or []output.ImportSpec, got %T", imports)
	}
}

func (w *generatorWrapper) ConvertImports(imports []output.ImportSpec) interface{} {
	return w.converter.Convert(imports)
}
