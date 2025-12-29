package generation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewDiskWriter tests creating a new DiskWriter
func TestNewDiskWriter(t *testing.T) {
	baseDir := "/tmp/test"
	writer := NewDiskWriter(baseDir)

	require.NotNil(t, writer, "NewDiskWriter should not return nil")
	assert.Equal(t, baseDir, writer.baseDir, "baseDir should match")
}

// TestDiskWriter_WriteFile tests writing a file
func TestDiskWriter_WriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewDiskWriter(tmpDir)

	content := []byte("test content")
	err := writer.WriteFile("test.txt", content)

	require.NoError(t, err, "WriteFile should not return error")

	// Verify file exists
	fullPath := filepath.Join(tmpDir, "test.txt")
	assert.FileExists(t, fullPath, "file should exist")

	// Verify content
	readContent, err := os.ReadFile(fullPath)
	require.NoError(t, err, "should read file")
	assert.Equal(t, content, readContent, "file content should match")
}

// TestDiskWriter_WriteFile_Nested tests writing a file in nested directories
func TestDiskWriter_WriteFile_Nested(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewDiskWriter(tmpDir)

	content := []byte("nested content")
	err := writer.WriteFile("a/b/c/test.txt", content)

	require.NoError(t, err, "WriteFile should not return error")

	// Verify file exists
	fullPath := filepath.Join(tmpDir, "a", "b", "c", "test.txt")
	assert.FileExists(t, fullPath, "nested file should exist")

	// Verify content
	readContent, err := os.ReadFile(fullPath)
	require.NoError(t, err, "should read file")
	assert.Equal(t, content, readContent, "file content should match")

	// Verify directories were created
	assert.DirExists(t, filepath.Join(tmpDir, "a"), "directory a should exist")
	assert.DirExists(t, filepath.Join(tmpDir, "a", "b"), "directory a/b should exist")
	assert.DirExists(t, filepath.Join(tmpDir, "a", "b", "c"), "directory a/b/c should exist")
}

// TestDiskWriter_WriteFile_Overwrite tests overwriting an existing file
func TestDiskWriter_WriteFile_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewDiskWriter(tmpDir)

	// Write initial content
	initialContent := []byte("initial")
	err := writer.WriteFile("test.txt", initialContent)
	require.NoError(t, err, "WriteFile should not return error")

	// Overwrite with new content
	newContent := []byte("updated content")
	err = writer.WriteFile("test.txt", newContent)
	require.NoError(t, err, "WriteFile should not return error on overwrite")

	// Verify new content
	fullPath := filepath.Join(tmpDir, "test.txt")
	readContent, err := os.ReadFile(fullPath)
	require.NoError(t, err, "should read file")
	assert.Equal(t, newContent, readContent, "file content should be updated")
}

// TestDiskWriter_WriteFile_EmptyContent tests writing an empty file
func TestDiskWriter_WriteFile_EmptyContent(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewDiskWriter(tmpDir)

	content := []byte("")
	err := writer.WriteFile("empty.txt", content)

	require.NoError(t, err, "WriteFile should not return error for empty content")

	fullPath := filepath.Join(tmpDir, "empty.txt")
	assert.FileExists(t, fullPath, "empty file should exist")

	readContent, err := os.ReadFile(fullPath)
	require.NoError(t, err, "should read file")
	assert.Empty(t, readContent, "file should be empty")
}

// TestDiskWriter_MakeDirectory tests creating a directory
func TestDiskWriter_MakeDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewDiskWriter(tmpDir)

	err := writer.MakeDirectory("testdir")

	require.NoError(t, err, "MakeDirectory should not return error")

	fullPath := filepath.Join(tmpDir, "testdir")
	assert.DirExists(t, fullPath, "directory should exist")
}

// TestDiskWriter_MakeDirectory_Nested tests creating nested directories
func TestDiskWriter_MakeDirectory_Nested(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewDiskWriter(tmpDir)

	err := writer.MakeDirectory("a/b/c/d")

	require.NoError(t, err, "MakeDirectory should not return error")

	// Verify all directories exist
	assert.DirExists(t, filepath.Join(tmpDir, "a"), "directory a should exist")
	assert.DirExists(t, filepath.Join(tmpDir, "a", "b"), "directory a/b should exist")
	assert.DirExists(t, filepath.Join(tmpDir, "a", "b", "c"), "directory a/b/c should exist")
	assert.DirExists(t, filepath.Join(tmpDir, "a", "b", "c", "d"), "directory a/b/c/d should exist")
}

// TestDiskWriter_MakeDirectory_AlreadyExists tests creating an existing directory
func TestDiskWriter_MakeDirectory_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewDiskWriter(tmpDir)

	// Create directory first time
	err := writer.MakeDirectory("testdir")
	require.NoError(t, err, "MakeDirectory should not return error")

	// Create same directory again
	err = writer.MakeDirectory("testdir")
	require.NoError(t, err, "MakeDirectory should not return error for existing directory")

	fullPath := filepath.Join(tmpDir, "testdir")
	assert.DirExists(t, fullPath, "directory should exist")
}

// TestDiskWriter_MultipleFiles tests writing multiple files
func TestDiskWriter_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewDiskWriter(tmpDir)

	files := map[string][]byte{
		"file1.txt":     []byte("content1"),
		"file2.txt":     []byte("content2"),
		"dir/file3.txt": []byte("content3"),
	}

	// Write all files
	for path, content := range files {
		err := writer.WriteFile(path, content)
		require.NoError(t, err, "WriteFile should not return error for %s", path)
	}

	// Verify all files exist with correct content
	for path, expectedContent := range files {
		fullPath := filepath.Join(tmpDir, path)
		assert.FileExists(t, fullPath, "file %s should exist", path)

		readContent, err := os.ReadFile(fullPath)
		require.NoError(t, err, "should read file %s", path)
		assert.Equal(t, expectedContent, readContent, "content should match for %s", path)
	}
}

// TestDiskWriter_WriteFile_InvalidPath tests writing to an invalid path
func TestDiskWriter_WriteFile_InvalidPath(t *testing.T) {
	// Use a directory that doesn't exist and can't be created (e.g., under a file)
	tmpDir := t.TempDir()
	writer := NewDiskWriter(tmpDir)

	// First create a regular file
	err := writer.WriteFile("blockerfile", []byte("blocker"))
	require.NoError(t, err, "should create blocker file")

	// Try to create a file under the regular file (should fail)
	err = writer.WriteFile("blockerfile/subfile.txt", []byte("test"))
	assert.Error(t, err, "WriteFile should return error when trying to create file under a regular file")
}

// MockWriter is a mock implementation of FileWriter for testing
type MockWriter struct {
	Files       map[string][]byte
	Directories map[string]bool
	WriteError  error
	DirError    error
}

// NewMockWriter creates a new MockWriter
func NewMockWriter() *MockWriter {
	return &MockWriter{
		Files:       make(map[string][]byte),
		Directories: make(map[string]bool),
	}
}

// WriteFile records the file write in memory
func (m *MockWriter) WriteFile(path string, content []byte) error {
	if m.WriteError != nil {
		return m.WriteError
	}
	m.Files[path] = content
	return nil
}

// MakeDirectory records the directory creation in memory
func (m *MockWriter) MakeDirectory(path string) error {
	if m.DirError != nil {
		return m.DirError
	}
	m.Directories[path] = true
	return nil
}

// TestMockWriter tests the MockWriter implementation
func TestMockWriter(t *testing.T) {
	mock := NewMockWriter()

	// Test WriteFile
	content := []byte("test content")
	err := mock.WriteFile("test.txt", content)
	require.NoError(t, err, "MockWriter.WriteFile should not return error")

	assert.Len(t, mock.Files, 1, "should have 1 file")
	assert.Equal(t, content, mock.Files["test.txt"], "file content should match")

	// Test MakeDirectory
	err = mock.MakeDirectory("testdir")
	require.NoError(t, err, "MockWriter.MakeDirectory should not return error")

	assert.Len(t, mock.Directories, 1, "should have 1 directory")
	assert.True(t, mock.Directories["testdir"], "directory should be recorded")
}

// TestMockWriter_Errors tests MockWriter error simulation
func TestMockWriter_Errors(t *testing.T) {
	t.Run("WriteFile error", func(t *testing.T) {
		mock := NewMockWriter()
		mock.WriteError = assert.AnError

		err := mock.WriteFile("test.txt", []byte("content"))
		assert.Error(t, err, "should return configured error")
		assert.Empty(t, mock.Files, "should not record file on error")
	})

	t.Run("MakeDirectory error", func(t *testing.T) {
		mock := NewMockWriter()
		mock.DirError = assert.AnError

		err := mock.MakeDirectory("testdir")
		assert.Error(t, err, "should return configured error")
		assert.Empty(t, mock.Directories, "should not record directory on error")
	})
}
