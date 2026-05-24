package output

import (
	"path/filepath"
	"strings"
)

func computeRelativeImport(fromFile, toFile string) string {
	fromFile = filepath.ToSlash(fromFile)
	toFile = filepath.ToSlash(toFile)

	fromDir := filepath.ToSlash(filepath.Dir(fromFile))

	if filepath.ToSlash(filepath.Dir(toFile)) == fromDir {
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

func stripExtension(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}
