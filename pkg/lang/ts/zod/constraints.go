package zod

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/typegraph"
)

// stringConstraints generates Zod string validation constraints.
func stringConstraints(field *typegraph.Field) string {
	if field == nil {
		return ""
	}

	var constraints []string

	if field.MinLength != nil {
		constraints = append(constraints, fmt.Sprintf("min(%d)", *field.MinLength))
	}
	if field.MaxLength != nil {
		constraints = append(constraints, fmt.Sprintf("max(%d)", *field.MaxLength))
	}
	if field.Pattern != nil {
		constraints = append(constraints, fmt.Sprintf("regex(/%s/)", escapeRegex(*field.Pattern)))
	}

	if len(constraints) == 0 {
		return ""
	}
	return "." + strings.Join(constraints, ".")
}

// numberConstraints generates Zod number validation constraints.
func numberConstraints(field *typegraph.Field) string {
	if field == nil {
		return ""
	}

	var constraints []string

	if field.Minimum != nil {
		constraints = append(constraints, fmt.Sprintf("gte(%v)", formatNumber(*field.Minimum)))
	}
	if field.Maximum != nil {
		constraints = append(constraints, fmt.Sprintf("lte(%v)", formatNumber(*field.Maximum)))
	}
	if field.ExclusiveMinimum != nil {
		constraints = append(constraints, fmt.Sprintf("gt(%v)", formatNumber(*field.ExclusiveMinimum)))
	}
	if field.ExclusiveMaximum != nil {
		constraints = append(constraints, fmt.Sprintf("lt(%v)", formatNumber(*field.ExclusiveMaximum)))
	}

	if len(constraints) == 0 {
		return ""
	}
	return "." + strings.Join(constraints, ".")
}

// arrayConstraints generates Zod array validation constraints.
func arrayConstraints(field *typegraph.Field) string {
	if field == nil {
		return ""
	}

	var constraints []string

	if field.MinItems != nil {
		constraints = append(constraints, fmt.Sprintf("min(%d)", *field.MinItems))
	}
	if field.MaxItems != nil {
		constraints = append(constraints, fmt.Sprintf("max(%d)", *field.MaxItems))
	}

	if len(constraints) == 0 {
		return ""
	}
	return "." + strings.Join(constraints, ".")
}

// formatNumber formats a float64 for JavaScript output.
// If the number is a whole number, it outputs without decimal point.
func formatNumber(n float64) string {
	if n == float64(int64(n)) {
		return fmt.Sprintf("%d", int64(n))
	}
	return fmt.Sprintf("%v", n)
}
