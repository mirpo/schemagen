package output

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mirpo/schemagen/pkg/constants"
	"github.com/mirpo/schemagen/pkg/naming"
	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/mirpo/schemagen/pkg/typegraph"
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
	Types        []*typegraph.Type
	Imports      []typegraph.ImportSpec
}

type BarrelFile struct {
	Path    string
	Exports []string
}

func PlanOutput(graph *typegraph.Graph, schemas []*schema.Schema, strategy OutputStrategy, language string, bundleName string) (*OutputPlan, error) {
	typeSourceMap := buildTypeSourceMap(schemas)

	switch strategy {
	case StrategyBundle:
		return planBundle(graph, language, bundleName), nil
	case StrategyMultiFile:
		return planMultiFile(graph, schemas, typeSourceMap, language), nil
	case StrategyBundleDeps:
		return planBundleDeps(graph, schemas, typeSourceMap, language, bundleName), nil
	default:
		return nil, fmt.Errorf("unsupported output strategy: %s", strategy)
	}
}

func buildTypeSourceMap(schemas []*schema.Schema) map[string]*schema.Schema {
	typeSourceMap := make(map[string]*schema.Schema)

	for _, s := range schemas {
		schemaTypes := extractTypesFromSchema(s)
		for _, typeName := range schemaTypes {
			typeSourceMap[typeName] = s
		}
	}

	return typeSourceMap
}

func extractTypesFromSchema(s *schema.Schema) []string {
	var types []string

	if s == nil {
		return types
	}

	if s.Name != "" {
		types = append(types, s.Name)
	}

	if s.Compiled != nil && s.Compiled.Defs != nil {
		for defName := range s.Compiled.Defs {
			types = append(types, naming.ToPascalCase(defName))
		}
	}

	return types
}

func planBundle(graph *typegraph.Graph, language string, bundleName string) *OutputPlan {
	ext := constants.GetExtension(language)
	filename := fmt.Sprintf("%s%s", bundleName, ext)

	return &OutputPlan{
		Strategy: StrategyBundle,
		Files: []OutputFile{
			{
				RelativePath: filename,
				Types:        graph.Types,
				Imports:      []typegraph.ImportSpec{},
			},
		},
	}
}

func planMultiFile(graph *typegraph.Graph, schemas []*schema.Schema, typeSourceMap map[string]*schema.Schema, language string) *OutputPlan {
	schemaFiles := make(map[string]*OutputFile)

	// For each schema, determine which types belong to it (including orphaned dependencies)
	// We do this in two passes to preserve graph.Types order (dependencies before parent)
	for _, s := range schemas {
		outputPath := convertSchemaPathToOutputForLanguage(s.RelativePath, language)

		// First pass: collect all type names that belong to this file
		belongsToFile := make(map[string]bool)
		for _, typ := range graph.Types {
			if typeSourceMap[typ.Name] == s {
				belongsToFile[typ.Name] = true
				// Mark orphaned types (no source schema) that are referenced by this type
				markOrphanedTypes(typ, graph, typeSourceMap, belongsToFile)
			}
		}

		// Second pass: iterate graph.Types in order and collect types that belong to this file
		// This preserves the type graph order (dependencies come before types that use them)
		var fileTypes []*typegraph.Type
		for _, typ := range graph.Types {
			if belongsToFile[typ.Name] {
				fileTypes = append(fileTypes, typ)
			}
		}

		if len(fileTypes) > 0 {
			schemaFiles[outputPath] = &OutputFile{
				RelativePath: outputPath,
				Types:        fileTypes,
				Imports:      []typegraph.ImportSpec{},
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

// markOrphanedTypes recursively marks orphaned types (types without a source schema)
// that are referenced by the given type.
func markOrphanedTypes(typ *typegraph.Type, graph *typegraph.Graph, typeSourceMap map[string]*schema.Schema, belongsToFile map[string]bool) {
	markOrphanedRef := func(r *typegraph.TypeRef) {
		if r.TypeName != "" && !belongsToFile[r.TypeName] && typeSourceMap[r.TypeName] == nil {
			if refType := graph.GetType(r.TypeName); refType != nil {
				belongsToFile[r.TypeName] = true
				markOrphanedTypes(refType, graph, typeSourceMap, belongsToFile)
			}
		}
	}

	for _, field := range typ.Fields {
		field.Type.Walk(markOrphanedRef)
	}

	for _, baseName := range typ.Extends {
		if !belongsToFile[baseName] && typeSourceMap[baseName] == nil {
			if baseType := graph.GetType(baseName); baseType != nil {
				belongsToFile[baseName] = true
				markOrphanedTypes(baseType, graph, typeSourceMap, belongsToFile)
			}
		}
	}
}

func planBundleDeps(graph *typegraph.Graph, schemas []*schema.Schema, typeSourceMap map[string]*schema.Schema, language string, bundleName string) *OutputPlan {
	ext := constants.GetExtension(language)
	filename := fmt.Sprintf("%s%s", bundleName, ext)

	rootSchema := schemas[0]
	includedTypes := collectDependentTypes(graph, rootSchema, typeSourceMap)

	return &OutputPlan{
		Strategy: StrategyBundleDeps,
		Files: []OutputFile{
			{
				RelativePath: filename,
				Types:        includedTypes,
				Imports:      []typegraph.ImportSpec{},
			},
		},
	}
}

func collectDependentTypes(graph *typegraph.Graph, rootSchema *schema.Schema, typeSourceMap map[string]*schema.Schema) []*typegraph.Type {
	included := make(map[string]bool)
	var result []*typegraph.Type

	for _, typ := range graph.Types {
		if typeSourceMap[typ.Name] == rootSchema {
			included[typ.Name] = true
			result = append(result, typ)
			collectReferencedTypes(typ, graph, included, &result)
		}
	}

	return result
}

func collectReferencedTypes(typ *typegraph.Type, graph *typegraph.Graph, included map[string]bool, result *[]*typegraph.Type) {
	for _, field := range typ.Fields {
		collectFieldReferences(field.Type, graph, included, result)
	}

	for _, baseName := range typ.Extends {
		if !included[baseName] {
			if baseType := graph.GetType(baseName); baseType != nil {
				included[baseName] = true
				*result = append(*result, baseType)
				collectReferencedTypes(baseType, graph, included, result)
			}
		}
	}
}

// collectFieldReferences recursively collects all types referenced by a TypeRef
func collectFieldReferences(ref *typegraph.TypeRef, graph *typegraph.Graph, included map[string]bool, result *[]*typegraph.Type) {
	if ref == nil {
		return
	}

	// Collect direct type reference
	if ref.TypeName != "" && !included[ref.TypeName] {
		if refType := graph.GetType(ref.TypeName); refType != nil {
			included[ref.TypeName] = true
			*result = append(*result, refType)
			collectReferencedTypes(refType, graph, included, result)
		}
	}

	// Collect union members
	for _, member := range ref.UnionMembers {
		collectFieldReferences(member, graph, included, result)
	}

	// Collect array item types
	if ref.ItemType != nil {
		collectFieldReferences(ref.ItemType, graph, included, result)
	}

	// Collect map value types
	if ref.ValueType != nil {
		collectFieldReferences(ref.ValueType, graph, included, result)
	}
}

func convertSchemaPathToOutputForLanguage(schemaPath string, language string) string {
	ext := constants.GetExtension(language)
	dir := filepath.Dir(schemaPath)
	base := filepath.Base(schemaPath)
	nameWithoutExt := stripExtension(base)

	// Sanitize filename for Python: convert hyphens to underscores
	if constants.IsPython(language) {
		nameWithoutExt = strings.ReplaceAll(nameWithoutExt, "-", "_")
	}

	outputName := nameWithoutExt + ext

	// If there's a directory structure (not just "."), preserve it
	if dir != "." && dir != "" {
		return filepath.Join(dir, outputName)
	}
	return outputName
}
