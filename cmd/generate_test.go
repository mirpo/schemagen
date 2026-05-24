package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCommand(t *testing.T) {
	t.Run("typescript output", func(t *testing.T) {
		tmpDir := t.TempDir()
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"generate", "../testdata/schemas/basic", "--out-ts", tmpDir})
		require.NoError(t, cmd.Execute())

		entries, err := os.ReadDir(tmpDir)
		require.NoError(t, err)
		assert.NotEmpty(t, entries)
	})

	t.Run("python output", func(t *testing.T) {
		tmpDir := t.TempDir()
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"generate", "../testdata/schemas/basic", "--out-py", tmpDir})
		require.NoError(t, cmd.Execute())

		entries, err := os.ReadDir(tmpDir)
		require.NoError(t, err)
		assert.NotEmpty(t, entries)
	})

	t.Run("go output", func(t *testing.T) {
		tmpDir := t.TempDir()
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"generate", "../testdata/schemas/basic", "--out-go", tmpDir})
		require.NoError(t, cmd.Execute())

		entries, err := os.ReadDir(tmpDir)
		require.NoError(t, err)
		assert.NotEmpty(t, entries)
	})

	t.Run("multiple languages", func(t *testing.T) {
		tsDir := t.TempDir()
		pyDir := t.TempDir()
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"generate", "../testdata/schemas/basic", "--out-ts", tsDir, "--out-py", pyDir})
		require.NoError(t, cmd.Execute())

		tsEntries, _ := os.ReadDir(tsDir)
		pyEntries, _ := os.ReadDir(pyDir)
		assert.NotEmpty(t, tsEntries)
		assert.NotEmpty(t, pyEntries)
	})

	t.Run("missing input arg", func(t *testing.T) {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"generate"})
		assert.Error(t, cmd.Execute())
	})

	t.Run("invalid input path", func(t *testing.T) {
		tmpDir := t.TempDir()
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"generate", "/nonexistent/path", "--out-ts", tmpDir})
		assert.Error(t, cmd.Execute())
	})

	t.Run("bundle strategy", func(t *testing.T) {
		tmpDir := t.TempDir()
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"generate", "../testdata/schemas/foundation", "--out-ts", tmpDir, "--output-strategy", "bundle"})
		require.NoError(t, cmd.Execute())

		_, err := os.Stat(filepath.Join(tmpDir, "types.ts"))
		assert.NoError(t, err)
	})
}
