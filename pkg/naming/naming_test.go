package naming

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"user", "User"},
		{"user_name", "UserName"},
		{"user-name", "UserName"},
		{"user name", "UserName"},
		{"user.name", "UserName"},
		{"api_key", "ApiKey"},
		{"API_KEY", "APIKEY"},
		{"some_long_variable_name", "SomeLongVariableName"},
		{"already_PascalCase", "AlreadyPascalCase"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, ToPascalCase(tt.in))
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"User", "user"},
		{"user_name", "userName"},
		{"user-name", "userName"},
		{"UserName", "userName"},
		{"api_key", "apiKey"},
		{"APIKey", "aPIKey"}, // keep existing behavior: only lower first rune
		{"some_long_variable_name", "someLongVariableName"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, ToCamelCase(tt.in))
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"user", "user"},
		{"userName", "user_name"},
		{"UserName", "user_name"},
		{"APIKey", "api_key"},
		{"someValue", "some_value"},
		{"HTTPResponse", "http_response"},
		{"already_snake_case", "already_snake_case"},
		{"XMLParser", "xml_parser"},
		{"parseHTMLDoc", "parse_html_doc"},
		{"IOError", "io_error"},
		{"ID", "id"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, ToSnakeCase(tt.in))
		})
	}
}

func TestToConstantCase(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"user", "USER"},
		{"userName", "USER_NAME"},
		{"UserName", "USER_NAME"},
		{"apiKey", "API_KEY"},
		{"APIKey", "API_KEY"},
		{"HTTPResponse", "HTTP_RESPONSE"},
		{"some_value", "SOME_VALUE"},
		{"api-key", "API_KEY"},
		{"api key", "API_KEY"},
		{"api.key", "API_KEY"},
		{"ALREADY_CONSTANT", "ALREADY_CONSTANT"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, ToConstantCase(tt.in))
		})
	}
}

func TestIsPascalCase(t *testing.T) {
	tests := []struct {
		in   string
		want bool
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
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, IsPascalCase(tt.in))
		})
	}
}

func TestSplitOnDelimiters(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"user", []string{"user"}},
		{"user_name", []string{"user", "name"}},
		{"user-name", []string{"user", "name"}},
		{"user name", []string{"user", "name"}},
		{"user.name", []string{"user", "name"}}, // dot delimiter supported
		{"user__name", []string{"user", "name"}},
		{"_user_name_", []string{"user", "name"}},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, splitOnDelimiters(tt.in))
		})
	}
}

func TestToGoFieldName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Name", "Name"},
		{"UserName", "UserName"},
		{"kebab-case", "KebabCase"},
		{"with spaces", "WithSpaces"},
		{"snake_case", "SnakeCase"},
		{"with.dots", "WithDots"},
		{"some.nested.path", "SomeNestedPath"},

		{"123numeric", "F123Numeric"},
		{"1stPlace", "F1StPlace"},
		{"99bottles", "F99Bottles"},

		{"$dollar", "FDollar"},
		{"@special", "FSpecial"},
		{"#hashtag", "FHashtag"},

		{"$123test", "F123Test"},
		{"@my-field", "FMyField"},

		{"", "Field"},
		{"_private", "Private"},
		{"__dunder", "Dunder"},

		{"PascalCase", "PascalCase"},
		{"CamelCase", "CamelCase"},
		{"API", "API"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, ToGoFieldName(tt.in))
		})
	}
}

func TestIsValidGoIdentifier(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"Name", true},
		{"UserName", true},
		{"_private", true},
		{"API", true},
		{"x", true},
		{"X123", true},

		{"123test", false},
		{"1", false},

		{"$dollar", false},
		{"@special", false},
		{"with.dots", false},
		{"with-dashes", false},
		{"with spaces", false},

		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, isValidGoIdentifier(tt.in))
		})
	}
}
