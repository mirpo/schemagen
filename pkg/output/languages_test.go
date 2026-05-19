package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShortName(t *testing.T) {
	assert.Equal(t, "ts", ShortName(LanguageTypeScript))
	assert.Equal(t, "py", ShortName(LanguagePython))
	assert.Equal(t, "go", ShortName(LanguageGo))
	assert.Equal(t, "rust", ShortName(Language("rust")))
}

func TestGetExtension(t *testing.T) {
	tests := []struct {
		name     string
		input    Language
		expected string
	}{
		{"typescript", LanguageTypeScript, ".ts"},
		{"python", LanguagePython, ".py"},
		{"go", LanguageGo, ".go"},
		{"unknown", Language("rust"), ".txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetExtension(tt.input))
		})
	}
}

func TestGetBarrelFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    Language
		expected string
	}{
		{"typescript", LanguageTypeScript, "index.ts"},
		{"python", LanguagePython, "__init__.py"},
		{"go has no barrel", LanguageGo, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetBarrelFileName(tt.input))
		})
	}
}
