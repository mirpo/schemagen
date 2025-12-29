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

// Config contains configuration for compare operations
type Config struct {
	Input       string
	Schemas     []*schema.Schema        // Pre-loaded schemas (optional, will load from Input if nil)
	Loader      *schema.Loader          // Pre-initialized loader (optional, will create if nil)
	Flags       *config.GenerationFlags // Generation flags
	ExistingDir string                  // Directory to compare against
}

// normalizeOutputStrategy converts CLI string input to output.OutputStrategy constant
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
		// Default to bundle if unrecognized
		return output.StrategyBundle
	}
}

// FileStatus represents the status of a file comparison
type FileStatus string

const (
	StatusModified FileStatus = "modified"
	StatusNew      FileStatus = "new"
	StatusDeleted  FileStatus = "deleted"
	StatusSame     FileStatus = "same"
)

// FileDiff represents a difference between generated and existing files
type FileDiff struct {
	Path       string
	Status     FileStatus
	OldContent string
	NewContent string
}

// Result contains the results of a comparison operation
type Result struct {
	HasDrift bool
	Diffs    []FileDiff
}

// Run generates code to a temporary directory and compares with existing directory
func Run(cfg *Config) (*Result, error) {
	// Load schemas if not provided
	schemas := cfg.Schemas
	loader := cfg.Loader
	if schemas == nil {
		if loader == nil {
			loader = schema.NewLoader()
		}
		var err error
		schemas, err = loader.Load(cfg.Input)
		if err != nil {
			return nil, fmt.Errorf("loading schemas: %w", err)
		}
	}
	if loader == nil {
		loader = schema.NewLoader()
	}

	// Generate to temp directory
	tmpDir, err := os.MkdirTemp("", "schemagen-compare-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate TypeScript if requested
	if cfg.Flags.OutTS != "" {
		outDir := filepath.Join(tmpDir, "ts")
		genCfg := &generation.Config{
			Schemas:          schemas,
			Compiler:         loader.Compiler(),
			OutDir:           outDir,
			Language:         generation.LanguageTypeScript,
			ExtractInline:    cfg.Flags.ExtractInline,
			DisableHeaders:   cfg.Flags.DisableHeaders,
			DisableTimestamp: cfg.Flags.DisableTimestamp,
			OutputStrategy:   normalizeOutputStrategy(cfg.Flags.OutputStrategy),
			TypeScript: &generation.TypeScriptConfig{
				UnknownAny:           cfg.Flags.TSUnknownAny,
				AdditionalProperties: cfg.Flags.TSAdditionalProperties,
			},
		}
		if err := generation.Run(genCfg); err != nil {
			return nil, fmt.Errorf("TypeScript generation failed: %w", err)
		}
	}

	// Generate Python if requested
	if cfg.Flags.OutPY != "" {
		outDir := filepath.Join(tmpDir, "py")
		genCfg := &generation.Config{
			Schemas:          schemas,
			Compiler:         loader.Compiler(),
			OutDir:           outDir,
			Language:         generation.LanguagePython,
			ExtractInline:    true, // Python always extracts
			DisableHeaders:   cfg.Flags.DisableHeaders,
			DisableTimestamp: cfg.Flags.DisableTimestamp,
			OutputStrategy:   normalizeOutputStrategy(cfg.Flags.OutputStrategy),
			Python: &generation.PythonConfig{
				SnakeCaseField: cfg.Flags.PySnakeCaseField,
			},
		}
		if err := generation.Run(genCfg); err != nil {
			return nil, fmt.Errorf("python generation failed: %w", err)
		}
	}

	// Generate Go if requested
	if cfg.Flags.OutGo != "" {
		outDir := filepath.Join(tmpDir, "go")
		genCfg := &generation.Config{
			Schemas:          schemas,
			Compiler:         loader.Compiler(),
			OutDir:           outDir,
			Language:         generation.LanguageGo,
			ExtractInline:    cfg.Flags.ExtractInline,
			DisableHeaders:   cfg.Flags.DisableHeaders,
			DisableTimestamp: cfg.Flags.DisableTimestamp,
			OutputStrategy:   normalizeOutputStrategy(cfg.Flags.OutputStrategy),
			Go: &generation.GoConfig{
				PackageName: cfg.Flags.GoPackageName,
				UsePointers: cfg.Flags.GoUsePointers,
				OmitEmpty:   cfg.Flags.GoOmitEmpty,
				ModulePath:  cfg.Flags.GoModulePath,
			},
		}
		if err := generation.Run(genCfg); err != nil {
			return nil, fmt.Errorf("go generation failed: %w", err)
		}
	}

	// Compare directories
	diffs, err := compareDirectories(tmpDir, cfg.ExistingDir, cfg.Flags)
	if err != nil {
		return nil, fmt.Errorf("comparing directories: %w", err)
	}

	return &Result{
		HasDrift: len(diffs) > 0,
		Diffs:    diffs,
	}, nil
}

// compareDirectories compares generated and existing directories
func compareDirectories(generatedRoot, existingRoot string, flags *config.GenerationFlags) ([]FileDiff, error) {
	var diffs []FileDiff

	// Compare each language directory
	if flags.OutTS != "" {
		langDiffs, err := compareLangDir(
			filepath.Join(generatedRoot, "ts"),
			flags.OutTS,
		)
		if err != nil {
			return nil, err
		}
		diffs = append(diffs, langDiffs...)
	}

	if flags.OutPY != "" {
		langDiffs, err := compareLangDir(
			filepath.Join(generatedRoot, "py"),
			flags.OutPY,
		)
		if err != nil {
			return nil, err
		}
		diffs = append(diffs, langDiffs...)
	}

	if flags.OutGo != "" {
		langDiffs, err := compareLangDir(
			filepath.Join(generatedRoot, "go"),
			flags.OutGo,
		)
		if err != nil {
			return nil, err
		}
		diffs = append(diffs, langDiffs...)
	}

	return diffs, nil
}

// compareLangDir compares a single language directory
func compareLangDir(generated, existing string) ([]FileDiff, error) {
	var diffs []FileDiff

	readFile := func(p string) (string, error) {
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		return normalizeLineEndings(string(b)), nil
	}

	// If existing directory doesn't exist, all files are new
	if _, err := os.Stat(existing); os.IsNotExist(err) {
		return walkGeneratedFiles(generated, "", readFile)
	}

	// Compare generated files with existing
	generatedDiffs, err := walkGeneratedFiles(generated, existing, readFile)
	if err != nil {
		return nil, err
	}
	diffs = append(diffs, generatedDiffs...)

	// Find deleted files (in existing but not in generated)
	deletedDiffs, err := walkExistingFiles(generated, existing, readFile)
	if err != nil {
		return nil, err
	}
	diffs = append(diffs, deletedDiffs...)

	return diffs, nil
}

// walkGeneratedFiles walks the generated directory and compares files with existing directory.
// If existing is empty, all files are marked as new.
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

		// If no existing directory specified, all files are new
		if existing == "" {
			diffs = append(diffs, FileDiff{
				Path:       rel,
				Status:     StatusNew,
				NewContent: genContent,
			})
			return nil
		}

		// Compare with existing file
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

		// Compare file contents
		if diff := compareFiles(genContent, existContent, rel); diff != nil {
			diffs = append(diffs, *diff)
		}

		return nil
	})

	return diffs, err
}

// walkExistingFiles walks the existing directory to find deleted files
// (files that exist in existing but not in generated).
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
			c, err := readFile(existPath)
			if err != nil {
				return err
			}
			diffs = append(diffs, FileDiff{
				Path:       rel,
				Status:     StatusDeleted,
				OldContent: c,
			})
		}
		return nil
	})

	return diffs, err
}

// compareFiles compares two file contents and returns a FileDiff if they differ.
// Returns nil if files are identical.
func compareFiles(genContent, existContent, relPath string) *FileDiff {
	if genContent == existContent {
		return nil
	}

	return &FileDiff{
		Path:       relPath,
		Status:     StatusModified,
		OldContent: existContent,
		NewContent: genContent,
	}
}

// normalizeLineEndings normalizes CRLF to LF for cross-platform comparison
func normalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}
