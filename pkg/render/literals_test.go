package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLiteralFormatter_FormatValue(t *testing.T) {
	tests := []struct {
		name  string
		val   any
		goFmt string
		tsFmt string
		pyFmt string
	}{
		{"string", "hello", `"hello"`, `"hello"`, `"hello"`},
		{"string with quotes", `say "hi"`, `"say \"hi\""`, `"say \"hi\""`, `"say \"hi\""`},
		{"float64 whole", float64(42), "42", "42", "42"},
		{"float64 decimal", 3.14, "3.14", "3.14", "3.14"},
		{"int", int(7), "7", "7", "7"},
		{"int64", int64(999), "999", "999", "999"},
		{"int32", int32(16), "16", "16", "16"},
		{"bool true", true, "true", "true", "True"},
		{"bool false", false, "false", "false", "False"},
		{"nil", nil, "nil", "null", "None"},
		{"default fallback", []int{1, 2}, "[1 2]", "[1 2]", "[1 2]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.goFmt, GoLiterals.FormatValue(tt.val))
			assert.Equal(t, tt.tsFmt, TSLiterals.FormatValue(tt.val))
			assert.Equal(t, tt.pyFmt, PyLiterals.FormatValue(tt.val))
		})
	}
}
