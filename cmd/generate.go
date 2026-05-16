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
		Args:         cobra.ExactArgs(1),
		RunE:         runGenerate,
		SilenceUsage: true,
	}

	AddGenerationFlags(cmd)

	return cmd
}

func runGenerate(cmd *cobra.Command, args []string) error {
	input := args[0]
	flags := GetGenerationFlags(cmd)

	log.Info().Str("input", input).Msg("Loading schemas")
	loader := schema.NewLoader()
	schemas, err := loader.Load(input)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load schemas")
		return errors.Wrap(err, "loading schemas")
	}
	log.Info().Int("count", len(schemas)).Msg("Loaded schemas")

	targets := generation.BuildTargets(flags)
	for _, t := range targets {
		cfg := generation.ConfigFromFlags(flags, schemas, loader.Compiler(), t.Dir, t.Lang)
		if err := generation.Run(cfg); err != nil {
			log.Error().Err(err).Str("lang", string(t.Lang)).Msg("Generation failed")
			return errors.Wrap(err, "generating "+string(t.Lang))
		}
		log.Info().Str("dir", t.Dir).Str("lang", string(t.Lang)).Msg("Generation complete")
	}

	return nil
}
