package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirpo/schemagen/pkg/typegraph"
)

func TestImportTracker_AddImport(t *testing.T) {
	tracker := NewImportTracker("event.ts")

	tracker.AddImport("header.ts", "EventHeader")
	tracker.AddImport("meta.ts", "EventMetadata")

	imports := tracker.GetImports()

	assert.Len(t, imports, 2, "should have 2 imports")
}

func TestImportTracker_Deduplication(t *testing.T) {
	tracker := NewImportTracker("event.ts")

	tracker.AddImport("header.ts", "EventHeader")
	tracker.AddImport("header.ts", "EventHeader")
	tracker.AddImport("header.ts", "EventType")

	imports := tracker.GetImports()

	require.Len(t, imports, 1, "should deduplicate to 1 import")
	require.Len(t, imports[0].TypeNames, 2, "should have 2 type names")

	expectedTypes := []string{"EventHeader", "EventType"}
	for i, expected := range expectedTypes {
		assert.Equal(t, expected, imports[0].TypeNames[i], "type name should match")
	}
}

func TestImportTracker_SelfReference_Skipped(t *testing.T) {
	tracker := NewImportTracker("event.ts")

	tracker.AddImport("event.ts", "Event")
	tracker.AddImport("header.ts", "EventHeader")

	imports := tracker.GetImports()

	require.Len(t, imports, 1, "should skip self-reference")
	assert.Equal(t, "header.ts", imports[0].ToPath, "import should be from header.ts")
}

func TestImportTracker_Sorting(t *testing.T) {
	tracker := NewImportTracker("event.ts")

	tracker.AddImport("z.ts", "Z")
	tracker.AddImport("a.ts", "A")
	tracker.AddImport("m.ts", "M")

	imports := tracker.GetImports()

	expectedOrder := []string{"a.ts", "m.ts", "z.ts"}
	for i, expected := range expectedOrder {
		assert.Equal(t, expected, imports[i].ToPath, "import should be sorted")
	}
}

func TestImportTracker_TypeNamesSorted(t *testing.T) {
	tracker := NewImportTracker("event.ts")

	tracker.AddImport("common.ts", "Z")
	tracker.AddImport("common.ts", "A")
	tracker.AddImport("common.ts", "M")

	imports := tracker.GetImports()

	require.Len(t, imports, 1, "should have 1 import")

	expectedOrder := []string{"A", "M", "Z"}
	for i, expected := range expectedOrder {
		assert.Equal(t, expected, imports[0].TypeNames[i], "type names should be sorted")
	}
}

func TestBuildTypeToFileMap(t *testing.T) {
	files := []OutputFile{
		{
			RelativePath: "event.ts",
			Types: []*typegraph.Type{
				{Name: "Event"},
				{Name: "EventPayload"},
			},
		},
		{
			RelativePath: "header.ts",
			Types: []*typegraph.Type{
				{Name: "EventHeader"},
			},
		},
	}

	typeToFile := BuildTypeToFileMap(files)

	expected := map[string]string{
		"Event":        "event.ts",
		"EventPayload": "event.ts",
		"EventHeader":  "header.ts",
	}

	for typeName, expectedFile := range expected {
		actualFile, exists := typeToFile[typeName]
		assert.True(t, exists, "type %s should exist in map", typeName)
		assert.Equal(t, expectedFile, actualFile, "type %s should map to correct file", typeName)
	}
}

func TestCollectTypeReferences_Fields(t *testing.T) {
	typ := &typegraph.Type{
		Name: "Event",
		Fields: []*typegraph.Field{
			{
				Name: "Header",
				Type: &typegraph.TypeRef{
					TypeName: "EventHeader",
				},
			},
			{
				Name: "Meta",
				Type: &typegraph.TypeRef{
					TypeName: "EventMetadata",
				},
			},
		},
	}

	tracker := NewImportTracker("event.ts")
	typeToFile := map[string]string{
		"EventHeader":   "header.ts",
		"EventMetadata": "meta.ts",
	}

	collectTypeReferences(typ, tracker, typeToFile)

	imports := tracker.GetImports()

	require.Len(t, imports, 2, "should collect 2 imports")

	foundHeader := false
	foundMeta := false
	for _, imp := range imports {
		if imp.ToPath == "header.ts" {
			foundHeader = true
		}
		if imp.ToPath == "meta.ts" {
			foundMeta = true
		}
	}

	assert.True(t, foundHeader, "should find import from header.ts")
	assert.True(t, foundMeta, "should find import from meta.ts")
}

func TestCollectTypeReferences_Extends(t *testing.T) {
	typ := &typegraph.Type{
		Name:    "User",
		Extends: []string{"IdObject", "Timestamped"},
	}

	tracker := NewImportTracker("user.ts")
	typeToFile := map[string]string{
		"IdObject":    "common/id.ts",
		"Timestamped": "common/timestamps.ts",
	}

	collectTypeReferences(typ, tracker, typeToFile)

	imports := tracker.GetImports()

	assert.Len(t, imports, 2, "should collect 2 imports from Extends")
}

func TestCollectTypeReferences_Array(t *testing.T) {
	typ := &typegraph.Type{
		Name: "UserList",
		Kind: typegraph.KindArray,
		ItemType: &typegraph.TypeRef{
			TypeName: "User",
		},
	}

	tracker := NewImportTracker("user-list.ts")
	typeToFile := map[string]string{
		"User": "user.ts",
	}

	collectTypeReferences(typ, tracker, typeToFile)

	imports := tracker.GetImports()

	require.Len(t, imports, 1, "should collect 1 import")
	assert.Equal(t, "user.ts", imports[0].ToPath, "import should be from user.ts")
}

func TestCollectTypeReferences_Union(t *testing.T) {
	typ := &typegraph.Type{
		Name: "Result",
		Fields: []*typegraph.Field{
			{
				Name: "Value",
				Type: &typegraph.TypeRef{
					UnionMembers: []*typegraph.TypeRef{
						{TypeName: "Success"},
						{TypeName: "Error"},
					},
				},
			},
		},
	}

	tracker := NewImportTracker("result.ts")
	typeToFile := map[string]string{
		"Success": "success.ts",
		"Error":   "error.ts",
	}

	collectTypeReferences(typ, tracker, typeToFile)

	imports := tracker.GetImports()

	assert.Len(t, imports, 2, "should collect 2 imports from union members")
}

func TestComputeImports_Integration(t *testing.T) {
	files := []OutputFile{
		{
			RelativePath: "event.ts",
			Types: []*typegraph.Type{
				{
					Name: "Event",
					Fields: []*typegraph.Field{
						{
							Name: "Header",
							Type: &typegraph.TypeRef{TypeName: "EventHeader"},
						},
					},
				},
			},
		},
		{
			RelativePath: "header.ts",
			Types: []*typegraph.Type{
				{Name: "EventHeader"},
			},
		},
	}

	typeToFile := BuildTypeToFileMap(files)
	files = ComputeImports(files, typeToFile)

	require.Len(t, files[0].Imports, 1, "event.ts should have 1 import")
	assert.Equal(t, "./header", files[0].Imports[0].ImportPath, "import path should be ./header")
	assert.Empty(t, files[1].Imports, "header.ts should have no imports")
}
