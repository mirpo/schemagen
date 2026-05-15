package compare

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mirpo/schemagen/pkg/constants"
	"github.com/mirpo/schemagen/pkg/generation"
	"github.com/mirpo/schemagen/pkg/schema"
)

// Config contains configuration for compare operations.
type Config struct {
	Input   string
	Schemas []*schema.Schema
	Loader  *schema.Loader
	Flags   *generation.GenerationFlags
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

	if err := generateAll(tmpDir, schemas, loader, cfg.Flags); err != nil {
		return nil, err
	}

	diffs, err := compareDirectories(tmpDir, cfg.Flags)
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
	flags *generation.GenerationFlags,
) error {
	targets := []struct {
		outDir   string
		shortDir string
		lang     generation.Language
	}{
		{flags.OutTS, constants.LanguageTypeScriptShort, generation.LanguageTypeScript},
		{flags.OutPY, constants.LanguagePythonShort, generation.LanguagePython},
		{flags.OutGo, constants.LanguageGoShort, generation.LanguageGo},
	}

	for _, t := range targets {
		if t.outDir == "" {
			continue
		}
		cfg := generation.ConfigFromFlags(flags, schemas, loader.Compiler(),
			filepath.Join(tmpDir, t.shortDir), t.lang)
		if err := generation.Run(cfg); err != nil {
			return fmt.Errorf("%s generation failed: %w", t.lang, err)
		}
	}

	return nil
}

// compareDirectories compares generated and existing directories.
func compareDirectories(generatedRoot string, flags *generation.GenerationFlags) ([]FileDiff, error) {
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

	if _, err := os.Stat(existing); err != nil {
		if os.IsNotExist(err) {
			return walkGeneratedFiles(generated, "", readFile)
		}
		return nil, err
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

		if _, statErr := os.Stat(filepath.Join(generated, rel)); statErr != nil {
			if !os.IsNotExist(statErr) {
				return statErr
			}
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
