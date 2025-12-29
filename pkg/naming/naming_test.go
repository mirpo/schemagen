package naming

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"user", "User"},
		{"user_name", "UserName"},
		{"user-name", "UserName"},
		{"user name", "UserName"},
		{"api_key", "ApiKey"},
		{"API_KEY", "APIKEY"},
		{"some_long_variable_name", "SomeLongVariableName"},
		{"already_PascalCase", "AlreadyPascalCase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToPascalCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"User", "user"},
		{"user_name", "userName"},
		{"user-name", "userName"},
		{"UserName", "userName"},
		{"api_key", "apiKey"},
		{"APIKey", "aPIKey"},
		{"some_long_variable_name", "someLongVariableName"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"user", "user"},
		{"userName", "user_name"},
		{"UserName", "user_name"},
		{"APIKey", "api_key"}, // Fixed: acronyms should stay together
		{"someValue", "some_value"},
		{"HTTPResponse", "http_response"}, // Fixed: acronyms should stay together
		{"already_snake_case", "already_snake_case"},
		{"XMLParser", "xml_parser"},        // Additional test case
		{"parseHTMLDoc", "parse_html_doc"}, // Acronym in middle
		{"IOError", "io_error"},            // Short acronym
		{"ID", "id"},                       // Just acronym
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToConstantCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"user", "USER"},
		{"userName", "USER_NAME"},
		{"UserName", "USER_NAME"},
		{"apiKey", "API_KEY"},
		{"some_value", "SOME_VALUE"},
		{"ALREADY_CONSTANT", "ALREADY_CONSTANT"}, // Already in constant case, unchanged
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToConstantCase(tt.input)
			assert.Equal(t, tt.expected, result, "ToConstantCase(%q) should equal %q", tt.input, tt.expected)
		})
	}
}

func TestIsPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"User", true},
		{"UserName", true},
		{"userName", false},
		{"user_name", false},
		{"123", false},
		{"APIKey", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("IsPascalCase(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSplitOnDelimiters(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"user", []string{"user"}},
		{"user_name", []string{"user", "name"}},
		{"user-name", []string{"user", "name"}},
		{"user name", []string{"user", "name"}},
		{"user__name", []string{"user", "name"}},
		{"_user_name_", []string{"user", "name"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitOnDelimiters(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitOnDelimiters(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("splitOnDelimiters(%q)[%d] = %q, want %q", tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// Issue 3: Go Field Name Sanitization
func TestToGoFieldName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Valid names should pass through unchanged
		{"Name", "Name"},
		{"UserName", "UserName"},
		{"kebab-case", "KebabCase"},
		{"with spaces", "WithSpaces"},
		{"snake_case", "SnakeCase"},

		// Invalid: starts with number
		{"123numeric", "F123Numeric"},
		{"1stPlace", "F1StPlace"},
		{"99bottles", "F99Bottles"},

		// Invalid: starts with special char
		{"$dollar", "FDollar"},
		{"@special", "FSpecial"},
		{"#hashtag", "FHashtag"},

		// Invalid: contains dots
		{"with.dots", "WithDots"},
		{"some.nested.path", "SomeNestedPath"},

		// Invalid: multiple issues
		{"$123test", "F123Test"},
		{"@my-field", "FMyField"},

		// Edge cases
		{"", "Field"},           // Empty string
		{"_private", "Private"}, // Underscore is valid but we convert to PascalCase
		{"__dunder", "Dunder"},  // Double underscore

		// Already valid PascalCase
		{"PascalCase", "PascalCase"},
		{"CamelCase", "CamelCase"},
		{"API", "API"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToGoFieldName(tt.input)
			assert.Equal(t, tt.expected, result, "ToGoFieldName(%q) should equal %q", tt.input, tt.expected)
		})
	}
}

func TestIsValidGoIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Valid
		{"Name", true},
		{"UserName", true},
		{"_private", true},
		{"API", true},
		{"x", true},
		{"X123", true},

		// Invalid: starts with number
		{"123test", false},
		{"1", false},

		// Invalid: contains special chars
		{"$dollar", false},
		{"@special", false},
		{"with.dots", false},
		{"with-dashes", false},
		{"with spaces", false},

		// Edge cases
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isValidGoIdentifier(tt.input)
			assert.Equal(t, tt.expected, result, "isValidGoIdentifier(%q) should be %v", tt.input, tt.expected)
		})
	}
}
