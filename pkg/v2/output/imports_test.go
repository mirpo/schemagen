package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirpo/schemagen/pkg/v2/graph"
)

func TestImportTracker_BasicBehavior(t *testing.T) {
	tracker := newImportTracker("event.ts")

	tracker.AddImport("header.ts", "Header")
	tracker.AddImport("meta.ts", "Meta")
	tracker.AddImport("header.ts", "Header") // duplicate

	deps := tracker.GetDependencies()

	require.Len(t, deps, 2)

	assert.Equal(t, "header.ts", deps[0].TargetFile)
	assert.Equal(t, []string{"Header"}, deps[0].TypeNames)

	assert.Equal(t, "meta.ts", deps[1].TargetFile)
	assert.Equal(t, []string{"Meta"}, deps[1].TypeNames)
}

func TestImportTracker_SelfReferenceSkipped(t *testing.T) {
	tracker := newImportTracker("event.ts")

	tracker.AddImport("event.ts", "Event")
	tracker.AddImport("other.ts", "Other")

	deps := tracker.GetDependencies()

	require.Len(t, deps, 1)
	assert.Equal(t, "other.ts", deps[0].TargetFile)
}

func TestImportTracker_TypeNamesSorted(t *testing.T) {
	tracker := newImportTracker("a.ts")

	tracker.AddImport("b.ts", "Z")
	tracker.AddImport("b.ts", "A")
	tracker.AddImport("b.ts", "M")

	deps := tracker.GetDependencies()

	require.Len(t, deps, 1)
	assert.Equal(t, []string{"A", "M", "Z"}, deps[0].TypeNames)
}

func TestCollectTypeReferences_DeepTraversal(t *testing.T) {
	typ := &graph.Type{
		Name: "Wrapper",
		Fields: []*graph.Field{
			{
				Type: &graph.TypeRef{
					Kind: graph.KindMap,
					ValueType: &graph.TypeRef{
						Kind: graph.KindArray,
						ItemType: &graph.TypeRef{
							Kind: graph.KindUnion,
							UnionMembers: []*graph.TypeRef{
								{TypeName: "A"},
								{TypeName: "B"},
							},
						},
					},
				},
			},
		},
	}

	tracker := newImportTracker("wrapper.ts")
	typeToFile := map[string]string{
		"A": "a.ts",
		"B": "b.ts",
	}

	collectTypeReferences(typ, tracker, typeToFile)

	deps := tracker.GetDependencies()
	require.Len(t, deps, 2)

	assert.Equal(t, "a.ts", deps[0].TargetFile)
	assert.Equal(t, "b.ts", deps[1].TargetFile)
}

func TestCollectTypeReferences_UnionMembers(t *testing.T) {
	typ := &graph.Type{
		Name: "Shape",
		Kind: graph.KindUnion,
		UnionMembers: []*graph.TypeRef{
			{Kind: graph.KindRef, TypeName: "Circle"},
			{Kind: graph.KindRef, TypeName: "Square"},
			{Kind: graph.KindRef, TypeName: "Triangle"},
		},
	}

	tracker := newImportTracker("shape.ts")
	typeToFile := map[string]string{
		"Circle":   "circle.ts",
		"Square":   "square.ts",
		"Triangle": "triangle.ts",
	}

	collectTypeReferences(typ, tracker, typeToFile)

	deps := tracker.GetDependencies()
	require.Len(t, deps, 3)

	assert.Equal(t, "circle.ts", deps[0].TargetFile)
	assert.Equal(t, []string{"Circle"}, deps[0].TypeNames)
	assert.Equal(t, "square.ts", deps[1].TargetFile)
	assert.Equal(t, "triangle.ts", deps[2].TargetFile)
}

func TestComputeImports_Integration(t *testing.T) {
	files := []OutputFile{
		{
			RelativePath: "event.ts",
			Types: []*graph.Type{
				{
					Name: "Event",
					Fields: []*graph.Field{
						{
							Type: &graph.TypeRef{
								TypeName: "Header",
							},
						},
					},
				},
			},
		},
		{
			RelativePath: "header.ts",
			Types: []*graph.Type{
				{Name: "Header"},
			},
		},
	}

	typeToFile := BuildTypeToFileMap(files)
	files = ComputeImports(files, typeToFile)

	require.Len(t, files[0].Imports, 1)
	assert.Equal(t, "./header", files[0].Imports[0].ImportPath)
	assert.Equal(t, []string{"Header"}, files[0].Imports[0].TypeNames)

	assert.Empty(t, files[1].Imports)
}
