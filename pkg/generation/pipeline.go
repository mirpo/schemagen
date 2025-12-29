package generation

import (
	pkgerrors "github.com/mirpo/schemagen/pkg/errors"
	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/rs/zerolog/log"
)

func Run(cfg *Config) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}

	graph, err := buildTypeGraph(cfg)
	if err != nil {
		return err
	}

	log.Debug().
		Str("language", string(cfg.Language)).
		Int("types", len(graph.Types)).
		Msg("Built type graph")

	plan, err := planOutput(graph, cfg)
	if err != nil {
		return err
	}

	typeToFile := output.BuildTypeToFileMap(plan.Files)
	plan.Files = output.ComputeImports(plan.Files, typeToFile)

	writer := NewDiskWriter(cfg.OutDir)

	if err := writer.MakeDirectory(""); err != nil {
		return err
	}

	if err := generateFiles(graph, plan, cfg, writer); err != nil {
		return err
	}

	return nil
}

func buildTypeGraph(cfg *Config) (*typegraph.Graph, error) {
	extractInline := cfg.ExtractInline

	if cfg.Language == LanguagePython || cfg.Language == LanguageGo {
		extractInline = true
	}

	builder := typegraph.NewBuilderWithConfig(cfg.Compiler, &typegraph.BuildConfig{
		ExtractInlined: extractInline,
	})

	return builder.Build(cfg.Schemas)
}

func planOutput(graph *typegraph.Graph, cfg *Config) (*output.OutputPlan, error) {
	var ext string

	switch cfg.Language {
	case LanguageTypeScript:
		ext = "ts"
	case LanguagePython:
		ext = "py"
	case LanguageGo:
		ext = "go"
	default:
		return nil, &pkgerrors.ValidationError{
			Field:   "language",
			Message: "unsupported language",
		}
	}

	return output.PlanOutput(
		graph,
		cfg.Schemas,
		cfg.OutputStrategy,
		ext,
		"types",
	)
}

func generateFiles(
	graph *typegraph.Graph,
	plan *output.OutputPlan,
	cfg *Config,
	writer FileWriter,
) error {
	generator, err := createGenerator(graph, cfg)
	if err != nil {
		return err
	}

	for _, file := range plan.Files {
		langImports := generator.ConvertImports(file.Imports)

		code, err := generator.Generate(file.Types, langImports)
		if err != nil {
			return &pkgerrors.GenerationError{
				Language: string(cfg.Language),
				File:     file.RelativePath,
				Message:  "generate code",
				Cause:    err,
			}
		}

		if err := writer.WriteFile(file.RelativePath, []byte(code)); err != nil {
			return err
		}

		log.Debug().
			Str("language", string(cfg.Language)).
			Str("file", file.RelativePath).
			Msg("Generated file")
	}

	if cfg.OutputStrategy == output.StrategyMultiFile {
		return generateBarrelFiles(plan, cfg, writer)
	}

	return nil
}

func generateBarrelFiles(plan *output.OutputPlan, cfg *Config, writer FileWriter) error {
	lang := string(cfg.Language)

	barrels := output.GenerateNestedBarrels(plan.Files, lang)

	for _, barrel := range barrels {
		content := output.GenerateBarrelContent(barrel, lang)

		if err := writer.WriteFile(barrel.Path, []byte(content)); err != nil {
			return err
		}

		log.Debug().
			Str("language", lang).
			Str("file", barrel.Path).
			Msg("Generated barrel file")
	}

	return nil
}

func validateConfig(cfg *Config) error {
	if cfg == nil {
		return &pkgerrors.ValidationError{Field: "config", Message: "config is nil"}
	}
	if len(cfg.Schemas) == 0 {
		return &pkgerrors.ValidationError{Field: "schemas", Message: "no schemas provided"}
	}
	if cfg.Compiler == nil {
		return &pkgerrors.ValidationError{Field: "compiler", Message: "compiler is nil"}
	}
	if cfg.OutDir == "" {
		return &pkgerrors.ValidationError{Field: "output-directory", Message: "output directory is empty"}
	}
	if cfg.Language == "" {
		return &pkgerrors.ValidationError{Field: "language", Message: "language not specified"}
	}

	if cfg.OutputStrategy == "" {
		cfg.OutputStrategy = output.StrategyBundle
	}

	switch cfg.Language {
	case LanguageTypeScript:
		if cfg.TypeScript == nil {
			cfg.TypeScript = &TypeScriptConfig{}
		}
	case LanguagePython:
		if cfg.Python == nil {
			cfg.Python = &PythonConfig{}
		}
	case LanguageGo:
		if cfg.Go == nil {
			cfg.Go = &GoConfig{
				PackageName: "models",
				UsePointers: true,
				OmitEmpty:   true,
			}
		}
	default:
		return &pkgerrors.ValidationError{
			Field:   "language",
			Message: "unsupported language",
		}
	}

	return nil
}
