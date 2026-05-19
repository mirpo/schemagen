package compare

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mirpo/schemagen/pkg/parse"
	"github.com/mirpo/schemagen/pkg/pipeline"
)

type Config struct {
	Input string
	Flags *pipeline.GenerationFlags
}

type FileStatus string

const (
	StatusModified FileStatus = "modified"
	StatusNew      FileStatus = "new"
	StatusDeleted  FileStatus = "deleted"
)

type FileDiff struct {
	Path       string
	Status     FileStatus
	OldContent string
	NewContent string
}

type Result struct {
	Diffs []FileDiff
}

func Run(cfg *Config) (*Result, error) {
	if cfg.Flags == nil {
		return nil, fmt.Errorf("generation flags are required")
	}

	schemas, err := parse.Load(cfg.Input)
	if err != nil {
		return nil, fmt.Errorf("loading schemas: %w", err)
	}

	var diffs []FileDiff
	for _, t := range pipeline.BuildTargets(cfg.Flags) {
		w := pipeline.NewMemoryWriter()
		pipeCfg := pipeline.ConfigFromFlags(cfg.Flags, schemas, t.Dir, t.Lang)
		pipeCfg.Writer = w
		if err := pipeline.Run(pipeCfg); err != nil {
			return nil, err
		}

		langDiffs, err := compareFiles(w.Files, t.Dir)
		if err != nil {
			return nil, err
		}
		diffs = append(diffs, langDiffs...)
	}

	return &Result{Diffs: diffs}, nil
}

func compareFiles(generated map[string][]byte, existingDir string) ([]FileDiff, error) {
	var diffs []FileDiff

	for path, genBytes := range generated {
		genContent := normalizeLineEndings(string(genBytes))

		existPath := filepath.Join(existingDir, path)
		existBytes, err := os.ReadFile(existPath)
		if os.IsNotExist(err) {
			diffs = append(diffs, FileDiff{Path: path, Status: StatusNew, NewContent: genContent})
			continue
		}
		if err != nil {
			return nil, err
		}

		existContent := normalizeLineEndings(string(existBytes))
		if genContent != existContent {
			diffs = append(diffs, FileDiff{Path: path, Status: StatusModified, OldContent: existContent, NewContent: genContent})
		}
	}

	if info, err := os.Stat(existingDir); err == nil && info.IsDir() {
		err := filepath.Walk(existingDir, func(existPath string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			rel, relErr := filepath.Rel(existingDir, existPath)
			if relErr != nil {
				return relErr
			}
			if _, exists := generated[rel]; !exists {
				content, readErr := os.ReadFile(existPath)
				if readErr != nil {
					return readErr
				}
				diffs = append(diffs, FileDiff{Path: rel, Status: StatusDeleted, OldContent: normalizeLineEndings(string(content))})
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return diffs, nil
}

func normalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}
