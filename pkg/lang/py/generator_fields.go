package py

import (
	"fmt"

	"github.com/mirpo/schemagen/pkg/typegraph"
)

// buildFieldParams generates Pydantic Field() parameters for a field.
// Returns empty slice if no Field() is needed (simple field).
// Returns non-empty slice with all constraint parameters if Field() is required.
func (g *Generator) buildFieldParams(field *typegraph.Field, required bool, needsAlias bool, jsonName string) []string {
	var params []string

	// Determine if Field() is needed
	needsField := field.Description != "" ||
		field.MinLength != nil || field.MaxLength != nil ||
		field.Pattern != nil ||
		field.Minimum != nil || field.Maximum != nil ||
		field.ExclusiveMinimum != nil || field.ExclusiveMaximum != nil ||
		field.MinItems != nil || field.MaxItems != nil ||
		needsAlias

	if !needsField {
		return []string{} // Empty slice = no Field() needed
	}

	// Note: pydantic_field import is set during import scanning phase in GenerateFile()

	// 1. Default value (required or optional)
	if required {
		params = append(params, "...")
	} else {
		params = append(params, "None")
	}

	// 2. Alias if needed
	if needsAlias {
		params = append(params, fmt.Sprintf("alias=%q", jsonName))
	}

	// 3. Description
	if field.Description != "" {
		params = append(params, fmt.Sprintf("description=%s", formatPythonString(field.Description)))
	}

	// 4. String constraints
	if field.MinLength != nil {
		params = append(params, fmt.Sprintf("min_length=%d", *field.MinLength))
	}
	if field.MaxLength != nil {
		params = append(params, fmt.Sprintf("max_length=%d", *field.MaxLength))
	}
	if field.Pattern != nil {
		params = append(params, fmt.Sprintf("pattern=%q", *field.Pattern))
	}

	// 5. Numeric constraints
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

	// 6. Array constraints (using min/max_length for list items)
	if field.MinItems != nil {
		params = append(params, fmt.Sprintf("min_length=%d", *field.MinItems))
	}
	if field.MaxItems != nil {
		params = append(params, fmt.Sprintf("max_length=%d", *field.MaxItems))
	}

	return params
}
