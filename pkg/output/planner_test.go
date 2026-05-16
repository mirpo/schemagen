package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/mirpo/schemagen/pkg/typegraph"
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
	graph := &typegraph.Graph{
		Types: []*typegraph.Type{
			{Name: "User"},
			{Name: "Profile"},
		},
	}

	plan, err := PlanOutput(
		graph,
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
	userSchema := &schema.Schema{
		Name:         "User",
		RelativePath: "user.json",
	}
	profileSchema := &schema.Schema{
		Name:         "Profile",
		RelativePath: "profile.json",
	}

	graph := &typegraph.Graph{
		Types: []*typegraph.Type{
			{Name: "User"},
			{Name: "Profile"},
		},
	}

	plan, err := PlanOutput(
		graph,
		[]*schema.Schema{userSchema, profileSchema},
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
	orphan := &typegraph.Type{
		Name: "Timestamped",
	}

	user := &typegraph.Type{
		Name:    "User",
		Extends: []string{"Timestamped"},
	}

	graph := &typegraph.Graph{
		Types: []*typegraph.Type{
			user,
			orphan,
		},
	}

	userSchema := &schema.Schema{
		Name:         "User",
		RelativePath: "user.json",
	}

	plan, err := PlanOutput(
		graph,
		[]*schema.Schema{userSchema},
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
	base := &typegraph.Type{
		Name: "Base",
	}

	child := &typegraph.Type{
		Name:    "Child",
		Extends: []string{"Base"},
	}

	unused := &typegraph.Type{
		Name: "Unused",
	}

	graph := &typegraph.Graph{
		Types: []*typegraph.Type{
			base,
			child,
			unused,
		},
	}

	childSchema := &schema.Schema{
		Name:         "Child",
		RelativePath: "child.json",
	}

	plan, err := PlanOutput(
		graph,
		[]*schema.Schema{childSchema},
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
	circle := &typegraph.Type{Name: "Circle", Kind: typegraph.KindStruct}
	square := &typegraph.Type{Name: "Square", Kind: typegraph.KindStruct}
	shape := &typegraph.Type{
		Name: "Shape",
		Kind: typegraph.KindUnion,
		UnionMembers: []*typegraph.TypeRef{
			{Kind: typegraph.KindRef, TypeName: "Circle"},
			{Kind: typegraph.KindRef, TypeName: "Square"},
		},
	}

	graph := &typegraph.Graph{
		Types: []*typegraph.Type{circle, square, shape},
	}
	graph.AddType(circle)
	graph.AddType(square)
	graph.AddType(shape)

	shapeSchema := &schema.Schema{
		Name:         "Shape",
		RelativePath: "shape.json",
	}

	plan, err := PlanOutput(
		graph,
		[]*schema.Schema{shapeSchema},
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
	schemas := make([]*schema.Schema, 10)
	types := make([]*typegraph.Type, 10)
	for i := range 10 {
		name := string(rune('A' + i))
		schemas[i] = &schema.Schema{Name: name, RelativePath: name + ".json"}
		types[i] = &typegraph.Type{Name: name}
	}

	graph := &typegraph.Graph{Types: types}

	var firstOrder []string
	for run := range 50 {
		plan, err := PlanOutput(graph, schemas, StrategyMultiFile, "ts", "")
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
