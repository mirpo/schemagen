package config

import (
	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/generation"
	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/schema"
)

// GenerationFlags holds all code generation flags
type GenerationFlags struct {
	// Output directories
	OutTS string
	OutPY string
	OutGo string

	// Universal flags
	ExtractInline    bool
	DisableHeaders   bool
	DisableTimestamp bool
	OutputStrategy   string

	// TypeScript-specific flags
	TSUnknownAny           bool
	TSAdditionalProperties bool

	// Python-specific flags
	PySnakeCaseField       bool
	PyAdditionalProperties bool

	// Go-specific flags
	GoPackageName string
	GoUsePointers bool
	GoOmitEmpty   bool
	GoModulePath  string
}

// ToGenerationConfig creates a generation.Config from flags
func ToGenerationConfig(f *GenerationFlags, schemas []*schema.Schema, compiler *jsonschema.Compiler, outDir string, lang generation.Language) *generation.Config {
	cfg := &generation.Config{
		Schemas:          schemas,
		Compiler:         compiler,
		OutDir:           outDir,
		Language:         lang,
		ExtractInline:    f.ExtractInline,
		DisableHeaders:   f.DisableHeaders,
		DisableTimestamp: f.DisableTimestamp,
		OutputStrategy:   normalizeOutputStrategy(f.OutputStrategy),
	}

	switch lang {
	case generation.LanguageTypeScript:
		cfg.TypeScript = &generation.TypeScriptConfig{
			UnknownAny:           f.TSUnknownAny,
			AdditionalProperties: f.TSAdditionalProperties,
		}
	case generation.LanguagePython:
		cfg.Python = &generation.PythonConfig{
			SnakeCaseField:       f.PySnakeCaseField,
			AdditionalProperties: f.PyAdditionalProperties,
		}
	case generation.LanguageGo:
		// Use flag values or defaults
		packageName := f.GoPackageName
		if packageName == "" {
			packageName = "models"
		}
		cfg.Go = &generation.GoConfig{
			PackageName: packageName,
			UsePointers: f.GoUsePointers,
			OmitEmpty:   f.GoOmitEmpty,
			ModulePath:  f.GoModulePath,
		}
	}

	return cfg
}

// normalizeOutputStrategy converts CLI string input to output.OutputStrategy constant
func normalizeOutputStrategy(strategy string) output.OutputStrategy {
	switch strategy {
	case "bundle":
		return output.StrategyBundle
	case "multifile", "multi-file":
		return output.StrategyMultiFile
	case "bundledeps", "bundle-deps":
		return output.StrategyBundleDeps
	case "bundle-per-dir":
		return output.StrategyBundlePerDir
	default:
		// Default to bundle if unrecognized
		return output.StrategyBundle
	}
}
