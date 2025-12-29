package typegraph

import (
	"fmt"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/naming"
)

// buildStruct builds a struct/object type from a schema.
func (b *Builder) buildStruct(typ *Type, schema *jsonschema.Schema) error {
	typ.Kind = KindStruct
	typ.Fields = make([]*Field, 0)
	typ.Extends = make([]string, 0)

	// Track required fields
	requiredMap := make(map[string]bool)
	if schema.Required != nil {
		for _, req := range schema.Required {
			requiredMap[req] = true
		}
	}

	// Handle allOf composition
	if len(schema.AllOf) > 0 {
		for allOfIndex, allOfSchema := range schema.AllOf {
			// Check if this is a $ref to another type
			if allOfSchema.Ref != "" {
				// Extract type name from $ref
				typeName := b.extractTypeNameFromRef(allOfSchema.Ref)

				if typeName != "" {
					typ.Extends = append(typ.Extends, typeName)
				}
				// Don't add fields from extended types - they'll be inherited
				continue
			}

			// Merge required fields from allOf BEFORE processing properties
			if allOfSchema.Required != nil {
				for _, req := range allOfSchema.Required {
					requiredMap[req] = true
				}
			}

			// For inline allOf schemas, merge their properties
			if allOfSchema.Properties != nil {
				// Construct path for this allOf branch
				allOfPath := fmt.Sprintf("%s#/allOf/%d", b.currentPath, allOfIndex)
				for _, propName := range b.getOrderedPropertyNames(allOfSchema.Properties, allOfPath) {
					propSchema := (*allOfSchema.Properties)[propName]
					field := &Field{
						Name:        naming.ToPascalCase(propName),
						JSONName:    propName,
						Description: getDescription(propSchema),
						Required:    requiredMap[propName],
						OmitEmpty:   !requiredMap[propName],
						Type:        b.buildTypeRef(propSchema, propName),
					}
					// Extract validation constraints
					b.extractConstraints(field, propSchema)
					typ.Fields = append(typ.Fields, field)
				}
			}
		}
	}

	// Extract properties from the main schema
	if schema.Properties != nil {
		// Iterate over properties in schema file order
		for _, propName := range b.getOrderedPropertyNames(schema.Properties, b.currentPath) {
			propSchema := (*schema.Properties)[propName]
			field := &Field{
				Name:        naming.ToPascalCase(propName),
				JSONName:    propName,
				Description: getDescription(propSchema),
				Required:    requiredMap[propName],
				OmitEmpty:   !requiredMap[propName],
				Type:        b.buildTypeRef(propSchema, propName),
			}
			// Extract validation constraints
			b.extractConstraints(field, propSchema)
			typ.Fields = append(typ.Fields, field)
		}
	}

	// Deduplicate fields (in case allOf branches define the same field)
	typ.Fields = b.deduplicateFields(typ.Fields)

	// Capture additionalProperties configuration
	typ.AdditionalProps = b.extractAdditionalProperties(schema)

	return nil
}

// deduplicateFields removes duplicate fields, keeping the most specific one.
func (b *Builder) deduplicateFields(fields []*Field) []*Field {
	seen := make(map[string]*Field)
	result := make([]*Field, 0, len(fields))

	for _, field := range fields {
		if existing, found := seen[field.JSONName]; found {
			// Field already exists - merge or pick the better one
			// Strategy: Keep the first non-interface type, or the required one
			if existing.Type != nil && existing.Type.Kind == KindInterface && field.Type != nil && field.Type.Kind != KindInterface {
				// Replace with more specific type
				seen[field.JSONName] = field
			} else if !existing.Required && field.Required {
				// Keep the required version
				seen[field.JSONName] = field
			}
			// Otherwise keep the existing one
		} else {
			seen[field.JSONName] = field
		}
	}

	// Rebuild array in original order (approximately)
	for _, field := range fields {
		if seen[field.JSONName] == field {
			result = append(result, field)
			delete(seen, field.JSONName) // Mark as added
		}
	}

	return result
}

// extractConstraints extracts validation constraints from a schema into a field.
func (b *Builder) extractConstraints(field *Field, schema *jsonschema.Schema) {
	// String constraints
	if schema.MinLength != nil {
		minLen := int(*schema.MinLength)
		field.MinLength = &minLen
	}
	if schema.MaxLength != nil {
		maxLen := int(*schema.MaxLength)
		field.MaxLength = &maxLen
	}
	if schema.Pattern != nil && *schema.Pattern != "" {
		pattern := *schema.Pattern
		field.Pattern = &pattern
	}

	// Number constraints
	if schema.Minimum != nil {
		if min, ok := schema.Minimum.Float64(); ok {
			field.Minimum = &min
		}
	}
	if schema.Maximum != nil {
		if max, ok := schema.Maximum.Float64(); ok {
			field.Maximum = &max
		}
	}
	if schema.ExclusiveMinimum != nil {
		if min, ok := schema.ExclusiveMinimum.Float64(); ok {
			field.ExclusiveMinimum = &min
		}
	}
	if schema.ExclusiveMaximum != nil {
		if max, ok := schema.ExclusiveMaximum.Float64(); ok {
			field.ExclusiveMaximum = &max
		}
	}

	// Array constraints
	if schema.MinItems != nil {
		minItems := int(*schema.MinItems)
		field.MinItems = &minItems
	}
	if schema.MaxItems != nil {
		maxItems := int(*schema.MaxItems)
		field.MaxItems = &maxItems
	}
}

// extractAdditionalProperties captures additionalProperties configuration from a schema.
func (b *Builder) extractAdditionalProperties(schema *jsonschema.Schema) *AdditionalPropsConfig {
	// Check if additionalProperties is explicitly set
	if schema.AdditionalProperties == nil {
		return nil
	}

	addlProps := schema.AdditionalProperties

	// Check if it's a boolean schema (additionalProperties: true or false)
	if addlProps.Boolean != nil {
		return &AdditionalPropsConfig{
			Allowed: *addlProps.Boolean,
			Type:    nil,
		}
	}

	// If it's a schema (not a boolean), additional properties are allowed and must match the schema
	config := &AdditionalPropsConfig{
		Allowed: true,
		Type:    b.buildTypeRef(addlProps, "additionalProperty"),
	}

	return config
}

// shouldExtractInlineObject checks if a schema represents an inline object that should be extracted.
func (b *Builder) shouldExtractInlineObject(schema *jsonschema.Schema) bool {
	// Don't extract if it's a $ref (already a named type)
	if schema.Ref != "" {
		return false
	}

	// Extract if it's an object with properties
	if len(schema.Type) > 0 && schema.Type[0] == "object" {
		return schema.Properties != nil && len(*schema.Properties) > 0
	}

	return false
}

// extractInlineObjectType extracts an inline object as a separate Type.
func (b *Builder) extractInlineObjectType(baseName string, schema *jsonschema.Schema) *Type {
	typ := &Type{
		ID:          b.nextTypeID(),
		Name:        baseName,
		Kind:        KindStruct,
		Description: getDescription(schema),
		Fields:      make([]*Field, 0),
	}

	// Build required map
	requiredMap := make(map[string]bool)
	if schema.Required != nil {
		for _, req := range schema.Required {
			requiredMap[req] = true
		}
	}

	// Extract properties
	if schema.Properties != nil {
		// Inline objects don't have pre-extracted order info, so use empty path (falls back to alphabetical)
		for _, propName := range b.getOrderedPropertyNames(schema.Properties, "") {
			propSchema := (*schema.Properties)[propName]
			field := &Field{
				Name:        naming.ToPascalCase(propName),
				JSONName:    propName,
				Description: getDescription(propSchema),
				Required:    requiredMap[propName],
				OmitEmpty:   !requiredMap[propName],
				Type:        b.buildTypeRef(propSchema, propName),
			}
			// Extract validation constraints
			b.extractConstraints(field, propSchema)
			typ.Fields = append(typ.Fields, field)
		}
	}

	// Capture additionalProperties configuration
	typ.AdditionalProps = b.extractAdditionalProperties(schema)

	return typ
}

// buildFieldsFromProperties extracts fields from schema properties for inline objects.
// This is used to populate ObjectFields in TypeRef for anonymous interfaces.
func (b *Builder) buildFieldsFromProperties(schema *jsonschema.Schema, orderPath string) []*Field {
	if schema.Properties == nil {
		return nil
	}

	// Build required map
	requiredMap := make(map[string]bool)
	if schema.Required != nil {
		for _, req := range schema.Required {
			requiredMap[req] = true
		}
	}

	fields := make([]*Field, 0)

	// Iterate over properties in order
	for _, propName := range b.getOrderedPropertyNames(schema.Properties, orderPath) {
		propSchema := (*schema.Properties)[propName]
		field := &Field{
			Name:        naming.ToPascalCase(propName),
			JSONName:    propName,
			Description: getDescription(propSchema),
			Required:    requiredMap[propName],
			OmitEmpty:   !requiredMap[propName],
			Type:        b.buildTypeRef(propSchema, propName),
		}
		// Extract validation constraints
		b.extractConstraints(field, propSchema)
		fields = append(fields, field)
	}

	return fields
}
