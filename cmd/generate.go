package cmd

import (
	"fmt"

	"github.com/mirpo/schemagen/pkg/config"
	"github.com/mirpo/schemagen/pkg/generation"
	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate [input]",
		Short: "Generate TypeScript, Python, and Go code from JSON Schema",
		Long: `Generate code from JSON Schema files.

Supports TypeScript interfaces, Python Pydantic v2 models, and Go structs.
Input can be a single file, directory, or glob pattern.`,
		Args: cobra.MinimumNArgs(1),
		RunE: runGenerate,
	}

	// Add shared generation flags
	AddGenerationFlags(cmd)

	return cmd
}

func runGenerate(cmd *cobra.Command, args []string) error {
	input := args[0]

	// Get flags using shared helper
	flags := GetGenerationFlags(cmd)

	// Need at least one output
	if flags.OutTS == "" && flags.OutPY == "" && flags.OutGo == "" {
		log.Warn().Msg("No output specified. Use --out-ts, --out-py, and/or --out-go")
		return nil
	}

	// Load schemas using new pkg/schema loader
	log.Info().Str("input", input).Msg("Loading schemas")
	loader := schema.NewLoader()
	schemas, err := loader.Load(input)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load schemas")
		return fmt.Errorf("loading schemas: %w", err)
	}
	log.Info().Int("count", len(schemas)).Msg("Loaded schemas")

	// Generate TypeScript if requested
	if flags.OutTS != "" {
		cfg := config.ToGenerationConfig(flags, schemas, loader.Compiler(), flags.OutTS, generation.LanguageTypeScript)
		if err := generation.Run(cfg); err != nil {
			log.Error().Err(err).Msg("TypeScript generation failed")
			return fmt.Errorf("generating TypeScript: %w", err)
		}
		log.Info().Str("dir", flags.OutTS).Msg("TypeScript generation complete")
	}

	// Generate Python if requested
	if flags.OutPY != "" {
		cfg := config.ToGenerationConfig(flags, schemas, loader.Compiler(), flags.OutPY, generation.LanguagePython)
		if err := generation.Run(cfg); err != nil {
			log.Error().Err(err).Msg("Python generation failed")
			return fmt.Errorf("generating Python: %w", err)
		}
		log.Info().Str("dir", flags.OutPY).Msg("Python generation complete")
	}

	// Generate Go if requested
	if flags.OutGo != "" {
		cfg := config.ToGenerationConfig(flags, schemas, loader.Compiler(), flags.OutGo, generation.LanguageGo)
		if err := generation.Run(cfg); err != nil {
			log.Error().Err(err).Msg("Go generation failed")
			return fmt.Errorf("generating Go: %w", err)
		}
		log.Info().Str("dir", flags.OutGo).Msg("Go generation complete")
	}

	return nil
}
