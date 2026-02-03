package generation

import (
	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/schema"
)

// GenerationFlags holds all code generation flags from CLI.
type GenerationFlags struct {
	// Output directories
	OutTS string
	OutPY string
	OutGo string

	// Universal flags
	ExtractInline    bool
	DisableHeaders   bool
	DisableTimestamp bool
	OutputStrategy   output.OutputStrategy

	// TypeScript-specific flags
	TSUnknownAny           bool
	TSAdditionalProperties bool

	// TypeScript Zod integration flags
	TSZod            bool
	TSZodOnly        bool
	TSZodCoerceDates bool
	TSZodStrict      bool

	// Python-specific flags
	PySnakeCaseField       bool
	PyAdditionalProperties bool

	// Go-specific flags
	GoPackageName string
	GoUsePointers bool
	GoOmitEmpty   bool
	GoModulePath  string
}

// ConfigFromFlags creates a Config from GenerationFlags for a specific language.
func ConfigFromFlags(f *GenerationFlags, schemas []*schema.Schema, compiler *jsonschema.Compiler, outDir string, lang Language) *Config {
	cfg := &Config{
		Schemas:          schemas,
		Compiler:         compiler,
		OutDir:           outDir,
		Language:         lang,
		ExtractInline:    f.ExtractInline,
		DisableHeaders:   f.DisableHeaders,
		DisableTimestamp: f.DisableTimestamp,
		OutputStrategy:   f.OutputStrategy,
	}

	switch lang {
	case LanguageTypeScript:
		cfg.TypeScript = &TypeScriptConfig{
			UnknownAny:           f.TSUnknownAny,
			AdditionalProperties: f.TSAdditionalProperties,
			Zod:                  f.TSZod,
			ZodOnly:              f.TSZodOnly,
			ZodCoerceDates:       f.TSZodCoerceDates,
			ZodStrict:            f.TSZodStrict,
		}
	case LanguagePython:
		cfg.Python = &PythonConfig{
			SnakeCaseField:       f.PySnakeCaseField,
			AdditionalProperties: f.PyAdditionalProperties,
		}
	case LanguageGo:
		cfg.Go = &GoConfig{
			PackageName: f.GoPackageName,
			UsePointers: f.GoUsePointers,
			OmitEmpty:   f.GoOmitEmpty,
			ModulePath:  f.GoModulePath,
		}
	}

	return cfg
}
