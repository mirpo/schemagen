package output

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeRelativeImport_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		expected string
	}{
		{
			name:     "same directory",
			from:     "events/event.ts",
			to:       "events/header.ts",
			expected: "./header",
		},
		{
			name:     "subdirectory",
			from:     "events/event.ts",
			to:       "events/payloads/subscribe.ts",
			expected: "./payloads/subscribe",
		},
		{
			name:     "parent directory",
			from:     "events/payloads/subscribe.ts",
			to:       "events/header.ts",
			expected: "../header",
		},
		{
			name:     "sibling directory",
			from:     "api/users/user.ts",
			to:       "api/auth/token.ts",
			expected: "../auth/token",
		},
		{
			name:     "deep nesting",
			from:     "a/b/c/d.ts",
			to:       "a/e/f.ts",
			expected: "../../e/f",
		},
		{
			name:     "root files",
			from:     "a.ts",
			to:       "b.ts",
			expected: "./b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeRelativeImport(tt.from, tt.to)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathMapper_InputPathToOutputPath_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		language string
		input    string
		expected string
	}{
		{
			name:     "typescript flat",
			language: "ts",
			input:    "/input/user.json",
			expected: "user.ts",
		},
		{
			name:     "typescript nested",
			language: "ts",
			input:    "/input/events/header.json",
			expected: filepath.Join("events", "header.ts"),
		},
		{
			name:     "python hyphen sanitize",
			language: "py",
			input:    "/input/user-profile.json",
			expected: "user_profile.py",
		},
		{
			name:     "python deep nested sanitize",
			language: "py",
			input:    "/input/edge-cases/special-props.json",
			expected: filepath.Join("edge-cases", "special_props.py"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewPathMapper("/input", "/output", tt.language)
			result := pm.InputPathToOutputPath(filepath.FromSlash(tt.input))
			assert.Equal(t, filepath.FromSlash(tt.expected), result)
		})
	}
}

func TestGetDirectoryLevels(t *testing.T) {
	result := GetDirectoryLevels(filepath.FromSlash("a/b/c.ts"))
	expected := []string{filepath.FromSlash("a"), filepath.FromSlash("a/b")}
	assert.Equal(t, expected, result)
}
