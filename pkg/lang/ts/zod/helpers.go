package zod

import (
	"fmt"
	"strings"
	"unicode"
)

// needsQuoting returns true if the property name needs to be quoted in JS/TS.
func needsQuoting(name string) bool {
	if len(name) == 0 {
		return true
	}

	// Must start with letter, underscore, or dollar sign
	first := rune(name[0])
	if !unicode.IsLetter(first) && first != '_' && first != '$' {
		return true
	}

	// Rest must be alphanumeric, underscore, or dollar sign
	for _, r := range name[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '$' {
			return true
		}
	}

	// Check for reserved words
	reserved := map[string]bool{
		"break": true, "case": true, "catch": true, "continue": true,
		"debugger": true, "default": true, "delete": true, "do": true,
		"else": true, "finally": true, "for": true, "function": true,
		"if": true, "in": true, "instanceof": true, "new": true,
		"return": true, "switch": true, "this": true, "throw": true,
		"try": true, "typeof": true, "var": true, "void": true,
		"while": true, "with": true, "class": true, "const": true,
		"enum": true, "export": true, "extends": true, "import": true,
		"super": true, "implements": true, "interface": true, "let": true,
		"package": true, "private": true, "protected": true, "public": true,
		"static": true, "yield": true,
	}

	return reserved[name]
}

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
		if needsQuoting(k) {
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
