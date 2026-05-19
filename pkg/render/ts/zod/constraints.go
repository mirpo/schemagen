package zod

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/graph"
)

type stringFormatters struct {
	MinLength func(int) string
	MaxLength func(int) string
	Pattern   func(string) string
}

type numberFormatters struct {
	Min          func(float64) string
	Max          func(float64) string
	ExclusiveMin func(float64) string
	ExclusiveMax func(float64) string
}

type arrayFormatters struct {
	MinItems func(int) string
	MaxItems func(int) string
}

var zodStringFmt = stringFormatters{
	MinLength: func(n int) string { return fmt.Sprintf("min(%d)", n) },
	MaxLength: func(n int) string { return fmt.Sprintf("max(%d)", n) },
	Pattern:   func(p string) string { return fmt.Sprintf("regex(/%s/)", escapeRegex(p)) },
}

var zodNumberFmt = numberFormatters{
	Min:          func(n float64) string { return fmt.Sprintf("gte(%v)", formatNumber(n)) },
	Max:          func(n float64) string { return fmt.Sprintf("lte(%v)", formatNumber(n)) },
	ExclusiveMin: func(n float64) string { return fmt.Sprintf("gt(%v)", formatNumber(n)) },
	ExclusiveMax: func(n float64) string { return fmt.Sprintf("lt(%v)", formatNumber(n)) },
}

var zodArrayFmt = arrayFormatters{
	MinItems: func(n int) string { return fmt.Sprintf("min(%d)", n) },
	MaxItems: func(n int) string { return fmt.Sprintf("max(%d)", n) },
}

func buildStringConstraints(field *graph.Field) []string {
	if field == nil {
		return nil
	}
	var result []string
	if field.MinLength != nil {
		result = append(result, zodStringFmt.MinLength(*field.MinLength))
	}
	if field.MaxLength != nil {
		result = append(result, zodStringFmt.MaxLength(*field.MaxLength))
	}
	if field.Pattern != nil {
		result = append(result, zodStringFmt.Pattern(*field.Pattern))
	}
	return result
}

func buildNumberConstraints(field *graph.Field) []string {
	if field == nil {
		return nil
	}
	var result []string
	if field.Minimum != nil {
		result = append(result, zodNumberFmt.Min(*field.Minimum))
	}
	if field.Maximum != nil {
		result = append(result, zodNumberFmt.Max(*field.Maximum))
	}
	if field.ExclusiveMinimum != nil {
		result = append(result, zodNumberFmt.ExclusiveMin(*field.ExclusiveMinimum))
	}
	if field.ExclusiveMaximum != nil {
		result = append(result, zodNumberFmt.ExclusiveMax(*field.ExclusiveMaximum))
	}
	return result
}

func buildArrayConstraints(field *graph.Field) []string {
	if field == nil {
		return nil
	}
	var result []string
	if field.MinItems != nil {
		result = append(result, zodArrayFmt.MinItems(*field.MinItems))
	}
	if field.MaxItems != nil {
		result = append(result, zodArrayFmt.MaxItems(*field.MaxItems))
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
