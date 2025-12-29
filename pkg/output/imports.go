package output

import (
	"sort"

	"github.com/mirpo/schemagen/pkg/typegraph"
)

type ImportTracker struct {
	currentFile  string
	dependencies map[string]map[string]bool
}

func NewImportTracker(currentFile string) *ImportTracker {
	return &ImportTracker{
		currentFile:  currentFile,
		dependencies: make(map[string]map[string]bool),
	}
}

func (it *ImportTracker) AddImport(targetFile, typeName string) {
	if targetFile == it.currentFile {
		return
	}

	if it.dependencies[targetFile] == nil {
		it.dependencies[targetFile] = make(map[string]bool)
	}

	it.dependencies[targetFile][typeName] = true
}

func (it *ImportTracker) GetImports() []ImportSpec {
	var imports []ImportSpec

	for targetFile, types := range it.dependencies {
		typeNames := make([]string, 0, len(types))
		for typeName := range types {
			typeNames = append(typeNames, typeName)
		}

		sort.Strings(typeNames)

		imports = append(imports, ImportSpec{
			FromPath:  it.currentFile,
			ToPath:    targetFile,
			TypeNames: typeNames,
		})
	}

	sort.Slice(imports, func(i, j int) bool {
		return imports[i].ToPath < imports[j].ToPath
	})

	return imports
}

func ComputeImports(files []OutputFile, typeToFile map[string]string) []OutputFile {
	for i := range files {
		tracker := NewImportTracker(files[i].RelativePath)

		for _, typ := range files[i].Types {
			collectTypeReferences(typ, tracker, typeToFile)
		}

		imports := tracker.GetImports()
		for j := range imports {
			imports[j].ImportPath = ComputeRelativeImport(
				files[i].RelativePath,
				imports[j].ToPath,
			)
		}

		files[i].Imports = imports
	}

	return files
}

func collectTypeReferences(typ *typegraph.Type, tracker *ImportTracker, typeToFile map[string]string) {
	for _, field := range typ.Fields {
		collectTypeRefReferences(field.Type, tracker, typeToFile)
	}

	for _, baseName := range typ.Extends {
		if targetFile, exists := typeToFile[baseName]; exists {
			tracker.AddImport(targetFile, baseName)
		}
	}

	if typ.ItemType != nil {
		collectTypeRefReferences(typ.ItemType, tracker, typeToFile)
	}

	if typ.ValueType != nil {
		collectTypeRefReferences(typ.ValueType, tracker, typeToFile)
	}

	if typ.TargetType != nil {
		collectTypeRefReferences(typ.TargetType, tracker, typeToFile)
	}
}

func collectTypeRefReferences(typeRef *typegraph.TypeRef, tracker *ImportTracker, typeToFile map[string]string) {
	if typeRef == nil {
		return
	}

	if typeRef.TypeName != "" {
		if targetFile, exists := typeToFile[typeRef.TypeName]; exists {
			tracker.AddImport(targetFile, typeRef.TypeName)
		}
	}

	if typeRef.ItemType != nil {
		collectTypeRefReferences(typeRef.ItemType, tracker, typeToFile)
	}

	if typeRef.ValueType != nil {
		collectTypeRefReferences(typeRef.ValueType, tracker, typeToFile)
	}

	for _, member := range typeRef.UnionMembers {
		collectTypeRefReferences(member, tracker, typeToFile)
	}
}

func BuildTypeToFileMap(files []OutputFile) map[string]string {
	typeToFile := make(map[string]string)

	for _, file := range files {
		for _, typ := range file.Types {
			typeToFile[typ.Name] = file.RelativePath
		}
	}

	return typeToFile
}
