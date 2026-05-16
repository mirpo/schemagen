package output

import (
	"maps"
	"slices"
	"sort"

	"github.com/mirpo/schemagen/pkg/typegraph"
)

type importDependency struct {
	TargetFile string
	TypeNames  []string
}

type importTracker struct {
	currentFile string
	deps        map[string]map[string]struct{}
}

func newImportTracker(currentFile string) *importTracker {
	return &importTracker{
		currentFile: currentFile,
		deps:        make(map[string]map[string]struct{}),
	}
}

func (it *importTracker) AddImport(targetFile, typeName string) {
	if targetFile == it.currentFile {
		return
	}

	if it.deps[targetFile] == nil {
		it.deps[targetFile] = make(map[string]struct{})
	}

	it.deps[targetFile][typeName] = struct{}{}
}

func (it *importTracker) GetDependencies() []importDependency {
	result := make([]importDependency, 0, len(it.deps))

	for target, typeSet := range it.deps {
		typeNames := slices.Sorted(maps.Keys(typeSet))

		result = append(result, importDependency{
			TargetFile: target,
			TypeNames:  typeNames,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TargetFile < result[j].TargetFile
	})

	return result
}

func ComputeImports(files []OutputFile, typeToFile map[string]string) []OutputFile {
	for i := range files {
		tracker := newImportTracker(files[i].RelativePath)

		for _, typ := range files[i].Types {
			collectTypeReferences(typ, tracker, typeToFile)
		}

		deps := tracker.GetDependencies()
		imports := make([]typegraph.ImportSpec, 0, len(deps))

		for _, d := range deps {
			imports = append(imports, typegraph.ImportSpec{
				FromPath:  files[i].RelativePath,
				ToPath:    d.TargetFile,
				TypeNames: d.TypeNames,
				ImportPath: computeRelativeImport(
					files[i].RelativePath,
					d.TargetFile,
				),
			})
		}

		files[i].Imports = imports
	}

	return files
}

func collectTypeReferences(typ *typegraph.Type, tracker *importTracker, typeToFile map[string]string) {
	for _, field := range typ.Fields {
		collectTypeRefReferences(field.Type, tracker, typeToFile)
	}

	for _, base := range typ.Extends {
		if target, ok := typeToFile[base]; ok {
			tracker.AddImport(target, base)
		}
	}

	for _, m := range typ.UnionMembers {
		collectTypeRefReferences(m, tracker, typeToFile)
	}
}

func collectTypeRefReferences(ref *typegraph.TypeRef, tracker *importTracker, typeToFile map[string]string) {
	ref.Walk(func(r *typegraph.TypeRef) {
		if r.TypeName != "" {
			if target, ok := typeToFile[r.TypeName]; ok {
				tracker.AddImport(target, r.TypeName)
			}
		}
	})
}

func BuildTypeToFileMap(files []OutputFile) map[string]string {
	result := make(map[string]string)

	for _, file := range files {
		for _, typ := range file.Types {
			result[typ.Name] = file.RelativePath
		}
	}

	return result
}
