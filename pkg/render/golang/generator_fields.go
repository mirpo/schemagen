package golang

import (
	"fmt"
	"strings"

	"github.com/mirpo/schemagen/pkg/graph"
)

func (g *Generator) fieldJSONTag(field *graph.Field) string {
	tag := field.JSONName

	// Add omitempty for optional fields if configured
	if !field.Required && g.config.OmitEmpty {
		tag += ",omitempty"
	}

	return tag
}

func (g *Generator) fieldValidateTag(field *graph.Field) string {
	var constraints []string

	// Required field
	if field.Required {
		constraints = append(constraints, "required")
	}

	// String constraints
	if field.MinLength != nil {
		constraints = append(constraints, fmt.Sprintf("min=%d", *field.MinLength))
	}
	if field.MaxLength != nil {
		constraints = append(constraints, fmt.Sprintf("max=%d", *field.MaxLength))
	}

	// Number constraints
	if field.Minimum != nil {
		constraints = append(constraints, fmt.Sprintf("gte=%g", *field.Minimum))
	}
	if field.Maximum != nil {
		constraints = append(constraints, fmt.Sprintf("lte=%g", *field.Maximum))
	}
	if field.ExclusiveMinimum != nil {
		constraints = append(constraints, fmt.Sprintf("gt=%g", *field.ExclusiveMinimum))
	}
	if field.ExclusiveMaximum != nil {
		constraints = append(constraints, fmt.Sprintf("lt=%g", *field.ExclusiveMaximum))
	}

	// Array constraints (dive for nested validation)
	if field.MinItems != nil {
		constraints = append(constraints, fmt.Sprintf("min=%d", *field.MinItems))
	}
	if field.MaxItems != nil {
		constraints = append(constraints, fmt.Sprintf("max=%d", *field.MaxItems))
	}

	// Email/URL validation from format
	if field.Type != nil && field.Type.Format != "" {
		switch field.Type.Format {
		case graph.FormatEmail:
			constraints = append(constraints, "email")
		case graph.FormatURI, graph.FormatURL:
			constraints = append(constraints, "url")
		case graph.FormatUUID:
			constraints = append(constraints, "uuid")
		}
	}

	// Inline enum validation (oneof)
	if field.Type != nil && len(field.Type.EnumValues) > 0 {
		// Build oneof constraint from enum values
		enumVals := make([]string, 0, len(field.Type.EnumValues))
		for _, val := range field.Type.EnumValues {
			switch v := val.Value.(type) {
			case string:
				enumVals = append(enumVals, v)
			case float64, int, int64:
				enumVals = append(enumVals, fmt.Sprintf("%v", v))
			}
		}
		if len(enumVals) > 0 {
			constraints = append(constraints, fmt.Sprintf("oneof=%s", strings.Join(enumVals, " ")))
		}
	}

	if len(constraints) == 0 {
		return ""
	}

	return strings.Join(constraints, ",")
}
