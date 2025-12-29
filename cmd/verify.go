package cmd

import (
	"fmt"

	"github.com/mirpo/schemagen/pkg/compare"
	"github.com/mirpo/schemagen/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify [input]",
		Short: "Verify generated code matches schema (CI mode)",
		Long: `Verify that generated code matches schemas without modifying files.

Generates code to a temporary directory and compares with existing files.
Useful for CI pipelines to detect manual edits or drift.

Exit codes:
  0 - Files match (no drift)
  1 - Error during verification
  2 - Files don't match (drift detected)`,
		Example: `  schemagen verify ./schemas --out-ts ./types
  schemagen verify ./schemas --out-ts ./ts --out-py ./py --quiet`,
		Args:         cobra.MinimumNArgs(1),
		RunE:         runVerify,
		SilenceUsage: true,
	}

	AddGenerationFlags(cmd)
	cmd.MarkFlagsOneRequired("out-ts", "out-py", "out-go")
	cmd.Flags().Bool("quiet", false, "Suppress output (only exit codes)")

	return cmd
}

func runVerify(cmd *cobra.Command, args []string) error {
	quiet, _ := cmd.Flags().GetBool("quiet")
	input := args[0]
	flags := GetGenerationFlags(cmd)

	// Run comparison
	result, err := compare.Run(&compare.Config{
		Input: input,
		Flags: flags,
	})
	if err != nil {
		return errors.Wrap(err, "verification failed")
	}

	// Log differences if not quiet
	if !quiet && len(result.Diffs) > 0 {
		for _, diff := range result.Diffs {
			switch diff.Status {
			case compare.StatusNew:
				log.Warn().Str("file", diff.Path).Msg("File exists in generated but not in existing")
			case compare.StatusDeleted:
				log.Warn().Str("file", diff.Path).Msg("File exists in existing but not in generated")
			case compare.StatusModified:
				log.Warn().Str("file", diff.Path).Msg("File content differs")
			}
		}
	}

	// Handle result
	if result.HasDrift {
		if !quiet {
			fmt.Println()
			fmt.Println("❌ Drift detected! Generated files don't match existing files.")
			fmt.Println("Run 'schemagen generate' to update generated files.")
		}
		return &errors.ExitCodeError{
			Message: "drift detected",
			Code:    2,
		}
	}

	if !quiet {
		fmt.Println()
		fmt.Println("✅ No drift detected! Generated files match existing files.")
	}
	return nil
}
