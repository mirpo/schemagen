package output

import (
	"sort"

	"github.com/mirpo/schemagen/pkg/typegraph"
)

type ImportDependency struct {
	TargetFile string
	TypeNames  []string
}

type ImportTracker struct {
	currentFile string
	deps        map[string]map[string]struct{}
}

func NewImportTracker(currentFile string) *ImportTracker {
	return &ImportTracker{
		currentFile: currentFile,
		deps:        make(map[string]map[string]struct{}),
	}
}

func (it *ImportTracker) AddImport(targetFile, typeName string) {
	if targetFile == it.currentFile {
		return
	}

	if it.deps[targetFile] == nil {
		it.deps[targetFile] = make(map[string]struct{})
	}

	it.deps[targetFile][typeName] = struct{}{}
}

func (it *ImportTracker) GetDependencies() []ImportDependency {
	result := make([]ImportDependency, 0, len(it.deps))

	for target, typeSet := range it.deps {
		typeNames := make([]string, 0, len(typeSet))
		for name := range typeSet {
			typeNames = append(typeNames, name)
		}

		sort.Strings(typeNames)

		result = append(result, ImportDependency{
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
		tracker := NewImportTracker(files[i].RelativePath)

		for _, typ := range files[i].Types {
			collectTypeReferences(typ, tracker, typeToFile)
		}

		deps := tracker.GetDependencies()
		imports := make([]ImportSpec, 0, len(deps))

		for _, d := range deps {
			imports = append(imports, ImportSpec{
				FromPath:  files[i].RelativePath,
				ToPath:    d.TargetFile,
				TypeNames: d.TypeNames,
				ImportPath: ComputeRelativeImport(
					files[i].RelativePath,
					d.TargetFile,
				),
			})
		}

		files[i].Imports = imports
	}

	return files
}

func collectTypeReferences(typ *typegraph.Type, tracker *ImportTracker, typeToFile map[string]string) {
	for _, field := range typ.Fields {
		collectTypeRefReferences(field.Type, tracker, typeToFile)
	}

	for _, base := range typ.Extends {
		if target, ok := typeToFile[base]; ok {
			tracker.AddImport(target, base)
		}
	}

	collectTypeRefReferences(typ.ItemType, tracker, typeToFile)
	collectTypeRefReferences(typ.ValueType, tracker, typeToFile)
	collectTypeRefReferences(typ.TargetType, tracker, typeToFile)
}

func collectTypeRefReferences(ref *typegraph.TypeRef, tracker *ImportTracker, typeToFile map[string]string) {
	if ref == nil {
		return
	}

	if ref.TypeName != "" {
		if target, ok := typeToFile[ref.TypeName]; ok {
			tracker.AddImport(target, ref.TypeName)
		}
	}

	collectTypeRefReferences(ref.ItemType, tracker, typeToFile)
	collectTypeRefReferences(ref.ValueType, tracker, typeToFile)

	for _, m := range ref.UnionMembers {
		collectTypeRefReferences(m, tracker, typeToFile)
	}
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
