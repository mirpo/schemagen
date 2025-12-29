package cmd

import (
	"fmt"

	"github.com/mirpo/schemagen/pkg/compare"
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
		Args:         cobra.MinimumNArgs(1),
		RunE:         runVerify,
		SilenceUsage: true, // Don't show usage on expected errors
	}

	// Add shared generation flags
	AddGenerationFlags(cmd)

	// Add verify-specific flag
	cmd.Flags().Bool("quiet", false, "Suppress output (only exit codes)")

	return cmd
}

func runVerify(cmd *cobra.Command, args []string) error {
	quiet, _ := cmd.Flags().GetBool("quiet")
	input := args[0]

	// Get generation flags using shared helper
	flags := GetGenerationFlags(cmd)

	// Validate at least one output directory is specified
	if flags.OutTS == "" && flags.OutPY == "" && flags.OutGo == "" {
		return fmt.Errorf("at least one output directory (--out-ts, --out-py, or --out-go) must be specified")
	}

	// Run comparison using pkg/compare
	result, err := compare.Run(&compare.Config{
		Input: input,
		Flags: flags,
	})
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
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
		return ExitCodeError{
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
