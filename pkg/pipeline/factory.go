package pipeline

import (
	"fmt"

	"github.com/mirpo/schemagen/pkg/graph"
	"github.com/mirpo/schemagen/pkg/render/golang"
	"github.com/mirpo/schemagen/pkg/render/py"
	"github.com/mirpo/schemagen/pkg/render/ts"
)

type Generator interface {
	Generate(types []*graph.Type, imports []graph.ImportSpec) (string, error)
	ConvertImports(imports []graph.ImportSpec) []graph.ImportSpec
}

type LanguageGenerator interface {
	GenerateFile(types []*graph.Type, imports []graph.ImportSpec) (string, error)
}

type ImportConverter interface {
	Convert(imports []graph.ImportSpec) []graph.ImportSpec
}

type PassthroughConverter struct{}

func (c *PassthroughConverter) Convert(imports []graph.ImportSpec) []graph.ImportSpec {
	return imports
}

type combinedGenerator struct {
	generator LanguageGenerator
	converter ImportConverter
}

func newCombinedGenerator(gen LanguageGenerator, conv ImportConverter) *combinedGenerator {
	return &combinedGenerator{generator: gen, converter: conv}
}

func (g *combinedGenerator) Generate(types []*graph.Type, imports []graph.ImportSpec) (string, error) {
	return g.generator.GenerateFile(types, imports)
}

func (g *combinedGenerator) ConvertImports(imports []graph.ImportSpec) []graph.ImportSpec {
	return g.converter.Convert(imports)
}

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

func newTypeScriptGenerator(cfg *Config) Generator {
	tsCfg := &ts.Config{
		DisableHeaders:       cfg.DisableHeaders,
		DisableTimestamp:     cfg.DisableTimestamp,
		UnknownAny:           cfg.TypeScript.UnknownAny,
		AdditionalProperties: cfg.TypeScript.AdditionalProperties,
		ZodCoerceDates:       cfg.TypeScript.ZodCoerceDates,
		ZodStrict:            cfg.TypeScript.ZodStrict,
	}

	if cfg.TypeScript.ZodOnly {
		tsCfg.ZodMode = ts.ZodModeOnly
	} else if cfg.TypeScript.Zod {
		tsCfg.ZodMode = ts.ZodModeWithInterface
	}

	return newCombinedGenerator(
		ts.NewGeneratorWithConfig(tsCfg),
		&PassthroughConverter{},
	)
}

func newPythonGenerator(cfg *Config) Generator {
	return newCombinedGenerator(
		py.NewGeneratorWithConfig(&py.Config{
			DisableHeaders:   cfg.DisableHeaders,
			DisableTimestamp: cfg.DisableTimestamp,
			SnakeCaseField:   cfg.Python.SnakeCaseField,
			AllowExtraFields: cfg.Python.AdditionalProperties,
		}),
		&py.ImportConverter{},
	)
}

func newGoGenerator(cfg *Config) Generator {
	return newCombinedGenerator(
		golang.NewGenerator(&golang.Config{
			PackageName:      cfg.Go.PackageName,
			UsePointers:      cfg.Go.UsePointers,
			OmitEmpty:        cfg.Go.OmitEmpty,
			DisableHeaders:   cfg.DisableHeaders,
			DisableTimestamp: cfg.DisableTimestamp,
		}),
		&golang.ImportConverter{ModulePath: cfg.Go.ModulePath},
	)
}
