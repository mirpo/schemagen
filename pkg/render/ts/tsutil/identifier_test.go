package tsutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNeedsQuoting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid identifiers - no quoting needed
		{"simple", "name", false},
		{"with underscore", "user_name", false},
		{"with dollar", "$value", false},
		{"underscore start", "_private", false},
		{"mixed case", "userName", false},
		{"with numbers", "user123", false},

		// Invalid identifiers - quoting needed
		{"empty", "", true},
		{"starts with number", "123abc", true},
		{"contains hyphen", "user-name", true},
		{"contains space", "user name", true},
		{"contains dot", "user.name", true},
		{"special char", "user@name", true},

		// Reserved keywords - quoting needed
		{"reserved class", "class", true},
		{"reserved const", "const", true},
		{"reserved function", "function", true},
		{"reserved if", "if", true},
		{"reserved return", "return", true},
		{"reserved export", "export", true},
		{"reserved import", "import", true},
		{"reserved await", "await", true},
		{"reserved yield", "yield", true},

		// Similar to reserved but not reserved
		{"className not reserved", "className", false},
		{"classes not reserved", "classes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, NeedsQuoting(tt.input))
		})
	}
}

func TestReservedKeywords(t *testing.T) {
	// Ensure all expected keywords are present
	expectedKeywords := []string{
		"break", "case", "catch", "continue", "debugger", "default", "delete", "do",
		"else", "finally", "for", "function", "if", "in", "instanceof", "new",
		"return", "switch", "this", "throw", "try", "typeof", "var", "void",
		"while", "with", "class", "const", "enum", "export", "extends", "import",
		"super", "implements", "interface", "let", "package", "private", "protected",
		"public", "static", "yield", "await",
	}

	for _, kw := range expectedKeywords {
		assert.True(t, ReservedKeywords[kw], "expected %q to be a reserved keyword", kw)
	}

	// Ensure count matches
	assert.Len(t, ReservedKeywords, len(expectedKeywords))
}
