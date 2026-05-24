package py

import (
	"fmt"

	"github.com/mirpo/schemagen/pkg/graph"
)

// needsField reports whether a Pydantic Field() wrapper is required for the given field.
// needsAlias is true when the Python field name differs from the JSON key (e.g. snake_case
// conversion or Python-keyword escaping) and an alias= parameter must be emitted.
func needsField(field *graph.Field, needsAlias bool) bool {
	return field.Description != "" || field.HasConstraints() || needsAlias
}

// buildFieldParams generates Pydantic Field() parameters for a field.
// Returns empty slice if no Field() is needed (simple field).
// Returns non-empty slice with all constraint parameters if Field() is required.
func (g *Generator) buildFieldParams(field *graph.Field, required bool, needsAlias bool, jsonName string) []string {
	var params []string

	if !needsField(field, needsAlias) {
		return []string{} // Empty slice = no Field() needed
	}

	if required {
		params = append(params, "...")
	} else {
		params = append(params, "None")
	}

	if needsAlias {
		params = append(params, fmt.Sprintf("alias=%q", jsonName))
	}

	if field.Description != "" {
		params = append(params, fmt.Sprintf("description=%s", formatPythonString(field.Description)))
	}

	if field.MinLength != nil {
		params = append(params, fmt.Sprintf("min_length=%d", *field.MinLength))
	}
	if field.MaxLength != nil {
		params = append(params, fmt.Sprintf("max_length=%d", *field.MaxLength))
	}
	if field.Pattern != nil {
		params = append(params, fmt.Sprintf("pattern=%q", *field.Pattern))
	}

	if field.Minimum != nil {
		params = append(params, fmt.Sprintf("ge=%g", *field.Minimum))
	}
	if field.Maximum != nil {
		params = append(params, fmt.Sprintf("le=%g", *field.Maximum))
	}
	if field.ExclusiveMinimum != nil {
		params = append(params, fmt.Sprintf("gt=%g", *field.ExclusiveMinimum))
	}
	if field.ExclusiveMaximum != nil {
		params = append(params, fmt.Sprintf("lt=%g", *field.ExclusiveMaximum))
	}

	if field.MultipleOf != nil {
		params = append(params, fmt.Sprintf("multiple_of=%g", *field.MultipleOf))
	}

	if field.MinItems != nil && field.MinLength == nil {
		params = append(params, fmt.Sprintf("min_length=%d", *field.MinItems))
	}
	if field.MaxItems != nil && field.MaxLength == nil {
		params = append(params, fmt.Sprintf("max_length=%d", *field.MaxItems))
	}

	return params
}
