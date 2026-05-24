package py

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizePythonIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "field"},
		{"name", "name"},
		{"123abc", "field_123abc"},
		{"$value", "field__value"},
		{"class", "class_"},
		{"my-field", "my_field"},
		{"my.field", "my_field"},
		{"_private", "_private"},
		{"normalName", "normalName"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizePythonIdentifier(tt.input))
		})
	}
}
