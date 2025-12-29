package output

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeRelativeImport_SameDirectory(t *testing.T) {
	fromFile := "events/event.ts"
	toFile := "events/header.ts"

	result := ComputeRelativeImport(fromFile, toFile)
	expected := "./header"

	assert.Equal(t, expected, result, "relative import should be correct")
}

func TestComputeRelativeImport_Subdirectory(t *testing.T) {
	fromFile := "events/event.ts"
	toFile := "events/payloads/subscribe.ts"

	result := ComputeRelativeImport(fromFile, toFile)
	expected := "./payloads/subscribe"

	assert.Equal(t, expected, result, "relative import should be correct")
}

func TestComputeRelativeImport_ParentDirectory(t *testing.T) {
	fromFile := "events/payloads/subscribe.ts"
	toFile := "events/header.ts"

	result := ComputeRelativeImport(fromFile, toFile)
	expected := "../header"

	assert.Equal(t, expected, result, "relative import should be correct")
}

func TestComputeRelativeImport_SiblingDirectory(t *testing.T) {
	fromFile := "api/users/user.ts"
	toFile := "api/auth/token.ts"

	result := ComputeRelativeImport(fromFile, toFile)
	expected := "../auth/token"

	assert.Equal(t, expected, result, "relative import should be correct")
}

func TestComputeRelativeImport_DeeplyNested(t *testing.T) {
	fromFile := "a/b/c/d.ts"
	toFile := "a/e/f.ts"

	result := ComputeRelativeImport(fromFile, toFile)
	expected := "../../e/f"

	assert.Equal(t, expected, result, "relative import should be correct")
}

func TestPathMapper_InputPathToOutputPath_Flat(t *testing.T) {
	pm := NewPathMapper("/input", "/output", "ts")

	input := filepath.Join("/input", "user.json")
	result := pm.InputPathToOutputPath(input)
	expected := "user.ts"

	assert.Equal(t, expected, result, "output path should be correct")
}

func TestPathMapper_InputPathToOutputPath_Nested(t *testing.T) {
	pm := NewPathMapper("/input", "/output", "ts")

	input := filepath.Join("/input", "events", "header.json")
	result := pm.InputPathToOutputPath(input)
	expected := filepath.Join("events", "header.ts")

	assert.Equal(t, expected, result, "output path should be correct")
}

func TestPathMapper_InputPathToOutputPath_DeeplyNested(t *testing.T) {
	pm := NewPathMapper("/input", "/output", "py")

	input := filepath.Join("/input", "events", "payloads", "v1", "subscribe.json")
	result := pm.InputPathToOutputPath(input)
	expected := filepath.Join("events", "payloads", "v1", "subscribe.py")

	assert.Equal(t, expected, result, "output path should be correct")
}

func TestPathMapper_ComputeImportPath_SameDir(t *testing.T) {
	pm := NewPathMapper("/input", "/output", "ts")

	result := pm.ComputeImportPath("events/event.ts", "events/header.ts")
	expected := "./header"

	assert.Equal(t, expected, result, "import path should be correct")
}

func TestPathMapper_ComputeImportPath_ParentDir(t *testing.T) {
	pm := NewPathMapper("/input", "/output", "ts")

	result := pm.ComputeImportPath("events/payloads/subscribe.ts", "events/header.ts")
	expected := "../header"

	assert.Equal(t, expected, result, "import path should be correct")
}

func TestPathMapper_BarrelFilePath_TypeScript(t *testing.T) {
	pm := NewPathMapper("/input", "/output", "ts")

	tests := []struct {
		dir      string
		expected string
	}{
		{"", "index.ts"},
		{".", "index.ts"},
		{"events", filepath.Join("events", "index.ts")},
		{"events/payloads", filepath.Join("events", "payloads", "index.ts")},
	}

	for _, test := range tests {
		result := pm.BarrelFilePath(test.dir)
		assert.Equal(t, test.expected, result, "barrel file path should be correct for dir %s", test.dir)
	}
}

func TestPathMapper_BarrelFilePath_Python(t *testing.T) {
	pm := NewPathMapper("/input", "/output", "py")

	tests := []struct {
		dir      string
		expected string
	}{
		{"", "__init__.py"},
		{".", "__init__.py"},
		{"events", filepath.Join("events", "__init__.py")},
		{"events/payloads", filepath.Join("events", "payloads", "__init__.py")},
	}

	for _, test := range tests {
		result := pm.BarrelFilePath(test.dir)
		assert.Equal(t, test.expected, result, "barrel file path should be correct for dir %s", test.dir)
	}
}

func TestPathMapper_BarrelFilePath_Go(t *testing.T) {
	pm := NewPathMapper("/input", "/output", "go")

	result := pm.BarrelFilePath("events")

	assert.Empty(t, result, "Go should not generate barrel files")
}

func TestPathMapper_InputPathToOutputPath_PythonHyphenSanitization(t *testing.T) {
	pm := NewPathMapper("/input", "/output", "py")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple hyphenated filename",
			input:    filepath.Join("/input", "user-profile.json"),
			expected: "user_profile.py",
		},
		{
			name:     "multiple hyphens",
			input:    filepath.Join("/input", "enum-complex.json"),
			expected: "enum_complex.py",
		},
		{
			name:     "nested path with hyphens",
			input:    filepath.Join("/input", "edge-cases", "special-props.json"),
			expected: filepath.Join("edge-cases", "special_props.py"),
		},
		{
			name:     "mixed underscores and hyphens",
			input:    filepath.Join("/input", "user_profile-data.json"),
			expected: "user_profile_data.py",
		},
		{
			name:     "no hyphens (should not change)",
			input:    filepath.Join("/input", "user.json"),
			expected: "user.py",
		},
		{
			name:     "already underscores (should not change)",
			input:    filepath.Join("/input", "user_profile.json"),
			expected: "user_profile.py",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := pm.InputPathToOutputPath(test.input)
			assert.Equal(t, test.expected, result, "output path should be correct")
		})
	}
}

func TestPathMapper_InputPathToOutputPath_TypeScriptNoSanitization(t *testing.T) {
	pm := NewPathMapper("/input", "/output", "ts")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "hyphens preserved in TypeScript",
			input:    filepath.Join("/input", "user-profile.json"),
			expected: "user-profile.ts",
		},
		{
			name:     "multiple hyphens preserved",
			input:    filepath.Join("/input", "enum-complex.json"),
			expected: "enum-complex.ts",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := pm.InputPathToOutputPath(test.input)
			assert.Equal(t, test.expected, result, "output path should be correct")
		})
	}
}

func TestGetDirectoryLevels(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{"user.ts", []string{}},
		{"events/event.ts", []string{"events"}},
		{"events/payloads/v1/subscribe.ts", []string{"events", "events/payloads", "events/payloads/v1"}},
		{"", []string{}},
		{".", []string{}},
	}

	for _, test := range tests {
		result := GetDirectoryLevels(test.path)

		assert.Len(t, result, len(test.expected), "path %s should have expected number of directory levels", test.path)

		for i, expected := range test.expected {
			// Normalize paths for comparison
			expectedNorm := filepath.FromSlash(expected)
			resultNorm := filepath.FromSlash(result[i])

			assert.Equal(t, expectedNorm, resultNorm, "directory level should match")
		}
	}
}
