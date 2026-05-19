package pipeline

import (
	"fmt"

	"github.com/mirpo/schemagen/pkg/graph"
	"github.com/mirpo/schemagen/pkg/render/golang"
	"github.com/mirpo/schemagen/pkg/render/py"
	"github.com/mirpo/schemagen/pkg/render/ts"
)

type generator interface {
	Generate(types []*graph.Type, imports []graph.ImportSpec) (string, error)
	ConvertImports(imports []graph.ImportSpec) []graph.ImportSpec
}

type langGenerator struct {
	generateFile  func([]*graph.Type, []graph.ImportSpec) (string, error)
	convertImport func([]graph.ImportSpec) []graph.ImportSpec
}

func (g *langGenerator) Generate(types []*graph.Type, imports []graph.ImportSpec) (string, error) {
	return g.generateFile(types, imports)
}

func (g *langGenerator) ConvertImports(imports []graph.ImportSpec) []graph.ImportSpec {
	if g.convertImport == nil {
		return imports
	}
	return g.convertImport(imports)
}

func createGenerator(cfg *Config) (generator, error) {
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

func newTypeScriptGenerator(cfg *Config) generator {
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

	gen := ts.NewGeneratorWithConfig(tsCfg)
	return &langGenerator{
		generateFile: gen.GenerateFile,
	}
}

func newPythonGenerator(cfg *Config) generator {
	gen := py.NewGeneratorWithConfig(&py.Config{
		DisableHeaders:   cfg.DisableHeaders,
		DisableTimestamp: cfg.DisableTimestamp,
		SnakeCaseField:   cfg.Python.SnakeCaseField,
		AllowExtraFields: cfg.Python.AdditionalProperties,
	})
	conv := &py.ImportConverter{}
	return &langGenerator{
		generateFile:  gen.GenerateFile,
		convertImport: conv.Convert,
	}
}

func newGoGenerator(cfg *Config) generator {
	gen := golang.NewGenerator(&golang.Config{
		PackageName:      cfg.Go.PackageName,
		UsePointers:      cfg.Go.UsePointers,
		OmitEmpty:        cfg.Go.OmitEmpty,
		DisableHeaders:   cfg.DisableHeaders,
		DisableTimestamp: cfg.DisableTimestamp,
	})
	conv := &golang.ImportConverter{ModulePath: cfg.Go.ModulePath}
	return &langGenerator{
		generateFile:  gen.GenerateFile,
		convertImport: conv.Convert,
	}
}
