package cmd

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "none"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schemagen",
		Short: "Generate TypeScript, Python, and Go from JSON Schema",
		Long: `schemagen is a fast CLI tool that converts JSON Schema into
TypeScript interfaces, Python Pydantic v2 models, and Go structs.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			jsonOutput, _ := cmd.Flags().GetBool("json")

			zerolog.SetGlobalLevel(zerolog.InfoLevel)
			if verbose {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			}
			if !jsonOutput {
				log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
			}
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

func Execute() error {
	return NewRootCmd().Execute()
}
