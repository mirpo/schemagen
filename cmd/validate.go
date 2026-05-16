package cmd

import (
	"fmt"

	"github.com/mirpo/schemagen/pkg/errors"
	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [input]",
		Short: "Validate JSON Schema files without generating code",
		Long: `Validate JSON Schema files for syntax, spec compliance, and reference resolution.

This command checks schemas without generating any code, making it fast and suitable
for pre-commit hooks and CI pipelines.

Exit codes:
  0 - All schemas valid
  1 - Validation errors found`,
		Example: `  schemagen validate ./schemas
  schemagen validate ./api/*.json --format json`,
		Args:         cobra.ExactArgs(1),
		RunE:         runValidate,
		SilenceUsage: true,
	}

	cmd.Flags().String("format", "text", "Output format: text or json")

	return cmd
}

func runValidate(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	// Use pkg/schema loader to validate schemas
	// If it can load and parse them, they're valid
	loader := schema.NewLoader()
	schemas, err := loader.Load(inputPath)
	if err != nil {
		log.Error().Err(err).Str("input", inputPath).Msg("Schema validation failed")
		return errors.Wrap(err, "validation failed")
	}

	log.Info().Int("count", len(schemas)).Msg("All schemas are valid")

	format, _ := cmd.Flags().GetString("format")
	if format == "json" {
		fmt.Printf(`{"valid":true,"count":%d}%s`, len(schemas), "\n")
	} else {
		fmt.Printf("✓ All %d schema(s) are valid\n", len(schemas))
	}

	return nil
}
