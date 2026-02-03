package generation

import (
	"fmt"

	"github.com/mirpo/schemagen/pkg/typegraph"
)

// Generator is the interface that all language-specific generators must implement
type Generator interface {
	// Generate generates code for the given types and imports
	Generate(types []*typegraph.Type, imports interface{}) (string, error)

	// ConvertImports converts generic imports to language-specific format
	ConvertImports(imports []typegraph.ImportSpec) interface{}
}

// createGenerator creates a language-specific generator based on the config
func createGenerator(graph *typegraph.Graph, cfg *Config) (Generator, error) {
	switch cfg.Language {
	case LanguageTypeScript:
		return newTypeScriptGenerator(graph, cfg), nil

	case LanguagePython:
		return newPythonGenerator(graph, cfg), nil

	case LanguageGo:
		return newGoGenerator(graph, cfg), nil

	default:
		return nil, fmt.Errorf("unsupported language: %s", cfg.Language)
	}
}
