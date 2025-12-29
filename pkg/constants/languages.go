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
