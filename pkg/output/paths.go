package output

import (
	"path/filepath"
	"strings"
)

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

// stripExtension removes the file extension from a path or filename.
func stripExtension(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}
