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
		Example: `  schemagen diff ./schemas --out-ts ./types
  schemagen diff ./schemas --out-py ./models --no-color`,
		Args:         cobra.ExactArgs(1),
		RunE:         runDiff,
		SilenceUsage: true,
	}

	AddGenerationFlags(cmd)
	cmd.Flags().Bool("no-color", false, "Disable colored output")

	return cmd
}

func runDiff(cmd *cobra.Command, args []string) error {
	noColor, _ := cmd.Flags().GetBool("no-color")
	input := args[0]

	if noColor {
		old := color.NoColor
		color.NoColor = true
		defer func() { color.NoColor = old }()
	}

	result, err := runCompare(cmd, input)
	if err != nil {
		return wrapErr(err, "diff failed")
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
			showUnifiedDiff(fileDiff.OldContent, fileDiff.NewContent)
		}
	}

	fmt.Println()
	color.Red("✗ Differences found")
	return &exitCodeError{
		Message: "differences found",
		Code:    exitDrift,
	}
}

func showUnifiedDiff(old, new string) {
	diffOutput := diff.Diff(old, new)

	if diffOutput == "" {
		return
	}

	lines := strings.Split(diffOutput, "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		switch {
		case strings.HasPrefix(line, "-"):
			fmt.Println(color.RedString("%s", line))
		case strings.HasPrefix(line, "+"):
			fmt.Println(color.GreenString("%s", line))
		default:
			fmt.Println(line)
		}
	}
	fmt.Println()
}
