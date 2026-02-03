package generation

import (
	"github.com/mirpo/schemagen/pkg/typegraph"
)

// LanguageGenerator is the interface that language-specific generators implement
type LanguageGenerator interface {
	GenerateFile(types []*typegraph.Type, imports []typegraph.ImportSpec) (string, error)
}

// combinedGenerator wraps a language generator with an import converter.
// This is a simplified adapter that implements the Generator interface.
type combinedGenerator struct {
	generator LanguageGenerator
	converter ImportConverter
}

// newCombinedGenerator creates a generator that combines a language generator with an import converter.
func newCombinedGenerator(gen LanguageGenerator, conv ImportConverter) *combinedGenerator {
	return &combinedGenerator{
		generator: gen,
		converter: conv,
	}
}

// Generate generates code for the given types and imports.
func (g *combinedGenerator) Generate(types []*typegraph.Type, imports []typegraph.ImportSpec) (string, error) {
	return g.generator.GenerateFile(types, imports)
}

// ConvertImports converts imports to language-specific format.
func (g *combinedGenerator) ConvertImports(imports []typegraph.ImportSpec) []typegraph.ImportSpec {
	return g.converter.Convert(imports)
}
