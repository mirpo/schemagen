package generation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiskWriter_WriteFile(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		content []byte
	}{
		{"simple file", "file.txt", []byte("hello")},
		{"nested file", "a/b/c/file.txt", []byte("nested")},
		{"overwrite", "overwrite.txt", []byte("second")},
		{"empty content", "empty.txt", nil},
	}

	tmpDir := t.TempDir()
	writer := NewDiskWriter(tmpDir)

	require.NoError(t, writer.WriteFile("overwrite.txt", []byte("first")))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := writer.WriteFile(tt.path, tt.content)
			require.NoError(t, err)

			fullPath := filepath.Join(tmpDir, tt.path)
			require.FileExists(t, fullPath)

			data, err := os.ReadFile(fullPath)
			require.NoError(t, err)

			if len(tt.content) == 0 {
				assert.Empty(t, data)
			} else {
				assert.Equal(t, tt.content, data)
			}
		})
	}
}

func TestDiskWriter_MakeDirectory(t *testing.T) {
	tests := []string{
		"simple",
		"a/b/c",
		"already/existing",
	}

	tmpDir := t.TempDir()
	writer := NewDiskWriter(tmpDir)

	// create once to test idempotency
	require.NoError(t, writer.MakeDirectory("already/existing"))

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			err := writer.MakeDirectory(path)
			require.NoError(t, err)
			assert.DirExists(t, filepath.Join(tmpDir, path))
		})
	}
}

func TestDiskWriter_WriteFile_InvalidPath(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewDiskWriter(tmpDir)

	// create a file
	require.NoError(t, writer.WriteFile("blocker", []byte("x")))

	// attempt to write under a file
	err := writer.WriteFile("blocker/file.txt", []byte("fail"))
	assert.Error(t, err)
}
