package constants

// Language represents a target programming language
type Language string

const (
	LanguageTypeScript Language = "typescript"
	LanguagePython     Language = "python"
	LanguageGo         Language = "go"

	LanguageTypeScriptShort = "ts"
	LanguagePythonShort     = "py"
	LanguageGoShort         = "go"
)

// File extensions
const (
	ExtTS   = ".ts"
	ExtPy   = ".py"
	ExtGo   = ".go"
	ExtJSON = ".json"
	ExtTxt  = ".txt"
)

// Default file names
const (
	DefaultBarrelTS     = "index.ts"
	DefaultBarrelPython = "__init__.py"
)

// JSON Schema reference constants
const (
	SchemaSelfRef    = "#"
	SchemaDefsPrefix = "#/$defs/"
)

// NormalizeLanguage converts short-form language names to their full form.
// If the input is already a full form, it returns it unchanged.
func NormalizeLanguage(lang string) string {
	switch lang {
	case LanguageTypeScriptShort, string(LanguageTypeScript):
		return string(LanguageTypeScript)
	case LanguagePythonShort, string(LanguagePython):
		return string(LanguagePython)
	case string(LanguageGo):
		return string(LanguageGo)
	default:
		return lang
	}
}

// GetExtension returns the file extension for a language.
func GetExtension(lang string) string {
	switch NormalizeLanguage(lang) {
	case string(LanguageTypeScript):
		return ExtTS
	case string(LanguagePython):
		return ExtPy
	case string(LanguageGo):
		return ExtGo
	default:
		return ExtTxt
	}
}

// IsPython checks if the language is Python.
func IsPython(lang string) bool {
	return NormalizeLanguage(lang) == string(LanguagePython)
}

// GetBarrelFileName returns the barrel/index file name for a language.
// Returns empty string for languages that don't support barrel files.
func GetBarrelFileName(lang string) string {
	switch NormalizeLanguage(lang) {
	case string(LanguageTypeScript):
		return DefaultBarrelTS
	case string(LanguagePython):
		return DefaultBarrelPython
	default:
		return ""
	}
}
