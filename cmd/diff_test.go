package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mirpo/schemagen/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffCommand_NoChanges(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate files
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"generate", "../testdata/schemas/foundation", "--out-ts", tmpDir})
	err := cmd.Execute()
	require.NoError(t, err)

	// Run diff - should show no differences
	cmd = NewRootCmd()
	cmd.SetArgs([]string{"diff", "../testdata/schemas/foundation", "--out-ts", tmpDir})
	err = cmd.Execute()
	assert.NoError(t, err, "diff should succeed with no changes")
}

func TestDiffCommand_HasChanges(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate files
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"generate", "../testdata/schemas/foundation", "--out-ts", tmpDir})
	err := cmd.Execute()
	require.NoError(t, err)

	// Modify the file
	userFile := filepath.Join(tmpDir, "User.ts")
	err = os.WriteFile(userFile, []byte("export interface User { modified: true; }"), 0o644)
	require.NoError(t, err)

	// Run diff - should detect changes
	cmd = NewRootCmd()
	cmd.SetArgs([]string{"diff", "../testdata/schemas/foundation", "--out-ts", tmpDir})
	err = cmd.Execute()
	assert.Error(t, err, "diff should fail when changes detected")

	// Check exit code
	if exitErr, ok := err.(*errors.ExitCodeError); ok {
		assert.Equal(t, 2, exitErr.Code, "exit code should be 2 for differences")
	}
}

func TestDiffCommand_NewFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Don't generate - directory is empty

	// Run diff against non-existent directory
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"diff", "../testdata/schemas/foundation", "--out-ts", tmpDir})
	err := cmd.Execute()
	require.Error(t, err, "diff should detect new files")

	// Check exit code
	if exitErr, ok := err.(*errors.ExitCodeError); ok {
		assert.Equal(t, 2, exitErr.Code, "exit code should be 2 for differences")
	}
}

func TestDiffCommand_NoColorFlag(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate files
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"generate", "../testdata/schemas/foundation", "--out-ts", tmpDir})
	err := cmd.Execute()
	require.NoError(t, err)

	// Modify the file
	userFile := filepath.Join(tmpDir, "User.ts")
	err = os.WriteFile(userFile, []byte("export interface User { modified: true; }"), 0o644)
	require.NoError(t, err)

	// Run diff with --no-color
	cmd = NewRootCmd()
	cmd.SetArgs([]string{"diff", "../testdata/schemas/foundation", "--out-ts", tmpDir, "--no-color"})
	err = cmd.Execute()
	assert.Error(t, err, "diff should detect changes even with --no-color")
}

func TestDiffCommand_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate multiple files from events directory
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"generate", "../testdata/schemas/events", "--out-ts", tmpDir, "--disable-timestamp"})
	err := cmd.Execute()
	require.NoError(t, err)

	// Modify one of the generated files
	eventFile := filepath.Join(tmpDir, "event.ts")
	err = os.WriteFile(eventFile, []byte("export interface Event { modified: true; }"), 0o644)
	require.NoError(t, err)

	// Run diff - should detect changes in event.ts
	cmd = NewRootCmd()
	cmd.SetArgs([]string{"diff", "../testdata/schemas/events", "--out-ts", tmpDir, "--disable-timestamp"})
	err = cmd.Execute()
	assert.Error(t, err, "diff should detect changes")

	if exitErr, ok := err.(*errors.ExitCodeError); ok {
		assert.Equal(t, 2, exitErr.Code)
	}
}

func TestDiffCommand_BothLanguages(t *testing.T) {
	tmpDirTS := t.TempDir()
	tmpDirPY := t.TempDir()

	// Generate both languages
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"generate", "../testdata/schemas/foundation", "--out-ts", tmpDirTS, "--out-py", tmpDirPY})
	err := cmd.Execute()
	require.NoError(t, err)

	// Modify TypeScript file
	userFile := filepath.Join(tmpDirTS, "User.ts")
	err = os.WriteFile(userFile, []byte("export interface User { modified: true; }"), 0o644)
	require.NoError(t, err)

	// Run diff for both
	cmd = NewRootCmd()
	cmd.SetArgs([]string{"diff", "../testdata/schemas/foundation", "--out-ts", tmpDirTS, "--out-py", tmpDirPY})
	err = cmd.Execute()
	assert.Error(t, err, "diff should detect TS changes")
}
