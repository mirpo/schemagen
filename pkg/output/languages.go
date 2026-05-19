package output

type Language string

const (
	LanguageTypeScript Language = "typescript"
	LanguagePython     Language = "python"
	LanguageGo         Language = "go"
)

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
