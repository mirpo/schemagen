package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mirpo/schemagen/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyCommand_NoChanges(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate files (use --disable-timestamp to avoid flaky tests on slow CI)
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"generate", "../testdata/schemas/foundation", "--out-ts", tmpDir, "--disable-timestamp"})
	err := cmd.Execute()
	require.NoError(t, err)

	// Run verify - should succeed with no drift
	cmd = NewRootCmd()
	cmd.SetArgs([]string{"verify", "../testdata/schemas/foundation", "--out-ts", tmpDir, "--disable-timestamp"})
	err = cmd.Execute()
	assert.NoError(t, err, "verify should succeed when no drift")
}

func TestVerifyCommand_HasDrift(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate files (use --disable-timestamp to avoid flaky tests on slow CI)
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"generate", "../testdata/schemas/foundation", "--out-ts", tmpDir, "--disable-timestamp"})
	err := cmd.Execute()
	require.NoError(t, err)

	// Modify the file to simulate drift
	userFile := filepath.Join(tmpDir, "User.ts")
	err = os.WriteFile(userFile, []byte("export interface User { modified: true; }"), 0o644)
	require.NoError(t, err)

	// Run verify - should detect drift
	cmd = NewRootCmd()
	cmd.SetArgs([]string{"verify", "../testdata/schemas/foundation", "--out-ts", tmpDir, "--disable-timestamp"})
	err = cmd.Execute()
	assert.Error(t, err, "verify should fail when drift detected")

	// Check exit code
	if exitErr, ok := err.(*errors.ExitCodeError); ok {
		assert.Equal(t, 2, exitErr.Code, "exit code should be 2 for drift")
	}
}

func TestVerifyCommand_Quiet(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate files (use --disable-timestamp to avoid flaky tests on slow CI)
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"generate", "../testdata/schemas/foundation", "--out-ts", tmpDir, "--disable-timestamp"})
	err := cmd.Execute()
	require.NoError(t, err)

	// Modify the file
	userFile := filepath.Join(tmpDir, "User.ts")
	err = os.WriteFile(userFile, []byte("export interface User { modified: true; }"), 0o644)
	require.NoError(t, err)

	// Run verify in quiet mode
	cmd = NewRootCmd()
	cmd.SetArgs([]string{"verify", "../testdata/schemas/foundation", "--out-ts", tmpDir, "--quiet", "--disable-timestamp"})
	err = cmd.Execute()
	require.Error(t, err, "verify should detect drift in quiet mode")

	// Should still have correct exit code
	if exitErr, ok := err.(*errors.ExitCodeError); ok {
		assert.Equal(t, 2, exitErr.Code)
	}
}

func TestVerifyCommand_MissingDirectory(t *testing.T) {
	// Run verify against non-existent directory
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"verify", "../testdata/schemas/foundation", "--out-ts", "/path/that/does/not/exist"})
	err := cmd.Execute()
	require.Error(t, err, "verify should fail when directory doesn't exist")

	// Check exit code
	if exitErr, ok := err.(*errors.ExitCodeError); ok {
		assert.Equal(t, 2, exitErr.Code)
	}
}

func TestVerifyCommand_BothLanguages(t *testing.T) {
	tmpDirTS := t.TempDir()
	tmpDirPY := t.TempDir()

	// Generate both languages (use --disable-timestamp to avoid flaky tests on slow CI)
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"generate", "../testdata/schemas/foundation", "--out-ts", tmpDirTS, "--out-py", tmpDirPY, "--disable-timestamp"})
	err := cmd.Execute()
	require.NoError(t, err)

	// Verify should succeed with no drift
	cmd = NewRootCmd()
	cmd.SetArgs([]string{"verify", "../testdata/schemas/foundation", "--out-ts", tmpDirTS, "--out-py", tmpDirPY, "--disable-timestamp"})
	err = cmd.Execute()
	assert.NoError(t, err, "verify should succeed for both languages when no drift")
}

func TestVerifyCommand_DeletedFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate User.ts (use --disable-timestamp to avoid flaky tests on slow CI)
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"generate", "../testdata/schemas/foundation", "--out-ts", tmpDir, "--disable-timestamp"})
	err := cmd.Execute()
	require.NoError(t, err)

	// Add an extra file that wasn't generated
	extraFile := filepath.Join(tmpDir, "Extra.ts")
	err = os.WriteFile(extraFile, []byte("export interface Extra {}"), 0o644)
	require.NoError(t, err)

	// Run verify - should detect the extra file as deleted (exists in dir but not generated)
	cmd = NewRootCmd()
	cmd.SetArgs([]string{"verify", "../testdata/schemas/foundation", "--out-ts", tmpDir, "--disable-timestamp"})
	err = cmd.Execute()
	assert.Error(t, err, "verify should detect extra files as drift")

	if exitErr, ok := err.(*errors.ExitCodeError); ok {
		assert.Equal(t, 2, exitErr.Code)
	}
}
