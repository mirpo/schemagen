package py

import (
	"strings"

	"github.com/mirpo/schemagen/pkg/graph"
)

type ImportConverter struct{}

func (c *ImportConverter) Convert(imports []graph.ImportSpec) []graph.ImportSpec {
	result := make([]graph.ImportSpec, len(imports))
	for i, imp := range imports {
		result[i] = imp
		result[i].ImportPath = toPythonImport(imp.ImportPath)
	}
	return result
}

func toPythonImport(path string) string {
	if !strings.HasPrefix(path, ".") {
		return path
	}
	dots := 1
	for strings.HasPrefix(path, "../") {
		dots++
		path = path[3:]
	}
	path = strings.TrimPrefix(path, "./")
	return strings.Repeat(".", dots) + strings.ReplaceAll(path, "/", ".")
}
