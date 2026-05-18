package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirpo/schemagen/pkg/v2/graph"
	"github.com/mirpo/schemagen/pkg/v2/parse"
)

func TestOutputStrategy_Set(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected OutputStrategy
		wantErr  bool
	}{
		// Valid inputs
		{"bundle", "bundle", StrategyBundle, false},
		{"multifile", "multifile", StrategyMultiFile, false},
		{"multi-file", "multi-file", StrategyMultiFile, false},
		{"bundledeps", "bundledeps", StrategyBundleDeps, false},
		{"bundle-deps", "bundle-deps", StrategyBundleDeps, false},
		// Invalid inputs should error
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

func TestPlanOutput_Bundle(t *testing.T) {
	g := &graph.Graph{
		Types: []*graph.Type{
			{Name: "User"},
			{Name: "Profile"},
		},
	}

	plan, err := PlanOutput(
		g,
		nil,
		StrategyBundle,
		"ts",
		"models",
	)

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
	userSchema := parse.NamedSchema{
		Name: "User",
		Path: "user.json",
	}
	profileSchema := parse.NamedSchema{
		Name: "Profile",
		Path: "profile.json",
	}

	g := &graph.Graph{
		Types: []*graph.Type{
			{Name: "User"},
			{Name: "Profile"},
		},
	}

	plan, err := PlanOutput(
		g,
		[]parse.NamedSchema{userSchema, profileSchema},
		StrategyMultiFile,
		"ts",
		"",
	)

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

func TestPlanOutput_MultiFile_IncludesOrphanedTypes(t *testing.T) {
	orphan := &graph.Type{
		Name: "Timestamped",
	}

	user := &graph.Type{
		Name:    "User",
		Extends: []string{"Timestamped"},
	}

	g := &graph.Graph{
		Types: []*graph.Type{
			user,
			orphan,
		},
	}

	userSchema := parse.NamedSchema{
		Name: "User",
		Path: "user.json",
	}

	plan, err := PlanOutput(
		g,
		[]parse.NamedSchema{userSchema},
		StrategyMultiFile,
		"ts",
		"",
	)

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
	base := &graph.Type{
		Name: "Base",
	}

	child := &graph.Type{
		Name:    "Child",
		Extends: []string{"Base"},
	}

	unused := &graph.Type{
		Name: "Unused",
	}

	g := &graph.Graph{
		Types: []*graph.Type{
			base,
			child,
			unused,
		},
	}

	childSchema := parse.NamedSchema{
		Name: "Child",
		Path: "child.json",
	}

	plan, err := PlanOutput(
		g,
		[]parse.NamedSchema{childSchema},
		StrategyBundleDeps,
		"ts",
		"bundle",
	)

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
	circle := &graph.Type{Name: "Circle", Kind: graph.KindStruct}
	square := &graph.Type{Name: "Square", Kind: graph.KindStruct}
	shape := &graph.Type{
		Name: "Shape",
		Kind: graph.KindUnion,
		UnionMembers: []*graph.TypeRef{
			{Kind: graph.KindRef, TypeName: "Circle"},
			{Kind: graph.KindRef, TypeName: "Square"},
		},
	}

	g := &graph.Graph{
		Types: []*graph.Type{circle, square, shape},
	}
	g.AddType(circle)
	g.AddType(square)
	g.AddType(shape)

	shapeSchema := parse.NamedSchema{
		Name: "Shape",
		Path: "shape.json",
	}

	plan, err := PlanOutput(
		g,
		[]parse.NamedSchema{shapeSchema},
		StrategyBundleDeps,
		"ts",
		"bundle",
	)

	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	typeNames := map[string]bool{}
	for _, typ := range plan.Files[0].Types {
		typeNames[typ.Name] = true
	}

	assert.True(t, typeNames["Shape"], "Shape should be included")
	assert.True(t, typeNames["Circle"], "Circle should be included as union member")
	assert.True(t, typeNames["Square"], "Square should be included as union member")
}

func TestPlanOutput_MultiFile_DeterministicOrder(t *testing.T) {
	schemas := make([]parse.NamedSchema, 10)
	types := make([]*graph.Type, 10)
	for i := range 10 {
		name := string(rune('A' + i))
		schemas[i] = parse.NamedSchema{Name: name, Path: name + ".json"}
		types[i] = &graph.Type{Name: name}
	}

	g := &graph.Graph{Types: types}

	var firstOrder []string
	for run := range 50 {
		plan, err := PlanOutput(g, schemas, StrategyMultiFile, "ts", "")
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

func TestPlanOutput_BundleDeps_EmptySchemas(t *testing.T) {
	g := &graph.Graph{
		Types: []*graph.Type{
			{Name: "Orphan"},
		},
	}

	plan, err := PlanOutput(
		g,
		[]parse.NamedSchema{},
		StrategyBundleDeps,
		"ts",
		"bundle",
	)

	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Empty(t, plan.Files)
}
