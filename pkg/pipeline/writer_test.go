package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryWriter(t *testing.T) {
	w := NewMemoryWriter()

	require.NoError(t, w.WriteFile("user.ts", []byte("export interface User {}")))
	require.NoError(t, w.WriteFile("models/profile.ts", []byte("export interface Profile {}")))

	assert.Equal(t, "export interface User {}", string(w.Files["user.ts"]))
	assert.Equal(t, "export interface Profile {}", string(w.Files["models/profile.ts"]))
	assert.Len(t, w.Files, 2)
}

func TestMemoryWriter_Overwrite(t *testing.T) {
	w := NewMemoryWriter()

	require.NoError(t, w.WriteFile("a.ts", []byte("v1")))
	require.NoError(t, w.WriteFile("a.ts", []byte("v2")))

	assert.Equal(t, "v2", string(w.Files["a.ts"]))
	assert.Len(t, w.Files, 1)
}
