package pipeline

import (
	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/parse"
)

type Config struct {
	Schemas  []*parse.NamedSchema
	OutDir   string
	Language Language

	ExtractInline    bool
	DisableHeaders   bool
	DisableTimestamp bool
	OutputStrategy   output.OutputStrategy

	TypeScript *TypeScriptConfig
	Python     *PythonConfig
	Go         *GoConfig

	Writer FileWriter
}

type Language = output.Language

const (
	LanguageTypeScript = output.LanguageTypeScript
	LanguagePython     = output.LanguagePython
	LanguageGo         = output.LanguageGo

	DefaultBundleName = "types"
)

type TypeScriptConfig struct {
	UnknownAny           bool
	AdditionalProperties bool
	Zod                  bool
	ZodOnly              bool
	ZodCoerceDates       bool
	ZodStrict            bool
}

type PythonConfig struct {
	SnakeCaseField       bool
	AdditionalProperties bool
}

type GoConfig struct {
	PackageName string
	UsePointers bool
	OmitEmpty   bool
	ModulePath  string
}

type GenerationFlags struct {
	OutTS string
	OutPY string
	OutGo string

	ExtractInline    bool
	DisableHeaders   bool
	DisableTimestamp bool
	OutputStrategy   output.OutputStrategy

	TSUnknownAny           bool
	TSAdditionalProperties bool
	TSZod                  bool
	TSZodOnly              bool
	TSZodCoerceDates       bool
	TSZodStrict            bool

	PySnakeCaseField       bool
	PyAdditionalProperties bool

	GoPackageName string
	GoUsePointers bool
	GoOmitEmpty   bool
	GoModulePath  string
}

func ConfigFromFlags(f *GenerationFlags, schemas []*parse.NamedSchema, outDir string, lang Language) *Config {
	cfg := &Config{
		Schemas:          schemas,
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

type GenerationTarget struct {
	Dir  string
	Lang Language
}

func BuildTargets(flags *GenerationFlags) []GenerationTarget {
	all := []GenerationTarget{
		{flags.OutTS, LanguageTypeScript},
		{flags.OutPY, LanguagePython},
		{flags.OutGo, LanguageGo},
	}

	targets := make([]GenerationTarget, 0, len(all))
	for _, t := range all {
		if t.Dir != "" {
			targets = append(targets, t)
		}
	}
	return targets
}
