package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeRelativeImport_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		expected string
	}{
		{
			name:     "same directory",
			from:     "events/event.ts",
			to:       "events/header.ts",
			expected: "./header",
		},
		{
			name:     "subdirectory",
			from:     "events/event.ts",
			to:       "events/payloads/subscribe.ts",
			expected: "./payloads/subscribe",
		},
		{
			name:     "parent directory",
			from:     "events/payloads/subscribe.ts",
			to:       "events/header.ts",
			expected: "../header",
		},
		{
			name:     "sibling directory",
			from:     "api/users/user.ts",
			to:       "api/auth/token.ts",
			expected: "../auth/token",
		},
		{
			name:     "deep nesting",
			from:     "a/b/c/d.ts",
			to:       "a/e/f.ts",
			expected: "../../e/f",
		},
		{
			name:     "root files",
			from:     "a.ts",
			to:       "b.ts",
			expected: "./b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeRelativeImport(tt.from, tt.to)
			assert.Equal(t, tt.expected, result)
		})
	}
}
