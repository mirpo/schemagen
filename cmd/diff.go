package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/kylelemons/godebug/diff"
	"github.com/mirpo/schemagen/pkg/compare"
	"github.com/spf13/cobra"
)

func newDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff [input]",
		Short: "Show diff between generated code and existing code",
		Long: `Show differences between generated code and existing files.

Generates code to a temporary directory and shows a unified diff.
Useful for reviewing what would change before running generate.

Exit codes:
  0 - No differences
  1 - Error during diff
  2 - Differences found`,
		Args:         cobra.MinimumNArgs(1),
		RunE:         runDiff,
		SilenceUsage: true, // Don't show usage on expected errors
	}

	// Add shared generation flags
	AddGenerationFlags(cmd)

	// Add diff-specific flag
	cmd.Flags().Bool("no-color", false, "Disable colored output")

	return cmd
}

func runDiff(cmd *cobra.Command, args []string) error {
	noColor, _ := cmd.Flags().GetBool("no-color")
	input := args[0]

	// Disable color if requested
	if noColor {
		color.NoColor = true
	}

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
		return fmt.Errorf("diff failed: %w", err)
	}

	// Show diffs if any
	if len(result.Diffs) == 0 {
		fmt.Println()
		color.Green("✓ No differences")
		return nil
	}

	// Show detailed diffs
	for _, fileDiff := range result.Diffs {
		switch fileDiff.Status {
		case compare.StatusNew:
			color.Green("\n+ %s (new file)", fileDiff.Path)
			fmt.Println(color.GreenString(fileDiff.NewContent))

		case compare.StatusDeleted:
			color.Red("\n- %s (deleted)", fileDiff.Path)
			fmt.Println(color.RedString(fileDiff.OldContent))

		case compare.StatusModified:
			color.Yellow("\n~ %s", fileDiff.Path)
			showUnifiedDiff(fileDiff.OldContent, fileDiff.NewContent, fileDiff.Path)
		}
	}

	fmt.Println()
	color.Red("✗ Differences found")
	return ExitCodeError{
		Message: "differences found",
		Code:    2,
	}
}

// showUnifiedDiff shows a unified diff with context lines
func showUnifiedDiff(old, new, filename string) {
	// Use godebug/diff to generate line-by-line diff
	diffOutput := diff.Diff(old, new)

	if diffOutput == "" {
		return
	}

	// Colorize the diff output
	lines := strings.Split(diffOutput, "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// Colorize based on prefix
		switch {
		case strings.HasPrefix(line, "-"):
			color.Red(line)
		case strings.HasPrefix(line, "+"):
			color.Green(line)
		default:
			// Context line (starts with space)
			fmt.Println(line)
		}
	}
	fmt.Println()
}
