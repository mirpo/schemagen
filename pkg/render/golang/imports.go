package golang

import (
	"path/filepath"
	"strings"

	"github.com/mirpo/schemagen/pkg/graph"
)

type ImportConverter struct {
	ModulePath string
}

func (c *ImportConverter) Convert(imports []graph.ImportSpec) []graph.ImportSpec {
	result := make([]graph.ImportSpec, len(imports))
	for i, imp := range imports {
		result[i] = imp
		if strings.HasPrefix(imp.ImportPath, ".") {
			result[i].ImportPath = c.resolveRelative(imp.ImportPath, imp.FromPath)
		}
	}
	return result
}

func (c *ImportConverter) resolveRelative(relPath, fromFile string) string {
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
