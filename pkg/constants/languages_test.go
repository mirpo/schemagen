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

func TestGetExtension(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"typescript full", "typescript", ".ts"},
		{"typescript short", "ts", ".ts"},
		{"python full", "python", ".py"},
		{"python short", "py", ".py"},
		{"go", "go", ".go"},
		{"unknown", "rust", ".txt"},
		{"empty", "", ".txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetExtension(tt.input))
		})
	}
}

func TestIsPython(t *testing.T) {
	assert.True(t, IsPython("python"))
	assert.True(t, IsPython("py"))
	assert.False(t, IsPython("typescript"))
	assert.False(t, IsPython("go"))
	assert.False(t, IsPython(""))
}

func TestGetBarrelFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"typescript full", "typescript", "index.ts"},
		{"typescript short", "ts", "index.ts"},
		{"python full", "python", "__init__.py"},
		{"python short", "py", "__init__.py"},
		{"go has no barrel", "go", ""},
		{"unknown has no barrel", "rust", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetBarrelFileName(tt.input))
		})
	}
}
