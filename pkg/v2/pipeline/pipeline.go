package pipeline

import (
	pkgerrors "github.com/mirpo/schemagen/pkg/errors"
	"github.com/mirpo/schemagen/pkg/v2/graph"
	"github.com/mirpo/schemagen/pkg/v2/output"
	"github.com/mirpo/schemagen/pkg/v2/parse"
	"github.com/rs/zerolog/log"
)

func Run(cfg *Config) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}
	applyDefaults(cfg)

	g, err := graph.Build(cfg.Schemas, graph.BuildConfig{
		ExtractInlined: cfg.ExtractInline,
	})
	if err != nil {
		return err
	}

	log.Debug().
		Str("language", string(cfg.Language)).
		Int("types", len(g.Types)).
		Msg("Built type graph")

	schemas := make([]parse.NamedSchema, len(cfg.Schemas))
	for i, s := range cfg.Schemas {
		schemas[i] = *s
	}

	plan, err := output.PlanOutput(
		g,
		schemas,
		cfg.OutputStrategy,
		cfg.Language,
		DefaultBundleName,
	)
	if err != nil {
		return err
	}

	typeToFile := output.BuildTypeToFileMap(plan.Files)
	plan.Files = output.ComputeImports(plan.Files, typeToFile)

	writer := NewDiskWriter(cfg.OutDir)

	if err := generateFiles(g, plan, cfg, writer); err != nil {
		return err
	}

	return nil
}

func generateFiles(
	g *graph.Graph,
	plan *output.OutputPlan,
	cfg *Config,
	writer FileWriter,
) error {
	generator, err := createGenerator(cfg)
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
	barrels := output.GenerateNestedBarrels(plan.Files, cfg.Language)

	for _, barrel := range barrels {
		content := output.GenerateBarrelContent(barrel, cfg.Language)

		if err := writer.WriteFile(barrel.Path, []byte(content)); err != nil {
			return err
		}

		log.Debug().
			Str("language", string(cfg.Language)).
			Str("file", barrel.Path).
			Msg("Generated barrel file")
	}

	return nil
}

func applyDefaults(cfg *Config) {
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
		cfg.ExtractInline = true
	case LanguageGo:
		cfg.ExtractInline = true
		if cfg.Go == nil {
			cfg.Go = &GoConfig{
				PackageName: "models",
				UsePointers: true,
				OmitEmpty:   true,
			}
		}
	}
}

func validateConfig(cfg *Config) error {
	if cfg == nil {
		return &pkgerrors.ValidationError{Field: "config", Message: "config is nil"}
	}
	if len(cfg.Schemas) == 0 {
		return &pkgerrors.ValidationError{Field: "schemas", Message: "no schemas provided"}
	}
	if cfg.OutDir == "" {
		return &pkgerrors.ValidationError{Field: "output-directory", Message: "output directory is empty"}
	}
	if cfg.Language == "" {
		return &pkgerrors.ValidationError{Field: "language", Message: "language not specified"}
	}

	switch cfg.Language {
	case LanguageTypeScript, LanguagePython, LanguageGo:
	default:
		return &pkgerrors.ValidationError{
			Field:   "language",
			Message: "unsupported language",
		}
	}

	return nil
}
