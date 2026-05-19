package cmd

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)

	origOut := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = origOut })

	fn()

	w.Close()
	os.Stdout = origOut

	data, err := io.ReadAll(r)
	r.Close()
	require.NoError(t, err)
	return string(data)
}

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
	if exitErr, ok := err.(*exitCodeError); ok {
		assert.Equal(t, code, exitErr.Code)
	}
}
