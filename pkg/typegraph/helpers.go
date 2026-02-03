package typegraph

import (
	"sort"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/schema"
)

// ExtractConstraints extracts validation constraints from a JSON schema into a field.
// This is a standalone helper to avoid circular dependencies between builders.
func ExtractConstraints(field *Field, sch *jsonschema.Schema) {
	// String constraints
	if sch.MinLength != nil {
		minLen := int(*sch.MinLength)
		field.MinLength = &minLen
	}
	if sch.MaxLength != nil {
		maxLen := int(*sch.MaxLength)
		field.MaxLength = &maxLen
	}
	if sch.Pattern != nil && *sch.Pattern != "" {
		pattern := *sch.Pattern
		field.Pattern = &pattern
	}

	// Number constraints
	if sch.Minimum != nil {
		if min, ok := sch.Minimum.Float64(); ok {
			field.Minimum = &min
		}
	}
	if sch.Maximum != nil {
		if max, ok := sch.Maximum.Float64(); ok {
			field.Maximum = &max
		}
	}
	if sch.ExclusiveMinimum != nil {
		if min, ok := sch.ExclusiveMinimum.Float64(); ok {
			field.ExclusiveMinimum = &min
		}
	}
	if sch.ExclusiveMaximum != nil {
		if max, ok := sch.ExclusiveMaximum.Float64(); ok {
			field.ExclusiveMaximum = &max
		}
	}

	// Array constraints
	if sch.MinItems != nil {
		minItems := int(*sch.MinItems)
		field.MinItems = &minItems
	}
	if sch.MaxItems != nil {
		maxItems := int(*sch.MaxItems)
		field.MaxItems = &maxItems
	}
}

// GetOrderedPropertyNames returns property names in schema file order.
// Falls back to alphabetical sorting if order information is not available.
// This is a standalone helper to avoid circular dependencies between builders.
func GetOrderedPropertyNames(properties *jsonschema.SchemaMap, schemaPath string, order *schema.PropertyOrder) []string {
	if properties == nil {
		return nil
	}

	// Try to get order from extracted property order
	if order != nil {
		if ordered := order.GetOrder(schemaPath); len(ordered) > 0 {
			// Filter to only include keys that exist in the properties map
			mapKeys := make(map[string]bool)
			for key := range *properties {
				mapKeys[key] = true
			}

			result := make([]string, 0, len(ordered))
			for _, key := range ordered {
				if mapKeys[key] {
					result = append(result, key)
					delete(mapKeys, key)
				}
			}

			// Add any keys not in order (shouldn't happen, but be safe)
			if len(mapKeys) > 0 {
				extra := make([]string, 0, len(mapKeys))
				for key := range mapKeys {
					extra = append(extra, key)
				}
				sort.Strings(extra)
				result = append(result, extra...)
			}

			return result
		}
	}

	// Fallback to alphabetical sorting for backward compatibility
	names := make([]string, 0, len(*properties))
	for name := range *properties {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// TypeRefBuilder callback type for building type references.
type TypeRefBuilderFunc func(sch *jsonschema.Schema, fieldName string) *TypeRef

// ExtractAdditionalProperties extracts additionalProperties config from a schema.
// Uses buildTypeRef callback to build the TypeRef for typed additional properties.
// This is a standalone helper to avoid circular dependencies between builders.
func ExtractAdditionalProperties(sch *jsonschema.Schema, buildTypeRef TypeRefBuilderFunc) *AdditionalPropsConfig {
	if sch.AdditionalProperties == nil {
		return nil
	}

	addlProps := sch.AdditionalProperties

	// Check if it's a boolean schema (additionalProperties: true or false)
	if addlProps.Boolean != nil {
		return &AdditionalPropsConfig{
			Allowed: *addlProps.Boolean,
			Type:    nil,
		}
	}

	// If it's a schema (not a boolean), additional properties are allowed and must match the schema
	return &AdditionalPropsConfig{
		Allowed: true,
		Type:    buildTypeRef(addlProps, "additionalProperty"),
	}
}
