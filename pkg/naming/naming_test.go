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
		{"user", "User"},
		{"user_name", "UserName"},
		{"user-name", "UserName"},
		{"user name", "UserName"},
		{"user.name", "UserName"},
		{"api_key", "ApiKey"},
		{"API_KEY", "APIKEY"},
		{"some_long_variable_name", "SomeLongVariableName"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, ToPascalCase(tt.in))
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"userName", "user_name"},
		{"UserName", "user_name"},
		{"APIKey", "api_key"},
		{"someValue", "some_value"},
		{"HTTPResponse", "http_response"},
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
		{"user", "USER"},
		{"userName", "USER_NAME"},
		{"UserName", "USER_NAME"},
		{"apiKey", "API_KEY"},
		{"APIKey", "API_KEY"},
		{"HTTPResponse", "HTTP_RESPONSE"},
		{"some_value", "SOME_VALUE"},
		{"api-key", "API_KEY"},
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
		{"User", true},
		{"UserName", true},
		{"userName", false},
		{"user_name", false},
		{"123", false},
		{"APIKey", true},
		{"HTTP_RESPONSE", false},
		{"Hello_world", false},
		{"Hello world", false},
		{"my-component", false},
		{"some.thing", false},
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
		{"user", []string{"user"}},
		{"user_name", []string{"user", "name"}},
		{"user-name", []string{"user", "name"}},
		{"user name", []string{"user", "name"}},
		{"user.name", []string{"user", "name"}},
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
		{"kebab-case", "KebabCase"},
		{"with spaces", "WithSpaces"},
		{"snake_case", "SnakeCase"},
		{"with.dots", "WithDots"},
		{"123numeric", "F123Numeric"},
		{"$dollar", "FDollar"},
		{"@special", "FSpecial"},
		{"$123test", "F123Test"},
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
		{"_private", true},
		{"API", true},
		{"X123", true},
		{"123test", false},
		{"$dollar", false},
		{"with.dots", false},
		{"with-dashes", false},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, isValidGoIdentifier(tt.in))
		})
	}
}
