package generation

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mirpo/schemagen/pkg/lang/golang"
	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

// goGenerator wraps the Go generator from pkg/lang/golang
type goGenerator struct {
	generator  *golang.Generator
	modulePath string // Module path for absolute imports
}

// newGoGenerator creates a new Go generator with the given config
func newGoGenerator(graph *typegraph.Graph, cfg *Config) Generator {
	return &goGenerator{
		generator: golang.NewGenerator(graph, &golang.Config{
			PackageName:      cfg.Go.PackageName,
			UsePointers:      cfg.Go.UsePointers,
			OmitEmpty:        cfg.Go.OmitEmpty,
			DisableComments:  false, // Always generate comments
			PackagePrefix:    cfg.Go.PackagePrefix,
			DisableHeaders:   cfg.DisableHeaders,
			DisableTimestamp: cfg.DisableTimestamp,
		}),
		modulePath: cfg.Go.ModulePath, // Store for import conversion
	}
}

// Generate generates Go code for the given types and imports
func (g *goGenerator) Generate(types []*typegraph.Type, imports interface{}) (string, error) {
	goImports, ok := imports.([]typegraph.ImportSpec)
	if !ok {
		return "", fmt.Errorf("invalid imports type: expected []typegraph.ImportSpec, got %T", imports)
	}
	return g.generator.GenerateFile(types, goImports)
}

// ConvertImports converts generic imports to Go-specific format
func (g *goGenerator) ConvertImports(imports []output.ImportSpec) interface{} {
	result := make([]typegraph.ImportSpec, len(imports))
	for i, imp := range imports {
		importPath := imp.ImportPath

		// Transform relative imports to absolute module paths
		if g.modulePath != "" && strings.HasPrefix(importPath, ".") {
			importPath = relativeToAbsoluteImport(importPath, imp.FromPath, g.modulePath)
		}

		result[i] = typegraph.ImportSpec{
			ImportPath: importPath,
			TypeNames:  imp.TypeNames,
		}
	}
	return result
}

// relativeToAbsoluteImport converts a relative import path to an absolute module path.
// Examples:
//
//	"./header" + "events/event.go" + "github.com/test/project" → "github.com/test/project/events/header"
//	"./payloads/ping" + "events/event.go" + "github.com/test/project" → "github.com/test/project/events/payloads/ping"
//	"../common/types" + "events/event.go" + "github.com/test/project" → "github.com/test/project/common/types"
func relativeToAbsoluteImport(relPath, fromFile, modulePath string) string {
	// If already absolute (doesn't start with . or ..), return as-is
	if !strings.HasPrefix(relPath, ".") {
		return relPath
	}

	// Remove leading "./" if present
	relPath = strings.TrimPrefix(relPath, "./")

	// Get directory of source file
	fromDir := filepath.Dir(fromFile)

	// Resolve relative path from source directory
	targetPath := filepath.Join(fromDir, relPath)

	// Clean path (resolve .. and .)
	targetPath = filepath.Clean(targetPath)

	// Convert to forward slashes (Go import paths always use /)
	targetPath = filepath.ToSlash(targetPath)

	// Prepend module path
	if modulePath != "" {
		return filepath.ToSlash(filepath.Join(modulePath, targetPath))
	}

	return targetPath
}
