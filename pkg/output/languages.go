package output

type Language string

const (
	LanguageTypeScript Language = "typescript"
	LanguagePython     Language = "python"
	LanguageGo         Language = "go"

	LanguageTypeScriptShort = "ts"
	LanguagePythonShort     = "py"
	LanguageGoShort         = "go"
)

const (
	ExtJSON = ".json"
)

func ShortName(lang Language) string {
	switch lang {
	case LanguageTypeScript:
		return LanguageTypeScriptShort
	case LanguagePython:
		return LanguagePythonShort
	case LanguageGo:
		return LanguageGoShort
	default:
		return string(lang)
	}
}

func GetExtension(lang Language) string {
	switch lang {
	case LanguageTypeScript:
		return ".ts"
	case LanguagePython:
		return ".py"
	case LanguageGo:
		return ".go"
	default:
		return ".txt"
	}
}

func GetBarrelFileName(lang Language) string {
	switch lang {
	case LanguageTypeScript:
		return "index.ts"
	case LanguagePython:
		return "__init__.py"
	default:
		return ""
	}
}
