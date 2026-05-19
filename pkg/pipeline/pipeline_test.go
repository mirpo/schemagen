package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testdataDir() string {
	return filepath.Join("..", "..", "testdata")
}

func loadSchemas(t *testing.T, dir string) []*parse.NamedSchema {
	t.Helper()
	schemas, err := parse.ParseDir(filepath.Join(testdataDir(), "schemas", dir))
	require.NoError(t, err)
	require.NotEmpty(t, schemas)
	return schemas
}

func TestValidateConfig(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		assert.Error(t, Run(nil))
	})

	t.Run("no schemas", func(t *testing.T) {
		assert.Error(t, Run(&Config{
			OutDir:   "/tmp",
			Language: LanguageTypeScript,
		}))
	})

	t.Run("no outdir", func(t *testing.T) {
		assert.Error(t, Run(&Config{
			Schemas:  []*parse.NamedSchema{{}},
			Language: LanguageTypeScript,
		}))
	})

	t.Run("no language", func(t *testing.T) {
		assert.Error(t, Run(&Config{
			Schemas: []*parse.NamedSchema{{}},
			OutDir:  "/tmp",
		}))
	})

	t.Run("unsupported language", func(t *testing.T) {
		assert.Error(t, Run(&Config{
			Schemas:  []*parse.NamedSchema{{}},
			OutDir:   "/tmp",
			Language: "ruby",
		}))
	})
}

func TestConfigFromFlags(t *testing.T) {
	schemas := []*parse.NamedSchema{{Name: "test"}}

	t.Run("TypeScript config", func(t *testing.T) {
		flags := &GenerationFlags{
			ExtractInline:          true,
			DisableHeaders:         true,
			DisableTimestamp:       true,
			OutputStrategy:         output.StrategyMultiFile,
			TSUnknownAny:           true,
			TSAdditionalProperties: true,
			TSZod:                  true,
			TSZodOnly:              true,
			TSZodCoerceDates:       true,
			TSZodStrict:            true,
		}
		cfg := ConfigFromFlags(flags, schemas, "/out/ts", LanguageTypeScript)

		assert.Equal(t, schemas, cfg.Schemas)
		assert.Equal(t, "/out/ts", cfg.OutDir)
		assert.Equal(t, LanguageTypeScript, cfg.Language)
		assert.True(t, cfg.ExtractInline)
		assert.True(t, cfg.DisableHeaders)
		assert.True(t, cfg.DisableTimestamp)
		assert.Equal(t, output.StrategyMultiFile, cfg.OutputStrategy)

		require.NotNil(t, cfg.TypeScript)
		assert.True(t, cfg.TypeScript.UnknownAny)
		assert.True(t, cfg.TypeScript.AdditionalProperties)
		assert.True(t, cfg.TypeScript.Zod)
		assert.True(t, cfg.TypeScript.ZodOnly)
		assert.True(t, cfg.TypeScript.ZodCoerceDates)
		assert.True(t, cfg.TypeScript.ZodStrict)

		assert.Nil(t, cfg.Python)
		assert.Nil(t, cfg.Go)
	})

	t.Run("Python config", func(t *testing.T) {
		flags := &GenerationFlags{
			PySnakeCaseField:       true,
			PyAdditionalProperties: true,
		}
		cfg := ConfigFromFlags(flags, schemas, "/out/py", LanguagePython)

		require.NotNil(t, cfg.Python)
		assert.True(t, cfg.Python.SnakeCaseField)
		assert.True(t, cfg.Python.AdditionalProperties)

		assert.Nil(t, cfg.TypeScript)
		assert.Nil(t, cfg.Go)
	})

	t.Run("Go config", func(t *testing.T) {
		flags := &GenerationFlags{
			GoPackageName: "types",
			GoUsePointers: true,
			GoOmitEmpty:   true,
			GoModulePath:  "github.com/example/pkg",
		}
		cfg := ConfigFromFlags(flags, schemas, "/out/go", LanguageGo)

		require.NotNil(t, cfg.Go)
		assert.Equal(t, "types", cfg.Go.PackageName)
		assert.True(t, cfg.Go.UsePointers)
		assert.True(t, cfg.Go.OmitEmpty)
		assert.Equal(t, "github.com/example/pkg", cfg.Go.ModulePath)

		assert.Nil(t, cfg.TypeScript)
		assert.Nil(t, cfg.Python)
	})
}

func TestApplyDefaults(t *testing.T) {
	t.Run("python forces extract inline", func(t *testing.T) {
		cfg := &Config{Language: LanguagePython}
		applyDefaults(cfg)
		assert.True(t, cfg.ExtractInline)
		assert.NotNil(t, cfg.Python)
	})

	t.Run("go forces extract inline and defaults", func(t *testing.T) {
		cfg := &Config{Language: LanguageGo}
		applyDefaults(cfg)
		assert.True(t, cfg.ExtractInline)
		assert.NotNil(t, cfg.Go)
		assert.Equal(t, "models", cfg.Go.PackageName)
		assert.True(t, cfg.Go.UsePointers)
		assert.True(t, cfg.Go.OmitEmpty)
	})

	t.Run("typescript defaults", func(t *testing.T) {
		cfg := &Config{Language: LanguageTypeScript}
		applyDefaults(cfg)
		assert.NotNil(t, cfg.TypeScript)
	})

	t.Run("default strategy is bundle", func(t *testing.T) {
		cfg := &Config{Language: LanguageTypeScript}
		applyDefaults(cfg)
		assert.Equal(t, output.StrategyBundle, cfg.OutputStrategy)
	})
}

func TestBuildTargets(t *testing.T) {
	flags := &GenerationFlags{
		OutTS: "/tmp/ts",
		OutGo: "/tmp/go",
	}
	targets := BuildTargets(flags)
	assert.Len(t, targets, 2)
	assert.Equal(t, LanguageTypeScript, targets[0].Lang)
	assert.Equal(t, LanguageGo, targets[1].Lang)
}

func TestRun_TypeScript(t *testing.T) {
	schemas := loadSchemas(t, "foundation")
	outDir := t.TempDir()

	err := Run(&Config{
		Schemas:          schemas,
		OutDir:           outDir,
		Language:         LanguageTypeScript,
		DisableTimestamp: true,
		OutputStrategy:   output.StrategyMultiFile,
	})
	require.NoError(t, err)

	files, _ := filepath.Glob(filepath.Join(outDir, "*.ts"))
	assert.NotEmpty(t, files, "should generate .ts files")
}

func TestRun_Python(t *testing.T) {
	schemas := loadSchemas(t, "foundation")
	outDir := t.TempDir()

	err := Run(&Config{
		Schemas:          schemas,
		OutDir:           outDir,
		Language:         LanguagePython,
		DisableTimestamp: true,
		OutputStrategy:   output.StrategyMultiFile,
	})
	require.NoError(t, err)

	files, _ := filepath.Glob(filepath.Join(outDir, "*.py"))
	assert.NotEmpty(t, files, "should generate .py files")
}

func TestRun_Go(t *testing.T) {
	schemas := loadSchemas(t, "foundation")
	outDir := t.TempDir()

	err := Run(&Config{
		Schemas:          schemas,
		OutDir:           outDir,
		Language:         LanguageGo,
		DisableTimestamp: true,
		OutputStrategy:   output.StrategyMultiFile,
		Go: &GoConfig{
			PackageName: "models",
			UsePointers: true,
			OmitEmpty:   true,
		},
	})
	require.NoError(t, err)

	files, _ := filepath.Glob(filepath.Join(outDir, "*.go"))
	assert.NotEmpty(t, files, "should generate .go files")
}

func TestRun_MultiFile(t *testing.T) {
	schemas := loadSchemas(t, "complex")
	outDir := t.TempDir()

	err := Run(&Config{
		Schemas:          schemas,
		OutDir:           outDir,
		Language:         LanguageTypeScript,
		DisableTimestamp: true,
		OutputStrategy:   output.StrategyMultiFile,
	})
	require.NoError(t, err)

	entries, _ := os.ReadDir(outDir)
	assert.NotEmpty(t, entries, "should create output files/dirs")
}

func TestRun_Bundle(t *testing.T) {
	schemas := loadSchemas(t, "basic")
	outDir := t.TempDir()

	err := Run(&Config{
		Schemas:          schemas,
		OutDir:           outDir,
		Language:         LanguageTypeScript,
		DisableTimestamp: true,
		OutputStrategy:   output.StrategyBundle,
	})
	require.NoError(t, err)

	files, _ := filepath.Glob(filepath.Join(outDir, "*.ts"))
	assert.NotEmpty(t, files)
}

func TestRun_AllSchemaCategories(t *testing.T) {
	categories := []string{
		"basic", "complex", "edge-cases", "allof", "anyof",
		"refs", "foundation", "yaml", "events", "messaging_api", "extraction",
	}

	for _, cat := range categories {
		t.Run(cat, func(t *testing.T) {
			schemas := loadSchemas(t, cat)
			outDir := t.TempDir()

			err := Run(&Config{
				Schemas:          schemas,
				OutDir:           outDir,
				Language:         LanguageTypeScript,
				DisableTimestamp: true,
				OutputStrategy:   output.StrategyMultiFile,
			})
			require.NoError(t, err, "generation failed for %s", cat)

			files := collectOutputFiles(t, outDir)
			assert.NotEmpty(t, files, "no files generated for %s", cat)
		})
	}
}

func TestRun_AllLanguages_AllCategories(t *testing.T) {
	categories := []string{
		"basic", "complex", "edge-cases", "allof", "anyof",
		"refs", "foundation", "yaml",
	}

	languages := []Language{LanguageTypeScript, LanguagePython, LanguageGo}

	for _, lang := range languages {
		for _, cat := range categories {
			name := string(lang) + "/" + cat
			t.Run(name, func(t *testing.T) {
				schemas := loadSchemas(t, cat)
				outDir := t.TempDir()

				cfg := &Config{
					Schemas:          schemas,
					OutDir:           outDir,
					Language:         lang,
					DisableTimestamp: true,
					OutputStrategy:   output.StrategyMultiFile,
				}

				err := Run(cfg)
				require.NoError(t, err, "generation failed for %s", name)

				files := collectOutputFiles(t, outDir)
				assert.NotEmpty(t, files, "no files generated for %s", name)
			})
		}
	}
}

func collectOutputFiles(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(dir, path)
			if !strings.HasPrefix(rel, ".") {
				files = append(files, rel)
			}
		}
		return nil
	})
	return files
}
