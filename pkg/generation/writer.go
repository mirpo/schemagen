package generation

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileWriter is an interface for writing files to disk
// This abstraction allows for easier testing with mock implementations
type FileWriter interface {
	// WriteFile writes content to a file at the given path
	// The path is relative to the base directory
	WriteFile(path string, content []byte) error

	// MakeDirectory creates a directory and any necessary parent directories
	// The path is relative to the base directory
	MakeDirectory(path string) error
}

// DiskWriter implements FileWriter by writing directly to the filesystem
type DiskWriter struct {
	baseDir string
}

// NewDiskWriter creates a new DiskWriter with the given base directory
func NewDiskWriter(baseDir string) *DiskWriter {
	return &DiskWriter{
		baseDir: baseDir,
	}
}

// WriteFile writes content to a file at the given path
// It creates any necessary parent directories before writing
func (w *DiskWriter) WriteFile(path string, content []byte) error {
	fullPath := filepath.Join(w.baseDir, path)

	// Create parent directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", path, err)
	}

	// Write file
	if err := os.WriteFile(fullPath, content, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// MakeDirectory creates a directory and any necessary parent directories
func (w *DiskWriter) MakeDirectory(path string) error {
	fullPath := filepath.Join(w.baseDir, path)

	if err := os.MkdirAll(fullPath, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", path, err)
	}

	return nil
}
