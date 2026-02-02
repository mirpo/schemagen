package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyCommand(t *testing.T) {
	t.Run("no changes", func(t *testing.T) {
		tmpDir := t.TempDir()
		generateToDir(t, "../testdata/schemas/foundation", tmpDir, "", "--disable-timestamp")
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"verify", "../testdata/schemas/foundation", "--out-ts", tmpDir, "--disable-timestamp"})
		assert.NoError(t, cmd.Execute())
	})

	t.Run("has drift", func(t *testing.T) {
		tmpDir := t.TempDir()
		generateToDir(t, "../testdata/schemas/foundation", tmpDir, "", "--disable-timestamp")
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "User.ts"), []byte("modified"), 0o644))
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"verify", "../testdata/schemas/foundation", "--out-ts", tmpDir, "--disable-timestamp"})
		requireExitCode(t, cmd.Execute(), 2)
	})

	t.Run("quiet mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		generateToDir(t, "../testdata/schemas/foundation", tmpDir, "", "--disable-timestamp")
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "User.ts"), []byte("modified"), 0o644))
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"verify", "../testdata/schemas/foundation", "--out-ts", tmpDir, "--quiet", "--disable-timestamp"})
		requireExitCode(t, cmd.Execute(), 2)
	})

	t.Run("missing directory", func(t *testing.T) {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"verify", "../testdata/schemas/foundation", "--out-ts", "/path/that/does/not/exist"})
		requireExitCode(t, cmd.Execute(), 2)
	})

	t.Run("both languages", func(t *testing.T) {
		tmpDirTS, tmpDirPY := t.TempDir(), t.TempDir()
		generateToDir(t, "../testdata/schemas/foundation", tmpDirTS, tmpDirPY, "--disable-timestamp")
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"verify", "../testdata/schemas/foundation", "--out-ts", tmpDirTS, "--out-py", tmpDirPY, "--disable-timestamp"})
		assert.NoError(t, cmd.Execute())
	})

	t.Run("deleted file", func(t *testing.T) {
		tmpDir := t.TempDir()
		generateToDir(t, "../testdata/schemas/foundation", tmpDir, "", "--disable-timestamp")
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Extra.ts"), []byte("extra"), 0o644))
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"verify", "../testdata/schemas/foundation", "--out-ts", tmpDir, "--disable-timestamp"})
		requireExitCode(t, cmd.Execute(), 2)
	})
}
