package zod

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/mirpo/schemagen/pkg/common"
	"github.com/mirpo/schemagen/pkg/lang/tscommon"
)

func escapeRegex(pattern string) string {
	return strings.ReplaceAll(pattern, "/", "\\/")
}

func formatLiteral(v any) string {
	switch val := v.(type) {
	case map[string]any:
		return formatJSObject(val)
	case []any:
		return formatJSArray(val)
	default:
		return common.TSLiterals.FormatValue(v)
	}
}

func formatJSObject(m map[string]any) string {
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

func formatJSArray(arr []any) string {
	if len(arr) == 0 {
		return "[]"
	}

	var parts []string
	for _, v := range arr {
		parts = append(parts, formatLiteral(v))
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}

func formatZodLiteral(v any) string {
	switch val := v.(type) {
	case map[string]any:
		return formatZodObject(val)
	case []any:
		return formatZodTuple(val)
	default:
		return fmt.Sprintf("z.literal(%s)", formatLiteral(v))
	}
}

func formatZodObject(m map[string]any) string {
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

func formatZodTuple(arr []any) string {
	if len(arr) == 0 {
		return "z.tuple([])"
	}

	var parts []string
	for _, v := range arr {
		parts = append(parts, formatZodLiteral(v))
	}
	return fmt.Sprintf("z.tuple([%s])", strings.Join(parts, ", "))
}
