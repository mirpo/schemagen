package generation

import (
	"fmt"

	"github.com/mirpo/schemagen/pkg/typegraph"
)

// Generator is the interface that all language-specific generators must implement
type Generator interface {
	// Generate generates code for the given types and imports
	Generate(types []*typegraph.Type, imports []typegraph.ImportSpec) (string, error)

	// ConvertImports converts imports to language-specific format
	ConvertImports(imports []typegraph.ImportSpec) []typegraph.ImportSpec
}

// createGenerator creates a language-specific generator based on the config
func createGenerator(cfg *Config) (Generator, error) {
	switch cfg.Language {
	case LanguageTypeScript:
		return newTypeScriptGenerator(cfg), nil

	case LanguagePython:
		return newPythonGenerator(cfg), nil

	case LanguageGo:
		return newGoGenerator(cfg), nil

	default:
		return nil, fmt.Errorf("unsupported language: %s", cfg.Language)
	}
}
