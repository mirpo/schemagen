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
	ext := constants.GetExtension(pm.language)

	relPath, err := filepath.Rel(pm.inputRoot, schemaPath)
	if err != nil {
		relPath = filepath.Base(schemaPath)
	}

	base := filepath.Base(relPath)
	name := stripExtension(base)

	if constants.IsPython(pm.language) {
		name = strings.ReplaceAll(name, "-", "_")
	}

	dir := filepath.Dir(relPath)
	if dir == "." {
		return name + ext
	}

	return filepath.Join(dir, name+ext)
}

func (pm *PathMapper) ComputeImportPath(fromFile, toFile string) string {
	return ComputeRelativeImport(fromFile, toFile)
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
	}

	return ""
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

	if !strings.HasPrefix(nameWithoutExt, ".") {
		nameWithoutExt = "./" + nameWithoutExt
	}

	return nameWithoutExt
}

func GetDirectoryLevels(path string) []string {
	if path == "" || path == "." {
		return nil
	}

	var levels []string
	dir := filepath.Dir(path)

	for dir != "" && dir != "." {
		levels = append([]string{dir}, levels...)
		dir = filepath.Dir(dir)
	}

	return levels
}

// stripExtension removes the file extension from a path or filename.
func stripExtension(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}
