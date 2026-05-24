package tsutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteJSDoc(t *testing.T) {
	t.Run("empty description writes nothing", func(t *testing.T) {
		var sb strings.Builder
		WriteJSDoc(&sb, "", "")
		assert.Empty(t, sb.String())
	})

	t.Run("writes multi-line block", func(t *testing.T) {
		var sb strings.Builder
		WriteJSDoc(&sb, "", "A user account")
		assert.Equal(t, "/**\n * A user account\n */\n", sb.String())
	})

	t.Run("respects indent", func(t *testing.T) {
		var sb strings.Builder
		WriteJSDoc(&sb, "  ", "Indented")
		assert.Equal(t, "  /**\n   * Indented\n   */\n", sb.String())
	})
}

func TestWriteJSDocSingleLine(t *testing.T) {
	t.Run("empty description writes nothing", func(t *testing.T) {
		var sb strings.Builder
		WriteJSDocSingleLine(&sb, "", "")
		assert.Empty(t, sb.String())
	})

	t.Run("writes single-line comment", func(t *testing.T) {
		var sb strings.Builder
		WriteJSDocSingleLine(&sb, "", "Short desc")
		assert.Equal(t, "/** Short desc */\n", sb.String())
	})

	t.Run("respects indent", func(t *testing.T) {
		var sb strings.Builder
		WriteJSDocSingleLine(&sb, "\t", "Tabbed")
		assert.Equal(t, "\t/** Tabbed */\n", sb.String())
	})
}

func TestWriteJSDocWithFormat(t *testing.T) {
	t.Run("both empty writes nothing", func(t *testing.T) {
		var sb strings.Builder
		WriteJSDocWithFormat(&sb, "", "", "")
		assert.Empty(t, sb.String())
	})

	t.Run("description only uses single-line", func(t *testing.T) {
		var sb strings.Builder
		WriteJSDocWithFormat(&sb, "", "A field", "")
		assert.Equal(t, "/** A field */\n", sb.String())
	})

	t.Run("format only uses multi-line", func(t *testing.T) {
		var sb strings.Builder
		WriteJSDocWithFormat(&sb, "", "", "date-time")
		assert.Equal(t, "/**\n * @format date-time\n */\n", sb.String())
	})

	t.Run("both description and format", func(t *testing.T) {
		var sb strings.Builder
		WriteJSDocWithFormat(&sb, "", "Created at", "date-time")
		expected := "/**\n * Created at\n * @format date-time\n */\n"
		assert.Equal(t, expected, sb.String())
	})

	t.Run("with indent", func(t *testing.T) {
		var sb strings.Builder
		WriteJSDocWithFormat(&sb, "  ", "Field", "uuid")
		expected := "  /**\n   * Field\n   * @format uuid\n   */\n"
		assert.Equal(t, expected, sb.String())
	})
}
