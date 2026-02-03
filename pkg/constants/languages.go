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
		return ".ts"
	case string(LanguagePython):
		return ".py"
	case string(LanguageGo):
		return ".go"
	default:
		return ".txt"
	}
}

// IsPython checks if the language is Python.
func IsPython(lang string) bool {
	return NormalizeLanguage(lang) == string(LanguagePython)
}

// IsTypeScript checks if the language is TypeScript.
func IsTypeScript(lang string) bool {
	return NormalizeLanguage(lang) == string(LanguageTypeScript)
}

// IsGo checks if the language is Go.
func IsGo(lang string) bool {
	return NormalizeLanguage(lang) == string(LanguageGo)
}

// GetBarrelFileName returns the barrel/index file name for a language.
// Returns empty string for languages that don't support barrel files.
func GetBarrelFileName(lang string) string {
	switch NormalizeLanguage(lang) {
	case string(LanguageTypeScript):
		return "index.ts"
	case string(LanguagePython):
		return "__init__.py"
	default:
		return ""
	}
}
