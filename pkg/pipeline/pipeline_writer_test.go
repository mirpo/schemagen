package pipeline

import (
	"strings"
	"testing"

	"github.com/mirpo/schemagen/pkg/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_WithMemoryWriter(t *testing.T) {
	schemas, err := parse.Load("../../testdata/schemas/foundation/foundation.json")
	require.NoError(t, err)

	w := NewMemoryWriter()
	cfg := &Config{
		Schemas:          schemas,
		OutDir:           "unused",
		Language:         LanguageTypeScript,
		DisableTimestamp: true,
		Writer:           w,
		TypeScript:       &TypeScriptConfig{},
	}

	require.NoError(t, Run(cfg))
	assert.NotEmpty(t, w.Files)

	for path, content := range w.Files {
		assert.NotEmpty(t, content, "file %s should have content", path)
		assert.True(t, strings.HasSuffix(path, ".ts"), "file %s should be .ts", path)
	}
}

func TestRun_WithoutWriter_DefaultsToDisk(t *testing.T) {
	schemas, err := parse.Load("../../testdata/schemas/foundation/foundation.json")
	require.NoError(t, err)

	outDir := t.TempDir()
	cfg := &Config{
		Schemas:          schemas,
		OutDir:           outDir,
		Language:         LanguageTypeScript,
		DisableTimestamp: true,
		TypeScript:       &TypeScriptConfig{},
	}

	require.NoError(t, Run(cfg))
}
