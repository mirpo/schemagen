package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"ts", "typescript"},
		{"py", "python"},
		{"rust", "rust"}, // unknown preserved
		{"", ""},         // empty preserved
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeLanguage(tt.input))
		})
	}
}
