package zod

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/mirpo/schemagen/pkg/common"
	"github.com/mirpo/schemagen/pkg/lang/tscommon"
)

// escapeRegex escapes special characters in a regex pattern for use in JavaScript.
// It also handles forward slashes which are regex delimiters in JS.
func escapeRegex(pattern string) string {
	// Escape forward slashes for JS regex literal syntax
	return strings.ReplaceAll(pattern, "/", "\\/")
}

func formatLiteral(v interface{}) string {
	switch val := v.(type) {
	case map[string]interface{}:
		return formatJSObject(val)
	case []interface{}:
		return formatJSArray(val)
	default:
		return common.TSLiterals.FormatValue(v)
	}
}

// formatJSObject formats a Go map as a JavaScript object literal.
func formatJSObject(m map[string]interface{}) string {
	if len(m) == 0 {
		return "{}"
	}

	var parts []string
	for _, k := range slices.Sorted(maps.Keys(m)) {
		var key string
		if tscommon.NeedsQuoting(k) {
			key = fmt.Sprintf("%q", k)
		} else {
			key = k
		}
		parts = append(parts, fmt.Sprintf("%s: %s", key, formatLiteral(m[k])))
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

// formatZodLiteral formats a value as a Zod schema for use in enum unions.
// For primitives, it wraps in z.literal(). For complex values (objects/arrays),
// it generates proper Zod structures since z.literal() only accepts primitives.
func formatZodLiteral(v interface{}) string {
	switch val := v.(type) {
	case map[string]interface{}:
		return formatZodObject(val)
	case []interface{}:
		return formatZodTuple(val)
	default:
		// Primitives use z.literal()
		return fmt.Sprintf("z.literal(%s)", formatLiteral(v))
	}
}

// formatZodObject formats a Go map as a Zod object schema with literal values.
func formatZodObject(m map[string]interface{}) string {
	if len(m) == 0 {
		return "z.object({}).strict()"
	}

	var parts []string
	for _, k := range slices.Sorted(maps.Keys(m)) {
		var key string
		if tscommon.NeedsQuoting(k) {
			key = fmt.Sprintf("%q", k)
		} else {
			key = k
		}
		parts = append(parts, fmt.Sprintf("%s: %s", key, formatZodLiteral(m[k])))
	}
	return fmt.Sprintf("z.object({ %s }).strict()", strings.Join(parts, ", "))
}

// formatZodTuple formats a Go slice as a Zod tuple schema with literal values.
func formatZodTuple(arr []interface{}) string {
	if len(arr) == 0 {
		return "z.tuple([])"
	}

	var parts []string
	for _, v := range arr {
		parts = append(parts, formatZodLiteral(v))
	}
	return fmt.Sprintf("z.tuple([%s])", strings.Join(parts, ", "))
}
