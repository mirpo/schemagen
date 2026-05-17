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
		{"API_KEY", "APIKEY"},
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
		{"parseHTMLDoc", "parse_html_doc"},
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
		{"some_value", "SOME_VALUE"},
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
		{"userName", false},
		{"user_name", false},
		{"123", false},
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
		{"with.dots", "WithDots"},
		{"123numeric", "F123Numeric"},
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
		{"X123", true},
		{"123test", false},
		{"with.dots", false},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, isValidGoIdentifier(tt.in))
		})
	}
}
