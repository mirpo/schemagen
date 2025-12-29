package generation

import (
	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/constants"
	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/schema"
)

// Config holds all parameters for code generation
type Config struct {
	// Required fields
	Schemas  []*schema.Schema
	Compiler *jsonschema.Compiler
	OutDir   string
	Language Language // "typescript", "python", "go"

	// Universal flags
	ExtractInline    bool
	DisableHeaders   bool
	DisableTimestamp bool
	OutputStrategy   output.OutputStrategy // bundle (default), multi-file, bundle-deps

	// Language-specific config
	TypeScript *TypeScriptConfig
	Python     *PythonConfig
	Go         *GoConfig
}

// Language represents a target programming language
// Use constants from pkg/constants for language values
type Language = constants.Language

const (
	LanguageTypeScript = constants.LanguageTypeScript
	LanguagePython     = constants.LanguagePython
	LanguageGo         = constants.LanguageGo
)

// TypeScriptConfig holds TypeScript-specific generation options
type TypeScriptConfig struct {
	UnknownAny           bool
	AdditionalProperties bool
}

// PythonConfig holds Python-specific generation options
type PythonConfig struct {
	SnakeCaseField       bool
	AdditionalProperties bool
}

// GoConfig holds Go-specific generation options
type GoConfig struct {
	PackageName   string
	UsePointers   bool
	OmitEmpty     bool
	PackagePrefix string
	ModulePath    string // Module path for absolute imports (e.g., "github.com/org/project")
}
