package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
)

type FileWriter interface {
	WriteFile(path string, content []byte) error
}

type DiskWriter struct {
	baseDir string
}

func NewDiskWriter(baseDir string) *DiskWriter {
	return &DiskWriter{baseDir: baseDir}
}

func (w *DiskWriter) WriteFile(path string, content []byte) error {
	fullPath := filepath.Join(w.baseDir, path)

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", path, err)
	}

	if err := os.WriteFile(fullPath, content, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}
