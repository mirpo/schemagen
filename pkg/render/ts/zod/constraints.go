package zod

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/graph"
)

func zodMinLength(n int) string      { return fmt.Sprintf("min(%d)", n) }
func zodMaxLength(n int) string      { return fmt.Sprintf("max(%d)", n) }
func zodPattern(p string) string     { return fmt.Sprintf("regex(/%s/)", escapeRegex(p)) }
func zodGte(n float64) string        { return fmt.Sprintf("gte(%v)", formatNumber(n)) }
func zodLte(n float64) string        { return fmt.Sprintf("lte(%v)", formatNumber(n)) }
func zodGt(n float64) string         { return fmt.Sprintf("gt(%v)", formatNumber(n)) }
func zodLt(n float64) string         { return fmt.Sprintf("lt(%v)", formatNumber(n)) }
func zodMultipleOf(n float64) string { return fmt.Sprintf("multipleOf(%v)", formatNumber(n)) }
func zodMinItems(n int) string       { return fmt.Sprintf("min(%d)", n) }
func zodMaxItems(n int) string       { return fmt.Sprintf("max(%d)", n) }

func buildStringConstraints(field *graph.Field) []string {
	if field == nil {
		return nil
	}
	var result []string
	if field.MinLength != nil {
		result = append(result, zodMinLength(*field.MinLength))
	}
	if field.MaxLength != nil {
		result = append(result, zodMaxLength(*field.MaxLength))
	}
	if field.Pattern != nil {
		result = append(result, zodPattern(*field.Pattern))
	}
	return result
}

func buildNumberConstraints(field *graph.Field) []string {
	if field == nil {
		return nil
	}
	var result []string
	if field.Minimum != nil {
		result = append(result, zodGte(*field.Minimum))
	}
	if field.Maximum != nil {
		result = append(result, zodLte(*field.Maximum))
	}
	if field.ExclusiveMinimum != nil {
		result = append(result, zodGt(*field.ExclusiveMinimum))
	}
	if field.ExclusiveMaximum != nil {
		result = append(result, zodLt(*field.ExclusiveMaximum))
	}
	if field.MultipleOf != nil {
		result = append(result, zodMultipleOf(*field.MultipleOf))
	}
	return result
}

func buildArrayConstraints(field *graph.Field) []string {
	if field == nil {
		return nil
	}
	var result []string
	if field.MinItems != nil {
		result = append(result, zodMinItems(*field.MinItems))
	}
	if field.MaxItems != nil {
		result = append(result, zodMaxItems(*field.MaxItems))
	}
	return result
}

func stringConstraints(field *graph.Field) string {
	result := buildStringConstraints(field)
	if len(result) == 0 {
		return ""
	}
	return "." + strings.Join(result, ".")
}

func numberConstraints(field *graph.Field) string {
	result := buildNumberConstraints(field)
	if len(result) == 0 {
		return ""
	}
	return "." + strings.Join(result, ".")
}

func arrayConstraints(field *graph.Field) string {
	result := buildArrayConstraints(field)
	if len(result) == 0 {
		return ""
	}
	return "." + strings.Join(result, ".")
}

func formatNumber(n float64) string {
	if n == float64(int64(n)) {
		return fmt.Sprintf("%d", int64(n))
	}
	return fmt.Sprintf("%v", n)
}
