package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirpo/schemagen/pkg/constants"
	"github.com/mirpo/schemagen/pkg/v2/graph"
)

func TestOutputStrategy_Set(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected OutputStrategy
		wantErr  bool
	}{
		{"bundle", "bundle", StrategyBundle, false},
		{"multifile", "multifile", StrategyMultiFile, false},
		{"multi-file", "multi-file", StrategyMultiFile, false},
		{"bundledeps", "bundledeps", StrategyBundleDeps, false},
		{"bundle-deps", "bundle-deps", StrategyBundleDeps, false},
		{"invalid", "invalid", "", true},
		{"empty", "", "", true},
		{"typo", "bundel", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s OutputStrategy
			err := s.Set(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid output strategy")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, s)
			}
		})
	}
}

func TestParseStrategy(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected OutputStrategy
	}{
		{"bundle", "bundle", StrategyBundle},
		{"multi-file", "multi-file", StrategyMultiFile},
		{"multifile", "multifile", StrategyMultiFile},
		{"bundle-deps", "bundle-deps", StrategyBundleDeps},
		{"bundledeps", "bundledeps", StrategyBundleDeps},
		{"invalid falls back to multi-file", "invalid", StrategyMultiFile},
		{"empty falls back to multi-file", "", StrategyMultiFile},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ParseStrategy(tt.input))
		})
	}
}

func TestPlanOutput_Bundle(t *testing.T) {
	g := &graph.Graph{
		Types: []*graph.Type{
			{Name: "User"},
			{Name: "Profile"},
		},
	}

	plan, err := PlanOutput(g, StrategyBundle, constants.LanguageTypeScript, "models", "")
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, StrategyBundle, plan.Strategy)
	require.Len(t, plan.Files, 1)

	file := plan.Files[0]
	assert.Equal(t, "models.ts", file.RelativePath)
	assert.Len(t, file.Types, 2)
	assert.Empty(t, file.Imports)
}

func TestPlanOutput_MultiFile_Basic(t *testing.T) {
	g := &graph.Graph{
		Types: []*graph.Type{
			{Name: "User", SourceFile: "user.json"},
			{Name: "Profile", SourceFile: "profile.json"},
		},
	}

	plan, err := PlanOutput(g, StrategyMultiFile, constants.LanguageTypeScript, "", "")
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, StrategyMultiFile, plan.Strategy)
	require.Len(t, plan.Files, 2)

	found := map[string]bool{}
	for _, f := range plan.Files {
		found[f.RelativePath] = true
	}

	assert.True(t, found["user.ts"])
	assert.True(t, found["profile.ts"])
}

func TestPlanOutput_MultiFile_SameSourceFileGrouped(t *testing.T) {
	g := &graph.Graph{
		Types: []*graph.Type{
			{Name: "User", SourceFile: "user.json"},
			{Name: "Address", SourceFile: "user.json"},
			{Name: "Product", SourceFile: "product.json"},
		},
	}

	plan, err := PlanOutput(g, StrategyMultiFile, constants.LanguageTypeScript, "", "")
	require.NoError(t, err)
	require.Len(t, plan.Files, 2)

	fileTypes := map[string][]string{}
	for _, f := range plan.Files {
		for _, typ := range f.Types {
			fileTypes[f.RelativePath] = append(fileTypes[f.RelativePath], typ.Name)
		}
	}

	assert.ElementsMatch(t, []string{"User", "Address"}, fileTypes["user.ts"])
	assert.ElementsMatch(t, []string{"Product"}, fileTypes["product.ts"])
}

func TestPlanOutput_MultiFile_OrphanedTypesPulledIn(t *testing.T) {
	orphan := &graph.Type{Name: "Timestamped"}
	user := &graph.Type{
		Name:       "User",
		SourceFile: "user.json",
		Extends:    []string{"Timestamped"},
	}

	g := &graph.Graph{
		Types: []*graph.Type{user, orphan},
	}

	plan, err := PlanOutput(g, StrategyMultiFile, constants.LanguageTypeScript, "", "")
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	file := plan.Files[0]
	require.Len(t, file.Types, 2)

	typeNames := map[string]bool{}
	for _, t := range file.Types {
		typeNames[t.Name] = true
	}

	assert.True(t, typeNames["User"])
	assert.True(t, typeNames["Timestamped"])
}

func TestPlanOutput_BundleDeps(t *testing.T) {
	base := &graph.Type{Name: "Base", SourceFile: "base.json"}
	child := &graph.Type{
		Name:       "Child",
		SourceFile: "child.json",
		Extends:    []string{"Base"},
	}
	unused := &graph.Type{Name: "Unused", SourceFile: "unused.json"}

	g := &graph.Graph{
		Types: []*graph.Type{base, child, unused},
	}

	plan, err := PlanOutput(g, StrategyBundleDeps, constants.LanguageTypeScript, "bundle", "child.json")
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	file := plan.Files[0]
	assert.Equal(t, "bundle.ts", file.RelativePath)

	typeNames := map[string]bool{}
	for _, t := range file.Types {
		typeNames[t.Name] = true
	}

	assert.True(t, typeNames["Child"])
	assert.True(t, typeNames["Base"])
	assert.False(t, typeNames["Unused"])
}

func TestPlanOutput_BundleDeps_IncludesUnionMembers(t *testing.T) {
	circle := &graph.Type{Name: "Circle", Kind: graph.KindStruct, SourceFile: "circle.json"}
	square := &graph.Type{Name: "Square", Kind: graph.KindStruct, SourceFile: "square.json"}
	shape := &graph.Type{
		Name:       "Shape",
		Kind:       graph.KindUnion,
		SourceFile: "shape.json",
		UnionMembers: []*graph.TypeRef{
			{Kind: graph.KindRef, TypeName: "Circle"},
			{Kind: graph.KindRef, TypeName: "Square"},
		},
	}

	g := graph.NewGraph()
	g.AddType(circle)
	g.AddType(square)
	g.AddType(shape)

	plan, err := PlanOutput(g, StrategyBundleDeps, constants.LanguageTypeScript, "bundle", "shape.json")
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	typeNames := map[string]bool{}
	for _, typ := range plan.Files[0].Types {
		typeNames[typ.Name] = true
	}

	assert.True(t, typeNames["Shape"])
	assert.True(t, typeNames["Circle"])
	assert.True(t, typeNames["Square"])
}

func TestPlanOutput_MultiFile_DeterministicOrder(t *testing.T) {
	types := make([]*graph.Type, 10)
	for i := range 10 {
		name := string(rune('A' + i))
		types[i] = &graph.Type{Name: name, SourceFile: name + ".json"}
	}

	g := &graph.Graph{Types: types}

	var firstOrder []string
	for run := range 50 {
		plan, err := PlanOutput(g, StrategyMultiFile, constants.LanguageTypeScript, "", "")
		require.NoError(t, err)
		require.Len(t, plan.Files, 10)

		var order []string
		for _, f := range plan.Files {
			order = append(order, f.RelativePath)
		}

		if run == 0 {
			firstOrder = order
		} else {
			assert.Equal(t, firstOrder, order,
				"file order must be deterministic across runs (failed on run %d)", run)
		}
	}
}

func TestPlanOutput_BundleDeps_EmptyRootSourceFile(t *testing.T) {
	g := &graph.Graph{
		Types: []*graph.Type{
			{Name: "Orphan"},
		},
	}

	plan, err := PlanOutput(g, StrategyBundleDeps, constants.LanguageTypeScript, "bundle", "")
	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Empty(t, plan.Files)
}
