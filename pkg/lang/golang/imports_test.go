package golang

import (
	"testing"

	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportConverter_Convert(t *testing.T) {
	converter := &ImportConverter{ModulePath: "github.com/test/project"}

	result := converter.Convert([]typegraph.ImportSpec{
		{FromPath: "events/event.go", ImportPath: "./header", TypeNames: []string{"Header"}},
		{ImportPath: "github.com/pkg/errors"},
	})

	require.Len(t, result, 2)
	assert.Equal(t, "github.com/test/project/events/header", result[0].ImportPath)
	assert.Equal(t, "github.com/pkg/errors", result[1].ImportPath)
}

func TestImportConverter_ResolveRelative(t *testing.T) {
	tests := []struct {
		name, relPath, fromFile, modulePath, expected string
	}{
		{"same dir", "./header", "events/event.go", "github.com/test/project", "github.com/test/project/events/header"},
		{"parent dir", "../common/types", "events/event.go", "github.com/test/project", "github.com/test/project/common/types"},
		{"no module path", "./header", "events/event.go", "", "events/header"},
		{"external unchanged", "github.com/pkg/errors", "events/event.go", "github.com/test/project", "github.com/pkg/errors"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := &ImportConverter{ModulePath: tt.modulePath}
			assert.Equal(t, tt.expected, converter.resolveRelative(tt.relPath, tt.fromFile))
		})
	}
}
