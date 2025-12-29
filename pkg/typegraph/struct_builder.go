package typegraph

import (
	"fmt"
	"sort"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/naming"
	"github.com/mirpo/schemagen/pkg/schema"
)

// FieldBuilder interface for building fields/type refs.
// Breaks circular dependency between StructBuilder and TypeRefBuilder.
type FieldBuilder interface {
	BuildTypeRef(schema *jsonschema.Schema, fieldName string) *TypeRef
	BuildFieldsFromProperties(schema *jsonschema.Schema, orderPath string) []*Field
	MapPrimitiveType(schema *jsonschema.Schema) string
}

// StructBuilder builds struct types from JSON schemas.
type StructBuilder struct {
	registry     *TypeRegistry
	resolver     *RefResolver
	fieldBuilder FieldBuilder
	currentOrder *schema.PropertyOrder
	currentPath  string
}

// NewStructBuilder creates a new StructBuilder.
func NewStructBuilder(registry *TypeRegistry, resolver *RefResolver) *StructBuilder {
	return &StructBuilder{
		registry: registry,
		resolver: resolver,
	}
}

// SetFieldBuilder sets the FieldBuilder (setter injection to break cycle).
func (b *StructBuilder) SetFieldBuilder(fb FieldBuilder) {
	b.fieldBuilder = fb
}

// SetCurrentOrder sets the current property order for field ordering.
func (b *StructBuilder) SetCurrentOrder(order *schema.PropertyOrder) {
	b.currentOrder = order
}

// SetCurrentPath sets the current schema path for order lookups.
func (b *StructBuilder) SetCurrentPath(path string) {
	b.currentPath = path
}

// Build builds a struct type from a schema.
func (b *StructBuilder) Build(typ *Type, schema *jsonschema.Schema) error {
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
				typeName := b.resolver.ExtractTypeName(allOfSchema.Ref)

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
				for _, propName := range b.GetOrderedPropertyNames(allOfSchema.Properties, allOfPath) {
					propSchema := (*allOfSchema.Properties)[propName]
					field := &Field{
						Name:        naming.ToPascalCase(propName),
						JSONName:    propName,
						Description: getDescription(propSchema),
						Required:    requiredMap[propName],
						OmitEmpty:   !requiredMap[propName],
						Type:        b.fieldBuilder.BuildTypeRef(propSchema, propName),
					}
					// Extract validation constraints
					b.ExtractConstraints(field, propSchema)
					typ.Fields = append(typ.Fields, field)
				}
			}
		}
	}

	// Extract properties from the main schema
	if schema.Properties != nil {
		// Iterate over properties in schema file order
		for _, propName := range b.GetOrderedPropertyNames(schema.Properties, b.currentPath) {
			propSchema := (*schema.Properties)[propName]
			field := &Field{
				Name:        naming.ToPascalCase(propName),
				JSONName:    propName,
				Description: getDescription(propSchema),
				Required:    requiredMap[propName],
				OmitEmpty:   !requiredMap[propName],
				Type:        b.fieldBuilder.BuildTypeRef(propSchema, propName),
			}
			// Extract validation constraints
			b.ExtractConstraints(field, propSchema)
			typ.Fields = append(typ.Fields, field)
		}
	}

	// Deduplicate fields (in case allOf branches define the same field)
	typ.Fields = b.DeduplicateFields(typ.Fields)

	// Capture additionalProperties configuration
	typ.AdditionalProps = b.ExtractAdditionalProperties(schema)

	return nil
}

// DeduplicateFields removes duplicate fields, keeping the most specific one.
func (b *StructBuilder) DeduplicateFields(fields []*Field) []*Field {
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

// ExtractConstraints extracts validation constraints from a schema into a field.
func (b *StructBuilder) ExtractConstraints(field *Field, schema *jsonschema.Schema) {
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

// ExtractAdditionalProperties captures additionalProperties configuration from a schema.
func (b *StructBuilder) ExtractAdditionalProperties(schema *jsonschema.Schema) *AdditionalPropsConfig {
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
		Type:    b.fieldBuilder.BuildTypeRef(addlProps, "additionalProperty"),
	}

	return config
}

// GetOrderedPropertyNames returns property names in schema file order.
// Falls back to alphabetical sorting if order information is not available.
func (b *StructBuilder) GetOrderedPropertyNames(properties *jsonschema.SchemaMap, schemaPath string) []string {
	if properties == nil {
		return nil
	}

	// Try to get order from extracted property order
	if b.currentOrder != nil {
		if ordered := b.currentOrder.GetOrder(schemaPath); len(ordered) > 0 {
			// Filter to only include keys that exist in the properties map
			// (defensive programming - should always match)
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
				sort.Strings(extra) // Sort extras for determinism
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
