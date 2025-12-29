package generation

import (
	"fmt"
	"path/filepath"

	pkgerrors "github.com/mirpo/schemagen/pkg/errors"
	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/rs/zerolog/log"
)

// Run executes the complete generation pipeline for a given language
func Run(cfg *Config) error {
	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Step 1: Build type graph
	graph, err := buildTypeGraph(cfg)
	if err != nil {
		return fmt.Errorf("building type graph: %w", err)
	}

	log.Debug().
		Str("language", string(cfg.Language)).
		Int("types", len(graph.Types)).
		Msg("Built type graph")

	// Step 2: Plan output files
	plan, err := planOutput(graph, cfg)
	if err != nil {
		return fmt.Errorf("planning output: %w", err)
	}

	// Step 3: Compute imports between files
	typeToFile := output.BuildTypeToFileMap(plan.Files)
	plan.Files = output.ComputeImports(plan.Files, typeToFile)

	// Step 4: Create file writer
	writer := NewDiskWriter(cfg.OutDir)

	// Step 5: Create output directory
	if err := writer.MakeDirectory(""); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Step 6: Generate and write files
	if err := generateFiles(graph, plan, cfg, writer); err != nil {
		return fmt.Errorf("generating files: %w", err)
	}

	return nil
}

// buildTypeGraph creates a type graph from schemas with language-specific configuration
func buildTypeGraph(cfg *Config) (*typegraph.Graph, error) {
	// Determine extraction behavior based on language
	extractInlined := cfg.ExtractInline

	// Python and Go always extract inline types
	// - Python: Pydantic doesn't support inline TypedDict
	// - Go: doesn't support inline anonymous structs idiomatically
	if cfg.Language == LanguagePython || cfg.Language == LanguageGo {
		extractInlined = true
	}

	builder := typegraph.NewBuilderWithConfig(cfg.Compiler, &typegraph.BuildConfig{
		ExtractInlined: extractInlined,
	})

	return builder.Build(cfg.Schemas)
}

// planOutput determines which types go into which files
func planOutput(graph *typegraph.Graph, cfg *Config) (*output.OutputPlan, error) {
	// Determine file extension based on language
	var ext string
	switch cfg.Language {
	case LanguageTypeScript:
		ext = "ts"
	case LanguagePython:
		ext = "py"
	case LanguageGo:
		ext = "go"
	default:
		return nil, fmt.Errorf("unsupported language: %s", cfg.Language)
	}

	// Use the strategy from config (already normalized)
	strategy := cfg.OutputStrategy
	if strategy == "" {
		strategy = output.StrategyBundle // Default to bundle
	}

	// bundleName only used by Bundle and BundleDeps strategies
	bundleName := "types"

	return output.PlanOutput(graph, cfg.Schemas, strategy, ext, bundleName)
}

// generateFiles generates code for all planned files and writes them to disk
func generateFiles(graph *typegraph.Graph, plan *output.OutputPlan, cfg *Config, writer FileWriter) error {
	// Create language-specific generator
	generator, err := createGenerator(graph, cfg)
	if err != nil {
		return err
	}

	// Generate each file
	for _, file := range plan.Files {
		// Convert imports to language-specific format
		langImports := generator.ConvertImports(file.Imports)

		// Generate code
		code, err := generator.Generate(file.Types, langImports)
		if err != nil {
			return &pkgerrors.GenerationError{
				Language: string(cfg.Language),
				File:     file.RelativePath,
				Message:  "generate code",
				Cause:    err,
			}
		}

		// Write file
		if err := writer.WriteFile(file.RelativePath, []byte(code)); err != nil {
			return err
		}

		log.Debug().
			Str("language", string(cfg.Language)).
			Str("file", filepath.Join(cfg.OutDir, file.RelativePath)).
			Msg("Generated file")
	}

	// After writing all regular files, generate barrel files for multifile strategy
	if cfg.OutputStrategy == output.StrategyMultiFile {
		if err := generateBarrelFiles(plan, cfg, writer); err != nil {
			return fmt.Errorf("generating barrel files: %w", err)
		}
	}

	return nil
}

// generateBarrelFiles generates and writes barrel/index files for multifile output
func generateBarrelFiles(plan *output.OutputPlan, cfg *Config, writer FileWriter) error {
	langStr := string(cfg.Language)

	// Generate barrel file specs
	barrels := output.GenerateBarrelFiles(plan, langStr)

	// Write each barrel file
	for _, barrel := range barrels {
		content := output.GenerateBarrelContent(barrel, langStr)

		// Write file
		if err := writer.WriteFile(barrel.Path, []byte(content)); err != nil {
			return err
		}

		log.Debug().
			Str("language", langStr).
			Str("file", filepath.Join(cfg.OutDir, barrel.Path)).
			Msg("Generated barrel file")
	}

	return nil
}

// validateConfig ensures the configuration is valid
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

	// Validate language-specific config
	switch cfg.Language {
	case LanguageTypeScript:
		if cfg.TypeScript == nil {
			cfg.TypeScript = &TypeScriptConfig{} // Use defaults
		}
	case LanguagePython:
		if cfg.Python == nil {
			cfg.Python = &PythonConfig{} // Use defaults
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
		return &pkgerrors.ValidationError{Field: "language", Message: fmt.Sprintf("unsupported language: %s", cfg.Language)}
	}

	return nil
}
