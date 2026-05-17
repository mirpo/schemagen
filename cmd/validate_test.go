package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCommand(t *testing.T) {
	t.Run("valid schemas", func(t *testing.T) {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"validate", "../testdata/schemas/events/"})
		require.NoError(t, cmd.Execute())
	})

	t.Run("rejects extra arguments", func(t *testing.T) {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"validate", "../testdata/schemas/basic/", "extra-arg"})
		require.Error(t, cmd.Execute())
	})

	t.Run("invalid file", func(t *testing.T) {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"validate", "nonexistent.json"})
		requireExitCode(t, cmd.Execute(), 1)
	})

	t.Run("yaml", func(t *testing.T) {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"validate", "../testdata/schemas/basic/simple.yaml"})
		require.NoError(t, cmd.Execute())
	})

	t.Run("invalid json", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "invalid-*.json")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		_, _ = tmpFile.Write([]byte(`{invalid json}`))
		tmpFile.Close()

		cmd := NewRootCmd()
		cmd.SetArgs([]string{"validate", tmpFile.Name()})
		requireExitCode(t, cmd.Execute(), 1)
	})

	t.Run("multiple files with errors", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "validate-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		_ = os.WriteFile(tmpDir+"/valid.json", []byte(`{"type": "object"}`), 0o644)
		_ = os.WriteFile(tmpDir+"/invalid.json", []byte(`{bad json}`), 0o644)

		cmd := NewRootCmd()
		cmd.SetArgs([]string{"validate", tmpDir})
		requireExitCode(t, cmd.Execute(), 1)
	})
}
