package cmd

import (
	"github.com/mirpo/schemagen/pkg/errors"
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
		Example: `  schemagen generate ./schemas --out-ts ./types
  schemagen generate ./api/*.json --out-py ./models --out-go ./pkg/models`,
		Args:         cobra.MinimumNArgs(1),
		RunE:         runGenerate,
		SilenceUsage: true,
	}

	AddGenerationFlags(cmd)
	cmd.MarkFlagsOneRequired("out-ts", "out-py", "out-go")

	return cmd
}

func runGenerate(cmd *cobra.Command, args []string) error {
	input := args[0]
	flags := GetGenerationFlags(cmd)

	// Load schemas
	log.Info().Str("input", input).Msg("Loading schemas")
	loader := schema.NewLoader()
	schemas, err := loader.Load(input)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load schemas")
		return errors.Wrap(err, "loading schemas")
	}
	log.Info().Int("count", len(schemas)).Msg("Loaded schemas")

	// Generate TypeScript if requested
	if flags.OutTS != "" {
		cfg := generation.ConfigFromFlags(flags, schemas, loader.Compiler(), flags.OutTS, generation.LanguageTypeScript)
		if err := generation.Run(cfg); err != nil {
			log.Error().Err(err).Msg("TypeScript generation failed")
			return errors.Wrap(err, "generating TypeScript")
		}
		log.Info().Str("dir", flags.OutTS).Msg("TypeScript generation complete")
	}

	// Generate Python if requested
	if flags.OutPY != "" {
		cfg := generation.ConfigFromFlags(flags, schemas, loader.Compiler(), flags.OutPY, generation.LanguagePython)
		if err := generation.Run(cfg); err != nil {
			log.Error().Err(err).Msg("Python generation failed")
			return errors.Wrap(err, "generating Python")
		}
		log.Info().Str("dir", flags.OutPY).Msg("Python generation complete")
	}

	// Generate Go if requested
	if flags.OutGo != "" {
		cfg := generation.ConfigFromFlags(flags, schemas, loader.Compiler(), flags.OutGo, generation.LanguageGo)
		if err := generation.Run(cfg); err != nil {
			log.Error().Err(err).Msg("Go generation failed")
			return errors.Wrap(err, "generating Go")
		}
		log.Info().Str("dir", flags.OutGo).Msg("Go generation complete")
	}

	return nil
}
