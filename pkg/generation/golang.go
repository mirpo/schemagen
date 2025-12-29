package generation

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mirpo/schemagen/pkg/lang/golang"
	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

type goGenerator struct {
	generator  *golang.Generator
	modulePath string
}

func newGoGenerator(graph *typegraph.Graph, cfg *Config) Generator {
	return &goGenerator{
		generator: golang.NewGenerator(graph, &golang.Config{
			PackageName:      cfg.Go.PackageName,
			UsePointers:      cfg.Go.UsePointers,
			OmitEmpty:        cfg.Go.OmitEmpty,
			DisableComments:  false,
			PackagePrefix:    cfg.Go.PackagePrefix,
			DisableHeaders:   cfg.DisableHeaders,
			DisableTimestamp: cfg.DisableTimestamp,
		}),
		modulePath: cfg.Go.ModulePath,
	}
}

func (g *goGenerator) Generate(types []*typegraph.Type, imports interface{}) (string, error) {
	goImports, ok := imports.([]typegraph.ImportSpec)
	if !ok {
		return "", fmt.Errorf(
			"invalid imports type: expected []typegraph.ImportSpec, got %T",
			imports,
		)
	}
	return g.generator.GenerateFile(types, goImports)
}

func (g *goGenerator) ConvertImports(imports []output.ImportSpec) interface{} {
	result := make([]typegraph.ImportSpec, len(imports))

	for i, imp := range imports {
		importPath := imp.ImportPath

		if strings.HasPrefix(importPath, ".") {
			importPath = relativeToAbsoluteImport(
				importPath,
				imp.FromPath,
				g.modulePath,
			)
		}

		result[i] = typegraph.ImportSpec{
			ImportPath: importPath,
			TypeNames:  imp.TypeNames,
		}
	}

	return result
}

func relativeToAbsoluteImport(relPath, fromFile, modulePath string) string {
	if !strings.HasPrefix(relPath, ".") {
		return relPath
	}

	fromDir := filepath.Dir(fromFile)
	targetPath := filepath.Clean(filepath.Join(fromDir, relPath))
	targetPath = filepath.ToSlash(targetPath)

	if modulePath != "" {
		return filepath.ToSlash(filepath.Join(modulePath, targetPath))
	}

	return targetPath
}
