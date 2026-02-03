package zod

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/constraints"
	"github.com/mirpo/schemagen/pkg/typegraph"
)

// Zod-specific formatters
var zodStringFormatters = constraints.StringFormatters{
	MinLength: func(n int) string { return fmt.Sprintf("min(%d)", n) },
	MaxLength: func(n int) string { return fmt.Sprintf("max(%d)", n) },
	Pattern:   func(p string) string { return fmt.Sprintf("regex(/%s/)", escapeRegex(p)) },
}

var zodNumberFormatters = constraints.NumberFormatters{
	Min:          func(n float64) string { return fmt.Sprintf("gte(%v)", formatNumber(n)) },
	Max:          func(n float64) string { return fmt.Sprintf("lte(%v)", formatNumber(n)) },
	ExclusiveMin: func(n float64) string { return fmt.Sprintf("gt(%v)", formatNumber(n)) },
	ExclusiveMax: func(n float64) string { return fmt.Sprintf("lt(%v)", formatNumber(n)) },
}

var zodArrayFormatters = constraints.ArrayFormatters{
	MinItems: func(n int) string { return fmt.Sprintf("min(%d)", n) },
	MaxItems: func(n int) string { return fmt.Sprintf("max(%d)", n) },
}

// stringConstraints generates Zod string validation constraints.
func stringConstraints(field *typegraph.Field) string {
	result := constraints.BuildStringConstraints(field, zodStringFormatters)
	if len(result) == 0 {
		return ""
	}
	return "." + strings.Join(result, ".")
}

// numberConstraints generates Zod number validation constraints.
func numberConstraints(field *typegraph.Field) string {
	result := constraints.BuildNumberConstraints(field, zodNumberFormatters)
	if len(result) == 0 {
		return ""
	}
	return "." + strings.Join(result, ".")
}

// arrayConstraints generates Zod array validation constraints.
func arrayConstraints(field *typegraph.Field) string {
	result := constraints.BuildArrayConstraints(field, zodArrayFormatters)
	if len(result) == 0 {
		return ""
	}
	return "." + strings.Join(result, ".")
}

// formatNumber formats a float64 for JavaScript output.
// If the number is a whole number, it outputs without decimal point.
func formatNumber(n float64) string {
	if n == float64(int64(n)) {
		return fmt.Sprintf("%d", int64(n))
	}
	return fmt.Sprintf("%v", n)
}
