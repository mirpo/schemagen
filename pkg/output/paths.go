package output

import (
	"path/filepath"
	"strings"

	"github.com/mirpo/schemagen/pkg/constants"
)

type PathMapper struct {
	inputRoot  string
	outputRoot string
	language   string
}

func NewPathMapper(inputRoot, outputRoot, language string) *PathMapper {
	return &PathMapper{
		inputRoot:  inputRoot,
		outputRoot: outputRoot,
		language:   language,
	}
}

func (pm *PathMapper) InputPathToOutputPath(schemaPath string) string {
	ext := getLanguageExtension(pm.language)

	relPath, err := filepath.Rel(pm.inputRoot, schemaPath)
	if err != nil {
		relPath = filepath.Base(schemaPath)
	}

	base := filepath.Base(relPath)
	nameWithoutExt := base[:len(base)-len(filepath.Ext(base))]

	// Sanitize filename for Python: convert hyphens to underscores
	if pm.language == constants.LanguagePythonShort || pm.language == string(constants.LanguagePython) {
		nameWithoutExt = strings.ReplaceAll(nameWithoutExt, "-", "_")
	}

	dir := filepath.Dir(relPath)
	if dir == "." {
		return nameWithoutExt + ext
	}

	return filepath.Join(dir, nameWithoutExt+ext)
}

func (pm *PathMapper) ComputeImportPath(fromFile, toFile string) string {
	fromDir := filepath.Dir(fromFile)
	toDir := filepath.Dir(toFile)
	toBase := filepath.Base(toFile)
	toName := toBase[:len(toBase)-len(filepath.Ext(toBase))]

	if fromDir == toDir {
		return "./" + toName
	}

	relPath, err := filepath.Rel(fromDir, toDir)
	if err != nil {
		return toName
	}

	if relPath == "." {
		return "./" + toName
	}

	relPath = filepath.ToSlash(relPath)

	return relPath + "/" + toName
}

func (pm *PathMapper) BarrelFilePath(dir string) string {
	switch pm.language {
	case constants.LanguageTypeScriptShort, string(constants.LanguageTypeScript):
		if dir == "" || dir == "." {
			return "index.ts"
		}
		return filepath.Join(dir, "index.ts")
	case constants.LanguagePythonShort, string(constants.LanguagePython):
		if dir == "" || dir == "." {
			return "__init__.py"
		}
		return filepath.Join(dir, "__init__.py")
	default:
		return ""
	}
}

func ComputeRelativeImport(fromFile, toFile string) string {
	fromDir := filepath.Dir(fromFile)

	if filepath.Dir(toFile) == fromDir {
		toBase := filepath.Base(toFile)
		toName := toBase[:len(toBase)-len(filepath.Ext(toBase))]
		return "./" + toName
	}

	relPath, err := filepath.Rel(fromDir, toFile)
	if err != nil {
		return toFile
	}

	relPath = filepath.ToSlash(relPath)

	ext := filepath.Ext(relPath)
	nameWithoutExt := relPath[:len(relPath)-len(ext)]

	if !strings.HasPrefix(nameWithoutExt, ".") && !strings.HasPrefix(nameWithoutExt, "/") {
		nameWithoutExt = "./" + nameWithoutExt
	}

	return nameWithoutExt
}

func GetDirectoryLevels(path string) []string {
	var levels []string

	if path == "" || path == "." {
		return levels
	}

	dir := filepath.Dir(path)
	for dir != "" && dir != "." {
		levels = append([]string{dir}, levels...)
		dir = filepath.Dir(dir)
	}

	return levels
}
