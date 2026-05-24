package graph

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
		{"hello_world", "HelloWorld"},
		{"hello-world", "HelloWorld"},
		{"hello world", "HelloWorld"},
		{"hello.world", "HelloWorld"},
		{"helloWorld", "HelloWorld"},
		{"ALLCAPS", "ALLCAPS"},
		{"already", "Already"},
		{"multi_word_string", "MultiWordString"},
		{"_leading", "Leading"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ToPascalCase(tt.input))
		})
	}
}

func TestToConstantCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"helloWorld", "HELLO_WORLD"},
		{"hello_world", "HELLO_WORLD"},
		{"HTTPSProxy", "HTTPS_PROXY"},
		{"simple", "SIMPLE"},
		{"already_UPPER", "ALREADY_UPPER"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ToConstantCase(tt.input))
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"firstName", "first_name"},
		{"HTTPSProxy", "https_proxy"},
		{"user123Name", "user123_name"},
		{"already_snake", "already_snake"},
		{"A", "a"},
		{"ABC", "abc"},
		{"camelCase", "camel_case"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ToSnakeCase(tt.input))
		})
	}
}
