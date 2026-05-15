package typegraph

import (
	"maps"
	"slices"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/schema"
)

func buildRequiredMap(required []string) map[string]bool {
	m := make(map[string]bool, len(required))
	for _, r := range required {
		m[r] = true
	}
	return m
}

func extractConstraints(field *Field, sch *jsonschema.Schema) {
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

func getOrderedPropertyNames(properties *jsonschema.SchemaMap, schemaPath string, order *schema.PropertyOrder) []string {
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
				extra := slices.Sorted(maps.Keys(mapKeys))
				result = append(result, extra...)
			}

			return result
		}
	}

	// Fallback to alphabetical sorting for backward compatibility
	names := slices.Sorted(maps.Keys(*properties))
	return names
}

func buildFieldsFromSchema(ctx *buildContext, sch *jsonschema.Schema, orderPath string, buildTypeRef func(*buildContext, *jsonschema.Schema, string) *TypeRef) []*Field {
	if sch.Properties == nil {
		return nil
	}

	requiredMap := buildRequiredMap(sch.Required)

	fields := make([]*Field, 0, len(*sch.Properties))

	for _, propName := range getOrderedPropertyNames(sch.Properties, orderPath, ctx.Order) {
		propSchema := (*sch.Properties)[propName]
		field := &Field{
			JSONName:    propName,
			Description: getDescription(propSchema),
			Required:    requiredMap[propName],
			Type:        buildTypeRef(ctx, propSchema, propName),
		}
		extractConstraints(field, propSchema)
		fields = append(fields, field)
	}

	return fields
}

type typeRefBuilderFunc func(sch *jsonschema.Schema, fieldName string) *TypeRef

func extractAdditionalProperties(sch *jsonschema.Schema, buildTypeRef typeRefBuilderFunc) *AdditionalPropsConfig {
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
