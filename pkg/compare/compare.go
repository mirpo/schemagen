package compare

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mirpo/schemagen/pkg/config"
	"github.com/mirpo/schemagen/pkg/generation"
	"github.com/mirpo/schemagen/pkg/output"
	"github.com/mirpo/schemagen/pkg/schema"
)

// Config contains configuration for compare operations.
type Config struct {
	Input       string
	Schemas     []*schema.Schema
	Loader      *schema.Loader
	Flags       *config.GenerationFlags
	ExistingDir string
}

// normalizeOutputStrategy converts CLI string input to output.OutputStrategy.
func normalizeOutputStrategy(strategy string) output.OutputStrategy {
	switch strategy {
	case "bundle":
		return output.StrategyBundle
	case "multifile", "multi-file":
		return output.StrategyMultiFile
	case "bundledeps", "bundle-deps":
		return output.StrategyBundleDeps
	case "bundle-per-dir":
		return output.StrategyBundlePerDir
	default:
		return output.StrategyBundle
	}
}

// FileStatus represents the status of a file comparison.
type FileStatus string

const (
	StatusModified FileStatus = "modified"
	StatusNew      FileStatus = "new"
	StatusDeleted  FileStatus = "deleted"
)

// FileDiff represents a difference between generated and existing files.
type FileDiff struct {
	Path       string
	Status     FileStatus
	OldContent string
	NewContent string
}

// Result contains the results of a comparison operation.
type Result struct {
	HasDrift bool
	Diffs    []FileDiff
}

// Run generates code to a temporary directory and compares with existing directory.
func Run(cfg *Config) (*Result, error) {
	if cfg.Flags == nil {
		return nil, fmt.Errorf("generation flags are required")
	}

	loader := cfg.Loader
	if loader == nil {
		loader = schema.NewLoader()
	}

	schemas := cfg.Schemas
	if schemas == nil {
		var err error
		schemas, err = loader.Load(cfg.Input)
		if err != nil {
			return nil, fmt.Errorf("loading schemas: %w", err)
		}
	}

	tmpDir, err := os.MkdirTemp("", "schemagen-compare-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	strategy := normalizeOutputStrategy(cfg.Flags.OutputStrategy)

	if err := generateAll(tmpDir, schemas, loader, cfg.Flags, strategy); err != nil {
		return nil, err
	}

	diffs, err := compareDirectories(tmpDir, cfg.ExistingDir, cfg.Flags)
	if err != nil {
		return nil, fmt.Errorf("comparing directories: %w", err)
	}

	return &Result{
		HasDrift: len(diffs) > 0,
		Diffs:    diffs,
	}, nil
}

func generateAll(
	tmpDir string,
	schemas []*schema.Schema,
	loader *schema.Loader,
	flags *config.GenerationFlags,
	strategy output.OutputStrategy,
) error {
	if flags.OutTS != "" {
		if err := generation.Run(&generation.Config{
			Schemas:          schemas,
			Compiler:         loader.Compiler(),
			OutDir:           filepath.Join(tmpDir, "ts"),
			Language:         generation.LanguageTypeScript,
			ExtractInline:    flags.ExtractInline,
			DisableHeaders:   flags.DisableHeaders,
			DisableTimestamp: flags.DisableTimestamp,
			OutputStrategy:   strategy,
			TypeScript: &generation.TypeScriptConfig{
				UnknownAny:           flags.TSUnknownAny,
				AdditionalProperties: flags.TSAdditionalProperties,
			},
		}); err != nil {
			return fmt.Errorf("typescript generation failed: %w", err)
		}
	}

	if flags.OutPY != "" {
		if err := generation.Run(&generation.Config{
			Schemas:          schemas,
			Compiler:         loader.Compiler(),
			OutDir:           filepath.Join(tmpDir, "py"),
			Language:         generation.LanguagePython,
			ExtractInline:    true,
			DisableHeaders:   flags.DisableHeaders,
			DisableTimestamp: flags.DisableTimestamp,
			OutputStrategy:   strategy,
			Python: &generation.PythonConfig{
				SnakeCaseField: flags.PySnakeCaseField,
			},
		}); err != nil {
			return fmt.Errorf("python generation failed: %w", err)
		}
	}

	if flags.OutGo != "" {
		if err := generation.Run(&generation.Config{
			Schemas:          schemas,
			Compiler:         loader.Compiler(),
			OutDir:           filepath.Join(tmpDir, "go"),
			Language:         generation.LanguageGo,
			ExtractInline:    flags.ExtractInline,
			DisableHeaders:   flags.DisableHeaders,
			DisableTimestamp: flags.DisableTimestamp,
			OutputStrategy:   strategy,
			Go: &generation.GoConfig{
				PackageName: flags.GoPackageName,
				UsePointers: flags.GoUsePointers,
				OmitEmpty:   flags.GoOmitEmpty,
				ModulePath:  flags.GoModulePath,
			},
		}); err != nil {
			return fmt.Errorf("go generation failed: %w", err)
		}
	}

	return nil
}

// compareDirectories compares generated and existing directories.
func compareDirectories(generatedRoot, existingRoot string, flags *config.GenerationFlags) ([]FileDiff, error) {
	var diffs []FileDiff

	type lang struct {
		outFlag string
		dir     string
	}

	langs := []lang{
		{flags.OutTS, "ts"},
		{flags.OutPY, "py"},
		{flags.OutGo, "go"},
	}

	for _, l := range langs {
		if l.outFlag == "" {
			continue
		}
		langDiffs, err := compareLangDir(
			filepath.Join(generatedRoot, l.dir),
			l.outFlag,
		)
		if err != nil {
			return nil, err
		}
		diffs = append(diffs, langDiffs...)
	}

	return diffs, nil
}

// compareLangDir compares a single language directory.
func compareLangDir(generated, existing string) ([]FileDiff, error) {
	readFile := func(p string) (string, error) {
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		return normalizeLineEndings(string(b)), nil
	}

	if _, err := os.Stat(existing); os.IsNotExist(err) {
		return walkGeneratedFiles(generated, "", readFile)
	}

	var diffs []FileDiff

	genDiffs, err := walkGeneratedFiles(generated, existing, readFile)
	if err != nil {
		return nil, err
	}
	diffs = append(diffs, genDiffs...)

	delDiffs, err := walkExistingFiles(generated, existing, readFile)
	if err != nil {
		return nil, err
	}
	diffs = append(diffs, delDiffs...)

	return diffs, nil
}

func walkGeneratedFiles(generated, existing string, readFile func(string) (string, error)) ([]FileDiff, error) {
	var diffs []FileDiff

	err := filepath.Walk(generated, func(genPath string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		rel, err := filepath.Rel(generated, genPath)
		if err != nil {
			return err
		}

		genContent, err := readFile(genPath)
		if err != nil {
			return err
		}

		if existing == "" {
			diffs = append(diffs, FileDiff{
				Path:       rel,
				Status:     StatusNew,
				NewContent: genContent,
			})
			return nil
		}

		existPath := filepath.Join(existing, rel)
		existContent, err := readFile(existPath)
		if os.IsNotExist(err) {
			diffs = append(diffs, FileDiff{
				Path:       rel,
				Status:     StatusNew,
				NewContent: genContent,
			})
			return nil
		}
		if err != nil {
			return err
		}

		if genContent != existContent {
			diffs = append(diffs, FileDiff{
				Path:       rel,
				Status:     StatusModified,
				OldContent: existContent,
				NewContent: genContent,
			})
		}

		return nil
	})

	return diffs, err
}

func walkExistingFiles(generated, existing string, readFile func(string) (string, error)) ([]FileDiff, error) {
	var diffs []FileDiff

	err := filepath.Walk(existing, func(existPath string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		rel, err := filepath.Rel(existing, existPath)
		if err != nil {
			return err
		}

		if _, err := os.Stat(filepath.Join(generated, rel)); os.IsNotExist(err) {
			content, err := readFile(existPath)
			if err != nil {
				return err
			}
			diffs = append(diffs, FileDiff{
				Path:       rel,
				Status:     StatusDeleted,
				OldContent: content,
			})
		}
		return nil
	})

	return diffs, err
}

// normalizeLineEndings normalizes CRLF to LF for cross-platform comparison.
func normalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}
