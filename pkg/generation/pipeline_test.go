package generation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
 Helpers
*/

func createTestCompiler(t *testing.T) *jsonschema.Compiler {
	t.Helper()

	compiler := jsonschema.NewCompiler()
	schemaJSON := []byte(`{
		"type": "object",
		"title": "User",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name"]
	}`)

	_, err := compiler.Compile(schemaJSON, "test.json")
	require.NoError(t, err)

	return compiler
}

func createTestSchema(t *testing.T) *schema.Schema {
	t.Helper()

	compiler := createTestCompiler(t)
	compiled, err := compiler.Compile([]byte(`{
		"type": "object",
		"title": "User",
		"properties": {
			"id": {"type": "string"}
		}
	}`), "test.json")
	require.NoError(t, err)

	return &schema.Schema{
		Name:         "User",
		Path:         "test.json",
		RelativePath: "test.json",
		Compiled:     compiled,
	}
}

/*
 validateConfig
*/

func TestValidateConfig(t *testing.T) {
	validCompiler := createTestCompiler(t)
	validSchema := createTestSchema(t)

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{"nil config", nil, true},
		{"no schemas", &Config{}, true},
		{
			"missing compiler",
			&Config{
				Schemas:  []*schema.Schema{validSchema},
				OutDir:   "/tmp",
				Language: LanguageTypeScript,
			},
			true,
		},
		{
			"missing outdir",
			&Config{
				Schemas:  []*schema.Schema{validSchema},
				Compiler: validCompiler,
				Language: LanguageTypeScript,
			},
			true,
		},
		{
			"missing language",
			&Config{
				Schemas:  []*schema.Schema{validSchema},
				Compiler: validCompiler,
				OutDir:   "/tmp",
			},
			true,
		},
		{
			"unsupported language",
			&Config{
				Schemas:  []*schema.Schema{validSchema},
				Compiler: validCompiler,
				OutDir:   "/tmp",
				Language: "java",
			},
			true,
		},
		{
			"valid typescript",
			&Config{
				Schemas:  []*schema.Schema{validSchema},
				Compiler: validCompiler,
				OutDir:   "/tmp",
				Language: LanguageTypeScript,
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	t.Run("sets output strategy", func(t *testing.T) {
		cfg := &Config{Language: LanguageTypeScript}
		applyDefaults(cfg)
		assert.Equal(t, output.StrategyBundle, cfg.OutputStrategy)
	})

	t.Run("does not override existing strategy", func(t *testing.T) {
		cfg := &Config{Language: LanguageTypeScript, OutputStrategy: output.StrategyMultiFile}
		applyDefaults(cfg)
		assert.Equal(t, output.StrategyMultiFile, cfg.OutputStrategy)
	})

	t.Run("initializes TypeScript config", func(t *testing.T) {
		cfg := &Config{Language: LanguageTypeScript}
		applyDefaults(cfg)
		assert.NotNil(t, cfg.TypeScript)
	})

	t.Run("initializes Go config with defaults", func(t *testing.T) {
		cfg := &Config{Language: LanguageGo}
		applyDefaults(cfg)
		require.NotNil(t, cfg.Go)
		assert.Equal(t, "models", cfg.Go.PackageName)
		assert.True(t, cfg.Go.UsePointers)
		assert.True(t, cfg.Go.OmitEmpty)
	})

	t.Run("forces ExtractInline for Go", func(t *testing.T) {
		cfg := &Config{Language: LanguageGo, ExtractInline: false}
		applyDefaults(cfg)
		assert.True(t, cfg.ExtractInline)
	})

	t.Run("forces ExtractInline for Python", func(t *testing.T) {
		cfg := &Config{Language: LanguagePython, ExtractInline: false}
		applyDefaults(cfg)
		assert.True(t, cfg.ExtractInline)
	})

	t.Run("preserves ExtractInline for TypeScript", func(t *testing.T) {
		cfg := &Config{Language: LanguageTypeScript, ExtractInline: false}
		applyDefaults(cfg)
		assert.False(t, cfg.ExtractInline)
	})

	t.Run("does not override existing Go config", func(t *testing.T) {
		cfg := &Config{Language: LanguageGo, Go: &GoConfig{PackageName: "custom"}}
		applyDefaults(cfg)
		assert.Equal(t, "custom", cfg.Go.PackageName)
	})
}

func TestBuildTargets(t *testing.T) {
	flags := &GenerationFlags{
		OutTS: "/tmp/ts",
		OutGo: "/tmp/go",
	}

	targets := BuildTargets(flags)
	assert.Len(t, targets, 2)
	assert.Equal(t, "/tmp/ts", targets[0].Dir)
	assert.Equal(t, LanguageTypeScript, targets[0].Lang)
	assert.Equal(t, "/tmp/go", targets[1].Dir)
	assert.Equal(t, LanguageGo, targets[1].Lang)
}

func TestBuildTargets_Empty(t *testing.T) {
	flags := &GenerationFlags{}
	targets := BuildTargets(flags)
	assert.Empty(t, targets)
}

/*
 Run – happy paths
*/

func TestRun_SupportedLanguages(t *testing.T) {
	tests := []struct {
		name     string
		language Language
		expected string
	}{
		{"typescript", LanguageTypeScript, "types.ts"},
		{"python", LanguagePython, "types.py"},
		{"go", LanguageGo, "types.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outDir := t.TempDir()

			cfg := &Config{
				Schemas:        []*schema.Schema{createTestSchema(t)},
				Compiler:       createTestCompiler(t),
				OutDir:         outDir,
				Language:       tt.language,
				OutputStrategy: output.StrategyBundle,
			}

			require.NoError(t, Run(cfg))
			assert.FileExists(t, filepath.Join(outDir, tt.expected))
		})
	}
}

/*
 Run – multifile strategy
*/

func TestRun_MultiFileStrategy(t *testing.T) {
	outDir := t.TempDir()

	cfg := &Config{
		Schemas:        []*schema.Schema{createTestSchema(t)},
		Compiler:       createTestCompiler(t),
		OutDir:         outDir,
		Language:       LanguageTypeScript,
		OutputStrategy: output.StrategyMultiFile,
	}

	require.NoError(t, Run(cfg))
	assert.FileExists(t, filepath.Join(outDir, "index.ts"))
}

/*
 Run – headers
*/

func TestRun_DisableHeaders(t *testing.T) {
	outDir := t.TempDir()

	cfg := &Config{
		Schemas:        []*schema.Schema{createTestSchema(t)},
		Compiler:       createTestCompiler(t),
		OutDir:         outDir,
		Language:       LanguageTypeScript,
		OutputStrategy: output.StrategyBundle,
		DisableHeaders: true,
	}

	require.NoError(t, Run(cfg))

	data, err := os.ReadFile(filepath.Join(outDir, "types.ts"))
	require.NoError(t, err)

	assert.NotContains(t, string(data), "Code generated by")
}

/*
 Run – Go always extracts inline
*/

func TestRun_GoAlwaysExtractsInline(t *testing.T) {
	outDir := t.TempDir()

	cfg := &Config{
		Schemas:        []*schema.Schema{createTestSchema(t)},
		Compiler:       createTestCompiler(t),
		OutDir:         outDir,
		Language:       LanguageGo,
		OutputStrategy: output.StrategyBundle,
		ExtractInline:  false, // ignored for Go
		Go: &GoConfig{
			PackageName: "models",
		},
	}

	require.NoError(t, Run(cfg))

	data, err := os.ReadFile(filepath.Join(outDir, "types.go"))
	require.NoError(t, err)

	assert.NotContains(t, string(data), "map[string]interface{}")
}

