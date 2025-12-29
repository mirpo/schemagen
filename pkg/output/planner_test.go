package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

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
	assert.Empty(t, plan.BarrelFiles)
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
