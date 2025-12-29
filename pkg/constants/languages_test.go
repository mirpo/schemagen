package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "TypeScript short form",
			input:    "ts",
			expected: "typescript",
		},
		{
			name:     "TypeScript full form",
			input:    "typescript",
			expected: "typescript",
		},
		{
			name:     "Python short form",
			input:    "py",
			expected: "python",
		},
		{
			name:     "Python full form",
			input:    "python",
			expected: "python",
		},
		{
			name:     "Go short form",
			input:    "go",
			expected: "go",
		},
		{
			name:     "Unknown language",
			input:    "rust",
			expected: "rust",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeLanguage(tt.input)
			assert.Equal(t, tt.expected, result, "NormalizeLanguage() should return expected value")
		})
	}
}
