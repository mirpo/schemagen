package golang

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToGoFieldName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "Field"},
		{"name", "Name"},
		{"user_name", "UserName"},
		{"user-name", "UserName"},
		{"my field", "MyField"},
		{"a.b.c", "ABC"},
		{"123abc", "F123Abc"},
		{"$value", "FValue"},
		{"MyField", "MyField"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, toGoFieldName(tt.input))
		})
	}
}

func TestIsValidGoIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"Name", true},
		{"_private", true},
		{"123", false},
		{"has space", false},
		{"has-dash", false},
		{"abc123", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, isValidGoIdentifier(tt.input))
		})
	}
}
