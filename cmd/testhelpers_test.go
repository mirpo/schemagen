package cmd

import (
	"testing"

	"github.com/mirpo/schemagen/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateToDir(t *testing.T, schemaPath, outTS, outPY string, extraFlags ...string) {
	t.Helper()
	args := []string{"generate", schemaPath}
	if outTS != "" {
		args = append(args, "--out-ts", outTS)
	}
	if outPY != "" {
		args = append(args, "--out-py", outPY)
	}
	args = append(args, extraFlags...)
	cmd := NewRootCmd()
	cmd.SetArgs(args)
	require.NoError(t, cmd.Execute())
}

func requireExitCode(t *testing.T, err error, code int) {
	t.Helper()
	require.Error(t, err)
	if exitErr, ok := err.(*errors.ExitCodeError); ok {
		assert.Equal(t, code, exitErr.Code)
	}
}
