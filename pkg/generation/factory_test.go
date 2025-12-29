package generation

import (
	"testing"

	"github.com/mirpo/schemagen/pkg/typegraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateGenerator_TypeScript tests creating a TypeScript generator
func TestCreateGenerator_TypeScript(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: LanguageTypeScript,
		TypeScript: &TypeScriptConfig{
			UnknownAny:           false,
			AdditionalProperties: false,
		},
	}

	gen, err := createGenerator(graph, cfg)

	require.NoError(t, err, "createGenerator() should not return error for TypeScript")
	require.NotNil(t, gen, "generator should not be nil")

	// Verify it's the correct type
	_, ok := gen.(*typeScriptGenerator)
	assert.True(t, ok, "generator should be typeScriptGenerator")
}

// TestCreateGenerator_TypeScript_WithConfig tests TypeScript generator with custom config
func TestCreateGenerator_TypeScript_WithConfig(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: LanguageTypeScript,
		TypeScript: &TypeScriptConfig{
			UnknownAny:           true,
			AdditionalProperties: true,
		},
		DisableHeaders:   true,
		DisableTimestamp: true,
	}

	gen, err := createGenerator(graph, cfg)

	require.NoError(t, err, "createGenerator() should not return error")
	require.NotNil(t, gen, "generator should not be nil")

	tsGen, ok := gen.(*typeScriptGenerator)
	require.True(t, ok, "generator should be typeScriptGenerator")
	assert.NotNil(t, tsGen.generator, "wrapped generator should not be nil")
}

// TestCreateGenerator_Python tests creating a Python generator
func TestCreateGenerator_Python(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: LanguagePython,
		Python: &PythonConfig{
			SnakeCaseField: false,
		},
	}

	gen, err := createGenerator(graph, cfg)

	require.NoError(t, err, "createGenerator() should not return error for Python")
	require.NotNil(t, gen, "generator should not be nil")

	// Verify it's the correct type
	_, ok := gen.(*pythonGenerator)
	assert.True(t, ok, "generator should be pythonGenerator")
}

// TestCreateGenerator_Python_WithConfig tests Python generator with custom config
func TestCreateGenerator_Python_WithConfig(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: LanguagePython,
		Python: &PythonConfig{
			SnakeCaseField: true,
		},
		DisableHeaders:   true,
		DisableTimestamp: false,
	}

	gen, err := createGenerator(graph, cfg)

	require.NoError(t, err, "createGenerator() should not return error")
	require.NotNil(t, gen, "generator should not be nil")

	pyGen, ok := gen.(*pythonGenerator)
	require.True(t, ok, "generator should be pythonGenerator")
	assert.NotNil(t, pyGen.generator, "wrapped generator should not be nil")
}

// TestCreateGenerator_Go tests creating a Go generator
func TestCreateGenerator_Go(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: LanguageGo,
		Go: &GoConfig{
			PackageName:   "test",
			UsePointers:   false,
			OmitEmpty:     false,
			PackagePrefix: "",
		},
	}

	gen, err := createGenerator(graph, cfg)

	require.NoError(t, err, "createGenerator() should not return error for Go")
	require.NotNil(t, gen, "generator should not be nil")

	// Verify it's the correct type
	_, ok := gen.(*goGenerator)
	assert.True(t, ok, "generator should be goGenerator")
}

// TestCreateGenerator_Go_WithConfig tests Go generator with custom config
func TestCreateGenerator_Go_WithConfig(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: LanguageGo,
		Go: &GoConfig{
			PackageName:   "mypackage",
			UsePointers:   true,
			OmitEmpty:     true,
			PackagePrefix: "github.com/example/",
		},
		DisableHeaders:   false,
		DisableTimestamp: true,
	}

	gen, err := createGenerator(graph, cfg)

	require.NoError(t, err, "createGenerator() should not return error")
	require.NotNil(t, gen, "generator should not be nil")

	goGen, ok := gen.(*goGenerator)
	require.True(t, ok, "generator should be goGenerator")
	assert.NotNil(t, goGen.generator, "wrapped generator should not be nil")
}

// TestCreateGenerator_Unknown tests creating generator with unknown language
func TestCreateGenerator_Unknown(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: "rust",
	}

	gen, err := createGenerator(graph, cfg)

	assert.Error(t, err, "createGenerator() should return error for unknown language")
	assert.Nil(t, gen, "generator should be nil on error")
	assert.Contains(t, err.Error(), "unsupported language", "error should mention unsupported language")
	assert.Contains(t, err.Error(), "rust", "error should include the language name")
}

// TestCreateGenerator_EmptyLanguage tests creating generator with empty language string
func TestCreateGenerator_EmptyLanguage(t *testing.T) {
	graph := &typegraph.Graph{}
	cfg := &Config{
		Language: "",
	}

	gen, err := createGenerator(graph, cfg)

	assert.Error(t, err, "createGenerator() should return error for empty language")
	assert.Nil(t, gen, "generator should be nil on error")
	assert.Contains(t, err.Error(), "unsupported language", "error should mention unsupported language")
}

// TestCreateGenerator_InvalidLanguage tests various invalid language values
func TestCreateGenerator_InvalidLanguage(t *testing.T) {
	tests := []struct {
		name     string
		language Language
	}{
		{"java", "java"},
		{"c++", "c++"},
		{"ruby", "ruby"},
		{"php", "php"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := &typegraph.Graph{}
			cfg := &Config{
				Language: tt.language,
			}

			gen, err := createGenerator(graph, cfg)

			assert.Error(t, err, "createGenerator() should return error for language: %s", tt.language)
			assert.Nil(t, gen, "generator should be nil on error")
			assert.Contains(t, err.Error(), "unsupported language", "error should mention unsupported language")
			assert.Contains(t, err.Error(), string(tt.language), "error should include the language name")
		})
	}
}

// TestCreateGenerator_NilGraph tests creating generator with nil graph
func TestCreateGenerator_NilGraph(t *testing.T) {
	tests := []struct {
		name     string
		language Language
		config   interface{}
	}{
		{
			name:     "typescript with nil graph",
			language: LanguageTypeScript,
			config: &TypeScriptConfig{
				UnknownAny:           false,
				AdditionalProperties: false,
			},
		},
		{
			name:     "python with nil graph",
			language: LanguagePython,
			config: &PythonConfig{
				SnakeCaseField: false,
			},
		},
		{
			name:     "go with nil graph",
			language: LanguageGo,
			config: &GoConfig{
				PackageName: "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Language: tt.language,
			}

			// Set language-specific config
			switch tt.language {
			case LanguageTypeScript:
				cfg.TypeScript = tt.config.(*TypeScriptConfig)
			case LanguagePython:
				cfg.Python = tt.config.(*PythonConfig)
			case LanguageGo:
				cfg.Go = tt.config.(*GoConfig)
			}

			// Nil graph should not cause panic
			gen, err := createGenerator(nil, cfg)

			require.NoError(t, err, "createGenerator() should not return error even with nil graph")
			assert.NotNil(t, gen, "generator should not be nil")
		})
	}
}
