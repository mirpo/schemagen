package cmd

import (
	"os"

	"github.com/mirpo/schemagen/pkg/errors"
	"github.com/mirpo/schemagen/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "none"
)

// NewRootCmd creates a new root command instance for testing isolation
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schemagen",
		Short: "Generate TypeScript, Python, and Go from JSON Schema",
		Long: `schemagen is a fast CLI tool that converts JSON Schema into
TypeScript interfaces, Python Pydantic v2 models, and Go structs.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			jsonOutput, _ := cmd.Flags().GetBool("json")

			logger.ConfigLogger(logger.LoggerConfig{
				Verbose:    verbose,
				JSONOutput: jsonOutput,
			})
		},
	}

	// Add persistent flags
	cmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose/debug logging")
	cmd.PersistentFlags().Bool("json", false, "Output logs in JSON format")

	// Add subcommands
	cmd.AddCommand(newGenerateCmd())
	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newVerifyCmd())
	cmd.AddCommand(newDiffCmd())
	cmd.AddCommand(newVersionCmd())

	return cmd
}

// rootCmd is the global instance for production use
var rootCmd = NewRootCmd()

func Execute() error {
	err := rootCmd.Execute()
	if exitErr, ok := err.(*errors.ExitCodeError); ok {
		os.Exit(exitErr.Code)
	}
	return err
}
