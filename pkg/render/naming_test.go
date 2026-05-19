package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeEnumMember(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "EMPTY"},
		{"ACTIVE", "ACTIVE"},
		{"123", "N_123"},
		{"0_VALUE", "N_0_VALUE"},
		{"normal", "normal"},
		{"9lives", "N_9lives"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, SanitizeEnumMember(tt.input))
		})
	}
}
