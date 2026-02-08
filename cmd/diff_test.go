package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffCommand(t *testing.T) {
	t.Run("no changes", func(t *testing.T) {
		tmpDir := t.TempDir()
		generateToDir(t, "../testdata/schemas/foundation", tmpDir, "", "--disable-timestamp")
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"diff", "../testdata/schemas/foundation", "--out-ts", tmpDir, "--disable-timestamp"})
		assert.NoError(t, cmd.Execute())
	})

	t.Run("has changes", func(t *testing.T) {
		tmpDir := t.TempDir()
		generateToDir(t, "../testdata/schemas/foundation", tmpDir, "")
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "User.ts"), []byte("modified"), 0o644))
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"diff", "../testdata/schemas/foundation", "--out-ts", tmpDir})
		requireExitCode(t, cmd.Execute(), 2)
	})

	t.Run("new file", func(t *testing.T) {
		tmpDir := t.TempDir()
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"diff", "../testdata/schemas/foundation", "--out-ts", tmpDir})
		requireExitCode(t, cmd.Execute(), 2)
	})

	t.Run("no color flag", func(t *testing.T) {
		tmpDir := t.TempDir()
		generateToDir(t, "../testdata/schemas/foundation", tmpDir, "")
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "User.ts"), []byte("modified"), 0o644))
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"diff", "../testdata/schemas/foundation", "--out-ts", tmpDir, "--no-color"})
		require.Error(t, cmd.Execute())
	})

	t.Run("multiple files", func(t *testing.T) {
		tmpDir := t.TempDir()
		generateToDir(t, "../testdata/schemas/events", tmpDir, "", "--disable-timestamp")
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "event.ts"), []byte("modified"), 0o644))
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"diff", "../testdata/schemas/events", "--out-ts", tmpDir, "--disable-timestamp"})
		requireExitCode(t, cmd.Execute(), 2)
	})

	t.Run("both languages", func(t *testing.T) {
		tmpDirTS, tmpDirPY := t.TempDir(), t.TempDir()
		generateToDir(t, "../testdata/schemas/foundation", tmpDirTS, tmpDirPY)
		require.NoError(t, os.WriteFile(filepath.Join(tmpDirTS, "User.ts"), []byte("modified"), 0o644))
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"diff", "../testdata/schemas/foundation", "--out-ts", tmpDirTS, "--out-py", tmpDirPY})
		require.Error(t, cmd.Execute())
	})
}
