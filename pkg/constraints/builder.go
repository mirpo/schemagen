package constraints

import "github.com/mirpo/schemagen/pkg/typegraph"

// StringFormatters defines formatters for string validation constraints.
type StringFormatters struct {
	MinLength func(int) string
	MaxLength func(int) string
	Pattern   func(string) string
}

// NumberFormatters defines formatters for numeric validation constraints.
type NumberFormatters struct {
	Min          func(float64) string
	Max          func(float64) string
	ExclusiveMin func(float64) string
	ExclusiveMax func(float64) string
}

// ArrayFormatters defines formatters for array validation constraints.
type ArrayFormatters struct {
	MinItems func(int) string
	MaxItems func(int) string
}

// BuildStringConstraints extracts string constraints from a field using the provided formatters.
// Returns nil if field is nil or no constraints apply.
func BuildStringConstraints(field *typegraph.Field, fmt StringFormatters) []string {
	if field == nil {
		return nil
	}

	var constraints []string

	if field.MinLength != nil && fmt.MinLength != nil {
		constraints = append(constraints, fmt.MinLength(*field.MinLength))
	}
	if field.MaxLength != nil && fmt.MaxLength != nil {
		constraints = append(constraints, fmt.MaxLength(*field.MaxLength))
	}
	if field.Pattern != nil && fmt.Pattern != nil {
		constraints = append(constraints, fmt.Pattern(*field.Pattern))
	}

	return constraints
}

// BuildNumberConstraints extracts number constraints from a field using the provided formatters.
// Returns nil if field is nil or no constraints apply.
func BuildNumberConstraints(field *typegraph.Field, fmt NumberFormatters) []string {
	if field == nil {
		return nil
	}

	var constraints []string

	if field.Minimum != nil && fmt.Min != nil {
		constraints = append(constraints, fmt.Min(*field.Minimum))
	}
	if field.Maximum != nil && fmt.Max != nil {
		constraints = append(constraints, fmt.Max(*field.Maximum))
	}
	if field.ExclusiveMinimum != nil && fmt.ExclusiveMin != nil {
		constraints = append(constraints, fmt.ExclusiveMin(*field.ExclusiveMinimum))
	}
	if field.ExclusiveMaximum != nil && fmt.ExclusiveMax != nil {
		constraints = append(constraints, fmt.ExclusiveMax(*field.ExclusiveMaximum))
	}

	return constraints
}

// BuildArrayConstraints extracts array constraints from a field using the provided formatters.
// Returns nil if field is nil or no constraints apply.
func BuildArrayConstraints(field *typegraph.Field, fmt ArrayFormatters) []string {
	if field == nil {
		return nil
	}

	var constraints []string

	if field.MinItems != nil && fmt.MinItems != nil {
		constraints = append(constraints, fmt.MinItems(*field.MinItems))
	}
	if field.MaxItems != nil && fmt.MaxItems != nil {
		constraints = append(constraints, fmt.MaxItems(*field.MaxItems))
	}

	return constraints
}

// HasConstraints checks if a field has any validation constraints.
func HasConstraints(field *typegraph.Field) bool {
	if field == nil {
		return false
	}
	return field.MinLength != nil || field.MaxLength != nil || field.Pattern != nil ||
		field.Minimum != nil || field.Maximum != nil ||
		field.ExclusiveMinimum != nil || field.ExclusiveMaximum != nil ||
		field.MinItems != nil || field.MaxItems != nil
}

// HasStringConstraints checks if a field has string validation constraints.
func HasStringConstraints(field *typegraph.Field) bool {
	if field == nil {
		return false
	}
	return field.MinLength != nil || field.MaxLength != nil || field.Pattern != nil
}

// HasNumberConstraints checks if a field has numeric validation constraints.
func HasNumberConstraints(field *typegraph.Field) bool {
	if field == nil {
		return false
	}
	return field.Minimum != nil || field.Maximum != nil ||
		field.ExclusiveMinimum != nil || field.ExclusiveMaximum != nil
}

// HasArrayConstraints checks if a field has array validation constraints.
func HasArrayConstraints(field *typegraph.Field) bool {
	if field == nil {
		return false
	}
	return field.MinItems != nil || field.MaxItems != nil
}
