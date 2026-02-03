package typegraph

import (
	"fmt"

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

	// Pre-allocate based on expected sizes
	fieldCount := 0
	if schema.Properties != nil {
		fieldCount = len(*schema.Properties)
	}
	typ.Fields = make([]*Field, 0, fieldCount)
	typ.Extends = make([]string, 0, len(schema.AllOf))

	// Track required fields with pre-allocated map
	requiredMap := make(map[string]bool, len(schema.Required))
	for _, req := range schema.Required {
		requiredMap[req] = true
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
				for _, propName := range GetOrderedPropertyNames(allOfSchema.Properties, allOfPath, b.currentOrder) {
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
					ExtractConstraints(field, propSchema)
					typ.Fields = append(typ.Fields, field)
				}
			}
		}
	}

	// Extract properties from the main schema
	if schema.Properties != nil {
		// Iterate over properties in schema file order
		for _, propName := range GetOrderedPropertyNames(schema.Properties, b.currentPath, b.currentOrder) {
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
			ExtractConstraints(field, propSchema)
			typ.Fields = append(typ.Fields, field)
		}
	}

	// Deduplicate fields (in case allOf branches define the same field)
	typ.Fields = b.DeduplicateFields(typ.Fields)

	// Capture additionalProperties configuration
	typ.AdditionalProps = ExtractAdditionalProperties(schema, b.fieldBuilder.BuildTypeRef)

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
