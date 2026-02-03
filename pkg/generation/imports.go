package generation

import (
	"path/filepath"
	"strings"

	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

// ImportConverter transforms output.ImportSpec to typegraph.ImportSpec
type ImportConverter interface {
	Convert(imports []output.ImportSpec) []typegraph.ImportSpec
}

// PassthroughConverter copies imports without transformation (used by TypeScript)
type PassthroughConverter struct{}

func (c *PassthroughConverter) Convert(imports []output.ImportSpec) []typegraph.ImportSpec {
	result := make([]typegraph.ImportSpec, len(imports))
	for i, imp := range imports {
		result[i] = typegraph.ImportSpec{
			ImportPath: imp.ImportPath,
			TypeNames:  imp.TypeNames,
		}
	}
	return result
}

// PythonImportConverter converts paths to Python module notation
type PythonImportConverter struct{}

func (c *PythonImportConverter) Convert(imports []output.ImportSpec) []typegraph.ImportSpec {
	result := make([]typegraph.ImportSpec, len(imports))

	for i, imp := range imports {
		path := imp.ImportPath

		if strings.HasPrefix(path, ".") {
			dots := 0

			for strings.HasPrefix(path, "../") {
				dots++
				path = strings.TrimPrefix(path, "../")
			}

			path = strings.TrimPrefix(path, "./")

			path = strings.ReplaceAll(path, "/", ".")
			path = strings.Repeat(".", dots+1) + path
		}

		result[i] = typegraph.ImportSpec{
			ImportPath: path,
			TypeNames:  imp.TypeNames,
		}
	}

	return result
}

// GoImportConverter converts relative paths to absolute Go imports
type GoImportConverter struct {
	ModulePath string
}

func (c *GoImportConverter) Convert(imports []output.ImportSpec) []typegraph.ImportSpec {
	result := make([]typegraph.ImportSpec, len(imports))

	for i, imp := range imports {
		importPath := imp.ImportPath

		if strings.HasPrefix(importPath, ".") {
			importPath = c.resolveRelative(importPath, imp.FromPath)
		}

		result[i] = typegraph.ImportSpec{
			ImportPath: importPath,
			TypeNames:  imp.TypeNames,
		}
	}

	return result
}

func (c *GoImportConverter) resolveRelative(relPath, fromFile string) string {
	if !strings.HasPrefix(relPath, ".") {
		return relPath
	}

	fromDir := filepath.Dir(fromFile)
	targetPath := filepath.Clean(filepath.Join(fromDir, relPath))
	targetPath = filepath.ToSlash(targetPath)

	if c.ModulePath != "" {
		return filepath.ToSlash(filepath.Join(c.ModulePath, targetPath))
	}

	return targetPath
}
