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
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeLanguage(tt.input))
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
		{"python full", "python", ".py"},
		{"go", "go", ".go"},
		{"unknown", "rust", ".txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetExtension(tt.input))
		})
	}
}

func TestIsPython(t *testing.T) {
	assert.True(t, IsPython("python"))
	assert.False(t, IsPython("typescript"))
}

func TestGetBarrelFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"typescript full", "typescript", "index.ts"},
		{"python full", "python", "__init__.py"},
		{"go has no barrel", "go", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetBarrelFileName(tt.input))
		})
	}
}
