package zod

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/lang/tscommon"
)

// escapeRegex escapes special characters in a regex pattern for use in JavaScript.
// It also handles forward slashes which are regex delimiters in JS.
func escapeRegex(pattern string) string {
	// Escape forward slashes for JS regex literal syntax
	return strings.ReplaceAll(pattern, "/", "\\/")
}

// formatLiteral formats a value as a JavaScript literal.
func formatLiteral(v interface{}) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case float64:
		// Check if it's a whole number
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%v", val)
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case nil:
		return "null"
	case map[string]interface{}:
		return formatJSObject(val)
	case []interface{}:
		return formatJSArray(val)
	default:
		// For unknown types, try to format as string
		return fmt.Sprintf("%q", fmt.Sprintf("%v", val))
	}
}

// formatJSObject formats a Go map as a JavaScript object literal.
func formatJSObject(m map[string]interface{}) string {
	if len(m) == 0 {
		return "{}"
	}

	var parts []string
	for k, v := range m {
		// Quote key if needed
		var key string
		if tscommon.NeedsQuoting(k) {
			key = fmt.Sprintf("%q", k)
		} else {
			key = k
		}
		parts = append(parts, fmt.Sprintf("%s: %s", key, formatLiteral(v)))
	}
	return fmt.Sprintf("{ %s }", strings.Join(parts, ", "))
}

// formatJSArray formats a Go slice as a JavaScript array literal.
func formatJSArray(arr []interface{}) string {
	if len(arr) == 0 {
		return "[]"
	}

	var parts []string
	for _, v := range arr {
		parts = append(parts, formatLiteral(v))
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}
