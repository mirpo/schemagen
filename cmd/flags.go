package cmd

import (
	"github.com/mirpo/schemagen/pkg/generation"
	"github.com/mirpo/schemagen/pkg/output"
	"github.com/spf13/cobra"
)

// AddGenerationFlags adds all generation-related flags to a command
func AddGenerationFlags(cmd *cobra.Command) {
	// Output directory flags
	cmd.Flags().String("out-ts", "", "Output directory for TypeScript")
	cmd.Flags().String("out-py", "", "Output directory for Python")
	cmd.Flags().String("out-go", "", "Output directory for Go")

	// Type extraction flag
	cmd.Flags().Bool("extract-inline", false, "Extract inline enums and nested objects to top-level types")

	// Header control flags
	cmd.Flags().Bool("disable-headers", false, "Disable all generated file headers")
	cmd.Flags().Bool("disable-timestamp", false, "Disable timestamp in generated file headers")

	// Output strategy flag
	cmd.Flags().String("output-strategy", "multifile", "Output file strategy: 'bundle', 'multifile', 'bundledeps', 'bundle-per-dir'")

	// TypeScript feature flags
	cmd.Flags().Bool("ts-unknown-any", false, "Use 'unknown' instead of 'any' for untyped schemas (TypeScript)")
	cmd.Flags().Bool("ts-additional-properties", false, "Add index signatures for additionalProperties (TypeScript)")

	// TypeScript Zod integration flags
	cmd.Flags().Bool("ts-zod", false, "Generate Zod schemas alongside TypeScript interfaces")
	cmd.Flags().Bool("ts-zod-only", false, "Generate only Zod schemas (no interfaces, use z.infer for types)")
	cmd.Flags().Bool("ts-zod-coerce-dates", false, "Use z.coerce.date() instead of z.iso.datetime() for date-time format")
	cmd.Flags().Bool("ts-zod-strict", false, "Add .strict() to all Zod object schemas")

	// Python feature flags
	cmd.Flags().Bool("py-snake-case-field", false, "Convert Python field names to snake_case with JSON alias")
	cmd.Flags().Bool("py-additional-properties", false, "Add model_config with extra='allow' for additionalProperties (Python)")

	// Go feature flags
	cmd.Flags().String("go-package", "models", "Go package name for generated files")
	cmd.Flags().Bool("go-pointers", true, "Use pointers for optional Go fields")
	cmd.Flags().Bool("go-omit-empty", true, "Add omitempty to optional Go JSON tags")
	cmd.Flags().String("go-module-path", "", "Go module path for absolute imports (e.g., github.com/org/project)")
}

// GetGenerationFlags extracts generation flags from a cobra command.
func GetGenerationFlags(cmd *cobra.Command) *generation.GenerationFlags {
	flags := &generation.GenerationFlags{}

	flags.OutTS, _ = cmd.Flags().GetString("out-ts")
	flags.OutPY, _ = cmd.Flags().GetString("out-py")
	flags.OutGo, _ = cmd.Flags().GetString("out-go")

	flags.ExtractInline, _ = cmd.Flags().GetBool("extract-inline")
	flags.DisableHeaders, _ = cmd.Flags().GetBool("disable-headers")
	flags.DisableTimestamp, _ = cmd.Flags().GetBool("disable-timestamp")

	strategyStr, _ := cmd.Flags().GetString("output-strategy")
	flags.OutputStrategy = output.ParseStrategy(strategyStr)

	flags.TSUnknownAny, _ = cmd.Flags().GetBool("ts-unknown-any")
	flags.TSAdditionalProperties, _ = cmd.Flags().GetBool("ts-additional-properties")

	// Zod flags
	flags.TSZod, _ = cmd.Flags().GetBool("ts-zod")
	flags.TSZodOnly, _ = cmd.Flags().GetBool("ts-zod-only")
	flags.TSZodCoerceDates, _ = cmd.Flags().GetBool("ts-zod-coerce-dates")
	flags.TSZodStrict, _ = cmd.Flags().GetBool("ts-zod-strict")

	flags.PySnakeCaseField, _ = cmd.Flags().GetBool("py-snake-case-field")
	flags.PyAdditionalProperties, _ = cmd.Flags().GetBool("py-additional-properties")

	flags.GoPackageName, _ = cmd.Flags().GetString("go-package")
	flags.GoUsePointers, _ = cmd.Flags().GetBool("go-pointers")
	flags.GoOmitEmpty, _ = cmd.Flags().GetBool("go-omit-empty")
	flags.GoModulePath, _ = cmd.Flags().GetString("go-module-path")

	return flags
}
