package output

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mirpo/schemagen/pkg/graph"
)

type OutputStrategy string

const (
	StrategyBundle     OutputStrategy = "bundle"
	StrategyBundleDeps OutputStrategy = "bundle-deps"
	StrategyMultiFile  OutputStrategy = "multi-file"
)

func (s *OutputStrategy) Set(v string) error {
	switch v {
	case "bundle":
		*s = StrategyBundle
	case "multifile", "multi-file":
		*s = StrategyMultiFile
	case "bundledeps", "bundle-deps":
		*s = StrategyBundleDeps
	default:
		return fmt.Errorf("invalid output strategy %q (must be bundle, multifile, or bundledeps)", v)
	}
	return nil
}

func ParseStrategy(v string) OutputStrategy {
	var s OutputStrategy
	if err := s.Set(v); err != nil {
		return StrategyMultiFile
	}
	return s
}

type OutputPlan struct {
	Files []OutputFile
}

type OutputFile struct {
	RelativePath string
	Types        []*graph.Type
	Imports      []graph.ImportSpec
}

type BarrelFile struct {
	Path    string
	Exports []string
}

func PlanOutput(g *graph.Graph, strategy OutputStrategy, language Language, bundleName string, rootSourceFile string) (*OutputPlan, error) {
	switch strategy {
	case StrategyBundle:
		return planBundle(g, language, bundleName), nil
	case StrategyMultiFile:
		return planMultiFile(g, language), nil
	case StrategyBundleDeps:
		return planBundleDeps(g, language, bundleName, rootSourceFile), nil
	default:
		return nil, fmt.Errorf("unsupported output strategy: %s", strategy)
	}
}

func planBundle(g *graph.Graph, language Language, bundleName string) *OutputPlan {
	ext := GetExtension(language)
	filename := fmt.Sprintf("%s%s", bundleName, ext)

	return &OutputPlan{
		Files: []OutputFile{
			{
				RelativePath: filename,
				Types:        g.Types,
				Imports:      []graph.ImportSpec{},
			},
		},
	}
}

func planMultiFile(g *graph.Graph, language Language) *OutputPlan {
	fileTypes := make(map[string][]*graph.Type)

	for _, typ := range g.Types {
		if typ.SourceFile == "" {
			continue
		}
		outputPath := convertSchemaPathToOutputForLanguage(typ.SourceFile, language)
		fileTypes[outputPath] = append(fileTypes[outputPath], typ)
	}

	assignOrphanedTypes(g, fileTypes)

	plan := &OutputPlan{
		Files: make([]OutputFile, 0, len(fileTypes)),
	}

	for _, p := range slices.Sorted(maps.Keys(fileTypes)) {
		plan.Files = append(plan.Files, OutputFile{
			RelativePath: p,
			Types:        fileTypes[p],
			Imports:      []graph.ImportSpec{},
		})
	}

	return plan
}

func assignOrphanedTypes(g *graph.Graph, fileTypes map[string][]*graph.Type) {
	type pending struct {
		path string
		typ  *graph.Type
	}

	assigned := make(map[string]bool)
	var orphans []pending

	for outputPath, types := range fileTypes {
		for _, typ := range types {
			walkReachableTypes(typ, g, func(name string) bool {
				t := g.GetType(name)
				return t != nil && t.SourceFile == "" && !assigned[name]
			}, func(name string) {
				assigned[name] = true
				if t := g.GetType(name); t != nil {
					orphans = append(orphans, pending{outputPath, t})
				}
			})
		}
	}

	for _, o := range orphans {
		fileTypes[o.path] = append(fileTypes[o.path], o.typ)
	}
}

func planBundleDeps(g *graph.Graph, language Language, bundleName string, rootSourceFile string) *OutputPlan {
	if rootSourceFile == "" {
		return &OutputPlan{}
	}

	ext := GetExtension(language)
	filename := fmt.Sprintf("%s%s", bundleName, ext)

	included := make(map[string]bool)
	var result []*graph.Type

	for _, typ := range g.Types {
		if typ.SourceFile == rootSourceFile {
			included[typ.Name] = true
			result = append(result, typ)
		}
	}

	for i := 0; i < len(result); i++ {
		walkReachableTypes(result[i], g, func(name string) bool {
			return !included[name]
		}, func(name string) {
			included[name] = true
			if t := g.GetType(name); t != nil {
				result = append(result, t)
			}
		})
	}

	return &OutputPlan{
		Files: []OutputFile{
			{
				RelativePath: filename,
				Types:        result,
				Imports:      []graph.ImportSpec{},
			},
		},
	}
}

func walkReachableTypes(typ *graph.Type, g *graph.Graph, shouldVisit func(string) bool, onVisit func(string)) {
	visit := func(name string) {
		if shouldVisit(name) {
			if t := g.GetType(name); t != nil {
				onVisit(name)
				walkReachableTypes(t, g, shouldVisit, onVisit)
			}
		}
	}

	for _, field := range typ.Fields {
		field.Type.Walk(func(r *graph.TypeRef) {
			if r.TypeName != "" {
				visit(r.TypeName)
			}
		})
	}

	for _, baseName := range typ.Extends {
		visit(baseName)
	}

	for _, m := range typ.UnionMembers {
		m.Walk(func(r *graph.TypeRef) {
			if r.TypeName != "" {
				visit(r.TypeName)
			}
		})
	}
}

func convertSchemaPathToOutputForLanguage(schemaPath string, language Language) string {
	ext := GetExtension(language)
	dir := filepath.Dir(schemaPath)
	base := filepath.Base(schemaPath)
	nameWithoutExt := stripExtension(base)

	if language == LanguagePython {
		nameWithoutExt = strings.ReplaceAll(nameWithoutExt, "-", "_")
	}

	outputName := nameWithoutExt + ext

	if dir != "." && dir != "" {
		return filepath.Join(dir, outputName)
	}
	return outputName
}
