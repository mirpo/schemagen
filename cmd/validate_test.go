package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCommandValidSchemas(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"validate", "../testdata/schemas/events/",
	})

	err := cmd.Execute()
	assert.NoError(t, err, "Valid schemas should pass")
}

func TestValidateCommandSingleFile(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"validate", "../testdata/schemas/foundation",
	})

	err := cmd.Execute()
	assert.NoError(t, err, "Valid single file should pass")
}

func TestValidateCommandInvalidFile(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"validate", "nonexistent.json",
	})

	err := cmd.Execute()
	require.Error(t, err)

	exitErr, ok := err.(ExitCodeError)
	require.True(t, ok, "Should return ExitCodeError")
	assert.Equal(t, 1, exitErr.Code, "Should exit with code 1 for validation failure")
}

func TestValidateCommandYAML(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"validate", "../testdata/schemas/basic/simple.yaml",
	})

	err := cmd.Execute()
	assert.NoError(t, err, "Valid YAML schema should pass")
}

func TestValidateCommandInvalidJSON(t *testing.T) {
	// Create a temporary invalid schema file
	tmpFile, err := os.CreateTemp("", "invalid-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte(`{invalid json}`))
	require.NoError(t, err)
	tmpFile.Close()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"validate", tmpFile.Name(),
	})

	err = cmd.Execute()
	require.Error(t, err)

	exitErr, ok := err.(ExitCodeError)
	require.True(t, ok, "Should return ExitCodeError")
	assert.Equal(t, 1, exitErr.Code, "Should exit with code 1 for validation error")
}

func TestValidateCommandWithExternalRefs(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"validate", "../testdata/schemas/events/event.json",
	})

	err := cmd.Execute()
	assert.NoError(t, err, "Schema with external refs should validate successfully")
}

func TestValidateCommandMultipleFilesWithErrors(t *testing.T) {
	// Create temp dir with mixed valid/invalid files
	tmpDir, err := os.MkdirTemp("", "validate-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Valid file
	validFile := tmpDir + "/valid.json"
	err = os.WriteFile(validFile, []byte(`{"type": "object", "properties": {"name": {"type": "string"}}}`), 0o644)
	require.NoError(t, err)

	// Invalid file
	invalidFile := tmpDir + "/invalid.json"
	err = os.WriteFile(invalidFile, []byte(`{bad json}`), 0o644)
	require.NoError(t, err)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"validate", tmpDir,
	})

	err = cmd.Execute()
	require.Error(t, err)

	exitErr, ok := err.(ExitCodeError)
	require.True(t, ok, "Should return ExitCodeError")
	assert.Equal(t, 1, exitErr.Code, "Should exit with code 1 for validation errors")
}
