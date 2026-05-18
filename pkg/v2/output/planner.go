package output

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mirpo/schemagen/pkg/constants"
	"github.com/mirpo/schemagen/pkg/naming"
	"github.com/mirpo/schemagen/pkg/v2/graph"
	"github.com/mirpo/schemagen/pkg/v2/parse"
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
	Strategy OutputStrategy
	Files    []OutputFile
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

func PlanOutput(g *graph.Graph, schemas []parse.NamedSchema, strategy OutputStrategy, language constants.Language, bundleName string) (*OutputPlan, error) {
	typeSourceMap := buildTypeSourceMap(schemas)

	switch strategy {
	case StrategyBundle:
		return planBundle(g, language, bundleName), nil
	case StrategyMultiFile:
		return planMultiFile(g, schemas, typeSourceMap, language), nil
	case StrategyBundleDeps:
		return planBundleDeps(g, schemas, typeSourceMap, language, bundleName), nil
	default:
		return nil, fmt.Errorf("unsupported output strategy: %s", strategy)
	}
}

func buildTypeSourceMap(schemas []parse.NamedSchema) map[string]*parse.NamedSchema {
	typeSourceMap := make(map[string]*parse.NamedSchema)

	for i := range schemas {
		s := &schemas[i]
		schemaTypes := extractTypesFromSchema(s)
		for _, typeName := range schemaTypes {
			typeSourceMap[typeName] = s
		}
	}

	return typeSourceMap
}

func extractTypesFromSchema(s *parse.NamedSchema) []string {
	var types []string

	if s.Name != "" {
		types = append(types, s.Name)
	}

	if s.Schema != nil && len(s.Schema.Defs) > 0 {
		for _, def := range s.Schema.Defs {
			types = append(types, naming.ToPascalCase(def.Name))
		}
	}

	return types
}

func planBundle(g *graph.Graph, language constants.Language, bundleName string) *OutputPlan {
	ext := constants.GetExtension(string(language))
	filename := fmt.Sprintf("%s%s", bundleName, ext)

	return &OutputPlan{
		Strategy: StrategyBundle,
		Files: []OutputFile{
			{
				RelativePath: filename,
				Types:        g.Types,
				Imports:      []graph.ImportSpec{},
			},
		},
	}
}

func planMultiFile(g *graph.Graph, schemas []parse.NamedSchema, typeSourceMap map[string]*parse.NamedSchema, language constants.Language) *OutputPlan {
	schemaFiles := make(map[string]*OutputFile)

	for i := range schemas {
		s := &schemas[i]
		outputPath := convertSchemaPathToOutputForLanguage(s.Path, language)

		belongsToFile := make(map[string]bool)
		for _, typ := range g.Types {
			if typeSourceMap[typ.Name] == s {
				belongsToFile[typ.Name] = true
				markOrphanedTypes(typ, g, typeSourceMap, belongsToFile)
			}
		}

		var fileTypes []*graph.Type
		for _, typ := range g.Types {
			if belongsToFile[typ.Name] {
				fileTypes = append(fileTypes, typ)
			}
		}

		if len(fileTypes) > 0 {
			schemaFiles[outputPath] = &OutputFile{
				RelativePath: outputPath,
				Types:        fileTypes,
				Imports:      []graph.ImportSpec{},
			}
		}
	}

	plan := &OutputPlan{
		Strategy: StrategyMultiFile,
		Files:    make([]OutputFile, 0, len(schemaFiles)),
	}

	for _, p := range slices.Sorted(maps.Keys(schemaFiles)) {
		plan.Files = append(plan.Files, *schemaFiles[p])
	}

	return plan
}

func markOrphanedTypes(typ *graph.Type, g *graph.Graph, typeSourceMap map[string]*parse.NamedSchema, belongsToFile map[string]bool) {
	walkReachableTypes(typ, g, func(name string) bool {
		return !belongsToFile[name] && typeSourceMap[name] == nil
	}, func(name string) {
		belongsToFile[name] = true
	})
}

func planBundleDeps(g *graph.Graph, schemas []parse.NamedSchema, typeSourceMap map[string]*parse.NamedSchema, language constants.Language, bundleName string) *OutputPlan {
	if len(schemas) == 0 {
		return &OutputPlan{}
	}

	ext := constants.GetExtension(string(language))
	filename := fmt.Sprintf("%s%s", bundleName, ext)

	rootSchema := &schemas[0]
	includedTypes := collectDependentTypes(g, rootSchema, typeSourceMap)

	return &OutputPlan{
		Strategy: StrategyBundleDeps,
		Files: []OutputFile{
			{
				RelativePath: filename,
				Types:        includedTypes,
				Imports:      []graph.ImportSpec{},
			},
		},
	}
}

func collectDependentTypes(g *graph.Graph, rootSchema *parse.NamedSchema, typeSourceMap map[string]*parse.NamedSchema) []*graph.Type {
	included := make(map[string]bool)
	var result []*graph.Type

	for _, typ := range g.Types {
		if typeSourceMap[typ.Name] == rootSchema {
			included[typ.Name] = true
			result = append(result, typ)
			collectReferencedTypes(typ, g, included, &result)
		}
	}

	return result
}

func collectReferencedTypes(typ *graph.Type, g *graph.Graph, included map[string]bool, result *[]*graph.Type) {
	walkReachableTypes(typ, g, func(name string) bool {
		return !included[name]
	}, func(name string) {
		included[name] = true
		*result = append(*result, g.GetType(name))
	})
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

func convertSchemaPathToOutputForLanguage(schemaPath string, language constants.Language) string {
	ext := constants.GetExtension(string(language))
	dir := filepath.Dir(schemaPath)
	base := filepath.Base(schemaPath)
	nameWithoutExt := stripExtension(base)

	if constants.IsPython(string(language)) {
		nameWithoutExt = strings.ReplaceAll(nameWithoutExt, "-", "_")
	}

	outputName := nameWithoutExt + ext

	if dir != "." && dir != "" {
		return filepath.Join(dir, outputName)
	}
	return outputName
}
