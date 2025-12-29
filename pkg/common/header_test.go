package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateHeader(t *testing.T) {
	tests := []struct {
		name            string
		cfg             HeaderConfig
		expectEmpty     bool
		expectPrefix    string
		expectTimestamp bool
		expectDoNotEdit bool
	}{
		{
			name: "TypeScript with timestamp",
			cfg: HeaderConfig{
				CommentPrefix:    CommentPrefixTypeScript,
				DisableHeaders:   false,
				DisableTimestamp: false,
			},
			expectEmpty:     false,
			expectPrefix:    "//",
			expectTimestamp: true,
			expectDoNotEdit: true,
		},
		{
			name: "Python with timestamp",
			cfg: HeaderConfig{
				CommentPrefix:    CommentPrefixPython,
				DisableHeaders:   false,
				DisableTimestamp: false,
			},
			expectEmpty:     false,
			expectPrefix:    "#",
			expectTimestamp: true,
			expectDoNotEdit: true,
		},
		{
			name: "Go with timestamp",
			cfg: HeaderConfig{
				CommentPrefix:    CommentPrefixGo,
				DisableHeaders:   false,
				DisableTimestamp: false,
			},
			expectEmpty:     false,
			expectPrefix:    "//",
			expectTimestamp: true,
			expectDoNotEdit: true,
		},
		{
			name: "TypeScript without timestamp",
			cfg: HeaderConfig{
				CommentPrefix:    CommentPrefixTypeScript,
				DisableHeaders:   false,
				DisableTimestamp: true,
			},
			expectEmpty:     false,
			expectPrefix:    "//",
			expectTimestamp: false,
			expectDoNotEdit: true,
		},
		{
			name: "Python without timestamp",
			cfg: HeaderConfig{
				CommentPrefix:    CommentPrefixPython,
				DisableHeaders:   false,
				DisableTimestamp: true,
			},
			expectEmpty:     false,
			expectPrefix:    "#",
			expectTimestamp: false,
			expectDoNotEdit: true,
		},
		{
			name: "Headers disabled",
			cfg: HeaderConfig{
				CommentPrefix:    CommentPrefixTypeScript,
				DisableHeaders:   true,
				DisableTimestamp: false,
			},
			expectEmpty:     true,
			expectPrefix:    "",
			expectTimestamp: false,
			expectDoNotEdit: false,
		},
		{
			name: "Headers disabled overrides timestamp setting",
			cfg: HeaderConfig{
				CommentPrefix:    CommentPrefixPython,
				DisableHeaders:   true,
				DisableTimestamp: true,
			},
			expectEmpty:     true,
			expectPrefix:    "",
			expectTimestamp: false,
			expectDoNotEdit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateHeader(tt.cfg)

			if tt.expectEmpty {
				assert.Empty(t, result, "GenerateHeader() should return empty string")
				return
			}

			assert.NotEmpty(t, result, "GenerateHeader() should return header content")

			if tt.expectPrefix != "" {
				assert.Contains(t, result, tt.expectPrefix, "GenerateHeader() should contain expected prefix")
			}

			if tt.expectDoNotEdit {
				assert.Contains(t, result, "DO NOT EDIT", "GenerateHeader() should contain 'DO NOT EDIT'")
			}

			if tt.expectTimestamp {
				assert.Contains(t, result, "timestamp:", "GenerateHeader() should contain 'timestamp:'")
			} else {
				assert.NotContains(t, result, "timestamp:", "GenerateHeader() should not contain 'timestamp:'")
			}

			assert.True(t, strings.HasSuffix(result, "\n"), "GenerateHeader() should end with newline")
		})
	}
}

func TestGenerateHeaderTimestampFormat(t *testing.T) {
	cfg := HeaderConfig{
		CommentPrefix:    CommentPrefixTypeScript,
		DisableHeaders:   false,
		DisableTimestamp: false,
	}

	result := GenerateHeader(cfg)

	timestampIndex := strings.Index(result, "timestamp:")
	require.NotEqual(t, -1, timestampIndex, "GenerateHeader() should contain timestamp")

	timestampLine := result[timestampIndex:]
	endOfLine := strings.Index(timestampLine, "\n")
	require.NotEqual(t, -1, endOfLine, "Should find end of timestamp line")
	timestampValue := timestampLine[len("timestamp: "):endOfLine]

	assert.NotEmpty(t, timestampValue, "Timestamp value should not be empty")
	assert.True(t, strings.Contains(timestampValue, "T") && strings.Contains(timestampValue, "Z"),
		"Timestamp should be in RFC3339 format (contain T and Z)")
}

func TestCommentPrefixValues(t *testing.T) {
	tests := []struct {
		name     string
		prefix   CommentPrefix
		expected string
	}{
		{
			name:     "TypeScript prefix",
			prefix:   CommentPrefixTypeScript,
			expected: "//",
		},
		{
			name:     "Python prefix",
			prefix:   CommentPrefixPython,
			expected: "#",
		},
		{
			name:     "Go prefix",
			prefix:   CommentPrefixGo,
			expected: "//",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.prefix), "CommentPrefix should match expected value")
		})
	}
}
