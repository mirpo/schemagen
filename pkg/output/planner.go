package output

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mirpo/schemagen/pkg/constants"
	"github.com/mirpo/schemagen/pkg/naming"
	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

type OutputStrategy string

const (
	StrategyBundle       OutputStrategy = "bundle"
	StrategyBundleDeps   OutputStrategy = "bundle-deps"
	StrategyMultiFile    OutputStrategy = "multi-file"
	StrategyBundlePerDir OutputStrategy = "bundle-per-dir"
)

func (s *OutputStrategy) String() string {
	return string(*s)
}

func (s *OutputStrategy) Set(v string) error {
	switch v {
	case "bundle":
		*s = StrategyBundle
	case "multifile", "multi-file":
		*s = StrategyMultiFile
	case "bundledeps", "bundle-deps":
		*s = StrategyBundleDeps
	case "bundle-per-dir":
		*s = StrategyBundlePerDir
	default:
		return fmt.Errorf("invalid output strategy %q (must be bundle, multifile, bundledeps, or bundle-per-dir)", v)
	}
	return nil
}

func (s *OutputStrategy) Type() string {
	return "strategy"
}

func ParseStrategy(v string) OutputStrategy {
	var s OutputStrategy
	if err := s.Set(v); err != nil {
		return StrategyMultiFile
	}
	return s
}

type OutputPlan struct {
	Strategy    OutputStrategy
	Files       []OutputFile
	BarrelFiles []BarrelFile
}

type OutputFile struct {
	Language     string
	RelativePath string
	Types        []*typegraph.Type
	Imports      []ImportSpec
}

type ImportSpec struct {
	FromPath   string
	ToPath     string
	TypeNames  []string
	ImportPath string
}

type BarrelFile struct {
	Path    string
	Exports []string
}

type TypeSource struct {
	TypeName   string
	SchemaPath string
	SchemaName string
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
	ext := getLanguageExtension(language)
	filename := fmt.Sprintf("%s%s", bundleName, ext)

	return &OutputPlan{
		Strategy: StrategyBundle,
		Files: []OutputFile{
			{
				Language:     language,
				RelativePath: filename,
				Types:        graph.Types,
				Imports:      []ImportSpec{},
			},
		},
		BarrelFiles: []BarrelFile{},
	}
}

func planMultiFile(graph *typegraph.Graph, schemas []*schema.Schema, typeSourceMap map[string]*schema.Schema, language string) *OutputPlan {
	schemaFiles := make(map[string]*OutputFile)

	// For each schema, collect types that belong to this schema and orphaned dependencies
	for _, s := range schemas {
		// Use input filename for output (notification.json → notification.ts)
		// This is predictable and matches source files
		outputPath := convertSchemaPathToOutputForLanguage(s.RelativePath, language)

		// Collect types that belong to this schema and orphaned dependencies
		included := make(map[string]bool)
		var fileTypes []*typegraph.Type

		for _, typ := range graph.Types {
			if typeSourceMap[typ.Name] == s {
				included[typ.Name] = true
				fileTypes = append(fileTypes, typ)
				// Only collect orphaned types (no source schema)
				collectOrphanedTypes(typ, graph, typeSourceMap, included, &fileTypes)
			}
		}

		if len(fileTypes) > 0 {
			schemaFiles[outputPath] = &OutputFile{
				Language:     language,
				RelativePath: outputPath,
				Types:        fileTypes,
				Imports:      []ImportSpec{},
			}
		}
	}

	plan := &OutputPlan{
		Strategy:    StrategyMultiFile,
		Files:       make([]OutputFile, 0, len(schemaFiles)),
		BarrelFiles: []BarrelFile{},
	}

	for _, file := range schemaFiles {
		plan.Files = append(plan.Files, *file)
	}

	return plan
}

func planBundleDeps(graph *typegraph.Graph, schemas []*schema.Schema, typeSourceMap map[string]*schema.Schema, language string, bundleName string) *OutputPlan {
	ext := getLanguageExtension(language)
	filename := fmt.Sprintf("%s%s", bundleName, ext)

	rootSchema := schemas[0]
	includedTypes := collectDependentTypes(graph, rootSchema, typeSourceMap)

	return &OutputPlan{
		Strategy: StrategyBundleDeps,
		Files: []OutputFile{
			{
				Language:     language,
				RelativePath: filename,
				Types:        includedTypes,
				Imports:      []ImportSpec{},
			},
		},
		BarrelFiles: []BarrelFile{},
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

// collectOrphanedTypes recursively collects only orphaned types (types without a source schema)
func collectOrphanedTypes(typ *typegraph.Type, graph *typegraph.Graph, typeSourceMap map[string]*schema.Schema, included map[string]bool, result *[]*typegraph.Type) {
	for _, field := range typ.Fields {
		collectOrphanedFieldReferences(field.Type, graph, typeSourceMap, included, result)
	}

	for _, baseName := range typ.Extends {
		// Only include if orphaned (no source schema)
		if !included[baseName] && typeSourceMap[baseName] == nil {
			if baseType := graph.GetType(baseName); baseType != nil {
				included[baseName] = true
				*result = append(*result, baseType)
				collectOrphanedTypes(baseType, graph, typeSourceMap, included, result)
			}
		}
	}
}

// collectOrphanedFieldReferences recursively collects orphaned types from a TypeRef
func collectOrphanedFieldReferences(ref *typegraph.TypeRef, graph *typegraph.Graph, typeSourceMap map[string]*schema.Schema, included map[string]bool, result *[]*typegraph.Type) {
	if ref == nil {
		return
	}

	// Collect direct type reference only if orphaned
	if ref.TypeName != "" && !included[ref.TypeName] && typeSourceMap[ref.TypeName] == nil {
		if refType := graph.GetType(ref.TypeName); refType != nil {
			included[ref.TypeName] = true
			*result = append(*result, refType)
			collectOrphanedTypes(refType, graph, typeSourceMap, included, result)
		}
	}

	// Collect union members
	for _, member := range ref.UnionMembers {
		collectOrphanedFieldReferences(member, graph, typeSourceMap, included, result)
	}

	// Collect array item types
	if ref.ItemType != nil {
		collectOrphanedFieldReferences(ref.ItemType, graph, typeSourceMap, included, result)
	}

	// Collect map value types
	if ref.ValueType != nil {
		collectOrphanedFieldReferences(ref.ValueType, graph, typeSourceMap, included, result)
	}
}

func convertSchemaPathToOutputForLanguage(schemaPath string, language string) string {
	ext := getLanguageExtension(language)
	dir := filepath.Dir(schemaPath)
	base := filepath.Base(schemaPath)
	nameWithoutExt := base[:len(base)-len(filepath.Ext(base))]

	// Sanitize filename for Python: convert hyphens to underscores
	if language == constants.LanguagePythonShort || language == string(constants.LanguagePython) {
		nameWithoutExt = strings.ReplaceAll(nameWithoutExt, "-", "_")
	}

	outputName := nameWithoutExt + ext

	// If there's a directory structure (not just "."), preserve it
	if dir != "." && dir != "" {
		return filepath.Join(dir, outputName)
	}
	return outputName
}

func getLanguageExtension(language string) string {
	switch language {
	case string(constants.LanguageGo):
		return ".go"
	case constants.LanguageTypeScriptShort, string(constants.LanguageTypeScript):
		return ".ts"
	case constants.LanguagePythonShort, string(constants.LanguagePython):
		return ".py"
	default:
		return ".txt"
	}
}
