package py

import (
	"testing"

	"github.com/mirpo/schemagen/pkg/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportConverter_Convert(t *testing.T) {
	converter := &ImportConverter{}

	tests := []struct {
		name, input, expected string
	}{
		{"same dir", "./module", ".module"},
		{"sub dir", "./dir/module", ".dir.module"},
		{"parent dir", "../common", "..common"},
		{"nested parent", "../../types", "...types"},
		{"absolute untouched", "pydantic", "pydantic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert([]graph.ImportSpec{{ImportPath: tt.input}})
			require.Len(t, result, 1)
			assert.Equal(t, tt.expected, result[0].ImportPath)
		})
	}
}
