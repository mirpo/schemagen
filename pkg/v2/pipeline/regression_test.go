package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mirpo/schemagen/pkg/v2/output"
	"github.com/mirpo/schemagen/pkg/v2/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoldenFileRegression(t *testing.T) {
	rootDir := filepath.Join("..", "..", "..", "testdata")
	schemasDir := filepath.Join(rootDir, "schemas")
	expectedDir := filepath.Join(rootDir, "expected")

	type testCase struct {
		name           string
		language       Language
		expectedSubdir string
		extraConfig    func(*Config)
	}

	tests := []testCase{
		{
			name:           "typescript/multifile",
			language:       LanguageTypeScript,
			expectedSubdir: "ts",
		},
		{
			name:           "typescript/extracted",
			language:       LanguageTypeScript,
			expectedSubdir: "ts-extracted",
			extraConfig: func(cfg *Config) {
				cfg.ExtractInline = true
			},
		},
		{
			name:           "python/multifile",
			language:       LanguagePython,
			expectedSubdir: "py",
		},
		{
			name:           "go/multifile",
			language:       LanguageGo,
			expectedSubdir: "go",
			extraConfig: func(cfg *Config) {
				cfg.Go = &GoConfig{
					PackageName: "models",
					UsePointers: true,
					OmitEmpty:   true,
					ModulePath:  "github.com/mirpo/schemagen/testdata/expected",
				}
			},
		},
	}

	type schemaCategory struct {
		name   string
		file   string
		subdir string
	}

	schemaCategories := []schemaCategory{
		{name: "basic"},
		{name: "complex", file: "ecommerce_order.json", subdir: "ecommerce_order"},
		{name: "edge-cases"},
		{name: "allof"},
		{name: "anyof"},
		{name: "refs", file: "organization.json", subdir: "organization"},
		{name: "foundation", file: "foundation.json"},
		{name: "yaml"},
		{name: "events"},
		{name: "messaging_api", file: "messaging_api.json"},
		{name: "extraction", file: "blog_post.json", subdir: "blog_post"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, cat := range schemaCategories {
				expectedCatDir := filepath.Join(expectedDir, tc.expectedSubdir, cat.name)
				if _, err := os.Stat(expectedCatDir); os.IsNotExist(err) {
					continue
				}

				t.Run(cat.name, func(t *testing.T) {
					var schemas []*parse.NamedSchema
					var err error
					var outDir string

					if cat.file != "" {
						ns, parseErr := parse.ParseFile(filepath.Join(schemasDir, cat.name, cat.file))
						require.NoError(t, parseErr)
						schemas = []*parse.NamedSchema{ns}

						if cat.subdir != "" {
							outDir = filepath.Join(t.TempDir(), cat.subdir)
							require.NoError(t, os.MkdirAll(outDir, 0o755))
						} else {
							outDir = t.TempDir()
						}
					} else {
						schemas, err = parse.ParseDir(filepath.Join(schemasDir, cat.name))
						require.NoError(t, err)
						outDir = t.TempDir()
					}
					require.NotEmpty(t, schemas)

					cfg := &Config{
						Schemas:          schemas,
						OutDir:           outDir,
						Language:         tc.language,
						DisableTimestamp: true,
						OutputStrategy:   output.StrategyMultiFile,
					}
					if tc.extraConfig != nil {
						tc.extraConfig(cfg)
					}

					err = Run(cfg)
					require.NoError(t, err, "generation failed for %s/%s", tc.name, cat.name)

					actualDir := outDir
					if cat.subdir != "" {
						actualDir = filepath.Dir(outDir)
					}
					compareDirectories(t, expectedCatDir, actualDir, tc.expectedSubdir+"/"+cat.name)
				})
			}
		})
	}
}

func compareDirectories(t *testing.T, expectedDir, actualDir, label string) {
	t.Helper()

	expectedFiles := collectFiles(t, expectedDir)
	actualFiles := collectFiles(t, actualDir)

	for relPath, expectedContent := range expectedFiles {
		actualContent, exists := actualFiles[relPath]
		if !assert.True(t, exists, "[%s] missing file: %s", label, relPath) {
			continue
		}

		expectedNorm := normalizeContent(expectedContent)
		actualNorm := normalizeContent(actualContent)

		if expectedNorm != actualNorm {
			t.Errorf("[%s] file %s differs\n--- expected ---\n%s\n--- actual ---\n%s",
				label, relPath, expectedContent, actualContent)
		}
	}

	for relPath := range actualFiles {
		if _, exists := expectedFiles[relPath]; !exists {
			t.Errorf("[%s] unexpected extra file: %s", label, relPath)
		}
	}
}

func collectFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := make(map[string]string)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		files[rel] = string(content)
		return nil
	})
	require.NoError(t, err)

	return files
}

func normalizeContent(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	var filtered []string
	for _, line := range lines {
		if strings.Contains(line, "Generated at") || strings.Contains(line, "generated at") {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}
