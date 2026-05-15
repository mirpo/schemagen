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
	extTS   = ".ts"
	extPy   = ".py"
	extGo   = ".go"
	ExtJSON = ".json"
	extTxt  = ".txt"
)

// Default file names
const (
	defaultBarrelTS     = "index.ts"
	defaultBarrelPython = "__init__.py"
)

// JSON Schema reference constants
const (
	SchemaSelfRef    = "#"
	SchemaDefsPrefix = "#/$defs/"
)

// normalizeLanguage converts short-form language names to their full form.
// If the input is already a full form, it returns it unchanged.
func normalizeLanguage(lang string) string {
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
	switch normalizeLanguage(lang) {
	case string(LanguageTypeScript):
		return extTS
	case string(LanguagePython):
		return extPy
	case string(LanguageGo):
		return extGo
	default:
		return extTxt
	}
}

// IsPython checks if the language is Python.
func IsPython(lang string) bool {
	return normalizeLanguage(lang) == string(LanguagePython)
}

// GetBarrelFileName returns the barrel/index file name for a language.
// Returns empty string for languages that don't support barrel files.
func GetBarrelFileName(lang string) string {
	switch normalizeLanguage(lang) {
	case string(LanguageTypeScript):
		return defaultBarrelTS
	case string(LanguagePython):
		return defaultBarrelPython
	default:
		return ""
	}
}
