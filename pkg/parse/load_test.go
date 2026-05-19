package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("loads single file", func(t *testing.T) {
		schemas, err := Load("../../testdata/schemas/foundation/foundation.json")
		require.NoError(t, err)
		assert.Len(t, schemas, 1)
		assert.Equal(t, "Foundation", schemas[0].Name)
	})

	t.Run("loads directory", func(t *testing.T) {
		schemas, err := Load("../../testdata/schemas/basic/")
		require.NoError(t, err)
		assert.Greater(t, len(schemas), 1)
	})

	t.Run("nonexistent path", func(t *testing.T) {
		_, err := Load("/nonexistent/path")
		assert.Error(t, err)
	})
}
