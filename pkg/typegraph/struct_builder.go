package typegraph

import (
	"fmt"

	"github.com/kaptinlin/jsonschema"
)

type fieldBuilder interface {
	BuildTypeRef(ctx *buildContext, schema *jsonschema.Schema, fieldName string) *TypeRef
	buildFieldsFromProperties(ctx *buildContext, schema *jsonschema.Schema, orderPath string) []*Field
}

type structBuilder struct {
	registry     *typeRegistry
	resolver     *refResolver
	fieldBuilder fieldBuilder
}

func newStructBuilder(registry *typeRegistry, resolver *refResolver) *structBuilder {
	return &structBuilder{
		registry: registry,
		resolver: resolver,
	}
}

func (b *structBuilder) setFieldBuilder(fb fieldBuilder) {
	b.fieldBuilder = fb
}

func (b *structBuilder) Build(ctx *buildContext, typ *Type, schema *jsonschema.Schema) error {
	typ.Kind = KindStruct

	// Pre-allocate based on expected sizes
	fieldCount := 0
	if schema.Properties != nil {
		fieldCount = len(*schema.Properties)
	}
	typ.Fields = make([]*Field, 0, fieldCount)
	typ.Extends = make([]string, 0, len(schema.AllOf))

	// Track required fields with pre-allocated map
	requiredMap := buildRequiredMap(schema.Required)

	// Handle allOf composition
	if len(schema.AllOf) > 0 {
		for allOfIndex, allOfSchema := range schema.AllOf {
			// Check if this is a $ref to another type
			if allOfSchema.Ref != "" {
				// Extract type name from $ref
				typeName := b.resolver.extractTypeName(allOfSchema.Ref)

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
				allOfPath := fmt.Sprintf("%s#/allOf/%d", ctx.Path, allOfIndex)
				for _, propName := range getOrderedPropertyNames(allOfSchema.Properties, allOfPath, ctx.Order) {
					propSchema := (*allOfSchema.Properties)[propName]
					field := &Field{
						JSONName:    propName,
						Description: getDescription(propSchema),
						Required:    requiredMap[propName],
						Type:        b.fieldBuilder.BuildTypeRef(ctx, propSchema, propName),
					}
					extractConstraints(field, propSchema)
					typ.Fields = append(typ.Fields, field)
				}
			}
		}
	}

	if schema.Properties != nil {
		fields := buildFieldsFromSchema(ctx, schema, ctx.Path, func(c *buildContext, s *jsonschema.Schema, name string) *TypeRef {
			return b.fieldBuilder.BuildTypeRef(c, s, name)
		})
		typ.Fields = append(typ.Fields, fields...)
	}

	// Deduplicate fields (in case allOf branches define the same field)
	typ.Fields = b.deduplicateFields(typ.Fields)

	// Capture additionalProperties configuration
	typ.AdditionalProps = extractAdditionalProperties(schema, func(s *jsonschema.Schema, fieldName string) *TypeRef {
		return b.fieldBuilder.BuildTypeRef(ctx, s, fieldName)
	})

	return nil
}

func (b *structBuilder) deduplicateFields(fields []*Field) []*Field {
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
