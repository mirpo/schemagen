package typegraph

import (
	"fmt"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/naming"
	"github.com/mirpo/schemagen/pkg/schema"
)

// TypeRefBuilder builds type references from JSON schemas.
// Implements FieldBuilder interface.
type TypeRefBuilder struct {
	registry      *TypeRegistry
	resolver      *RefResolver
	structBuilder *StructBuilder
	config        *BuildConfig
	currentOrder  *schema.PropertyOrder
}

// NewTypeRefBuilder creates a new TypeRefBuilder.
func NewTypeRefBuilder(registry *TypeRegistry, resolver *RefResolver, config *BuildConfig) *TypeRefBuilder {
	if config == nil {
		config = &BuildConfig{}
	}
	return &TypeRefBuilder{
		registry: registry,
		resolver: resolver,
		config:   config,
	}
}

// SetStructBuilder sets the StructBuilder for extracting inline objects.
func (b *TypeRefBuilder) SetStructBuilder(sb *StructBuilder) {
	b.structBuilder = sb
}

// SetCurrentOrder sets the current property order.
func (b *TypeRefBuilder) SetCurrentOrder(order *schema.PropertyOrder) {
	b.currentOrder = order
}

// Build builds a TypeRef from a schema property.
// fieldName is used for naming extracted inline types (can be empty).
func (b *TypeRefBuilder) BuildTypeRef(schema *jsonschema.Schema, fieldName string) *TypeRef {
	ref := &TypeRef{
		Nullable: false,
	}

	// Check for nullable types (type array containing "null")
	if len(schema.Type) > 0 {
		for _, t := range schema.Type {
			if t == "null" {
				ref.Nullable = true
			}
		}
	}

	// Check for $ref (external or internal reference)
	if schema.Ref != "" {
		refURI := schema.Ref

		// Extract type name from $ref (handles both #/$defs/Type and external refs)
		typeName := b.resolver.ExtractTypeName(refURI)
		if typeName != "" {
			ref.Kind = KindRef
			ref.TypeName = typeName
			return ref
		}

		// If we couldn't extract a type name, fall back to interface{}
		ref.Kind = KindInterface
		ref.GoType = "interface{}" // Unresolved ref
		return ref
	}

	// Check for const (single-value enum)
	if schema.Const != nil && schema.Const.IsSet {
		constValue := schema.Const.Value
		ref.Kind = KindEnum
		ref.EnumValues = []interface{}{constValue}
		// Determine base type from const value
		switch constValue.(type) {
		case string:
			ref.GoType = "string"
		case float64, int, int64:
			ref.GoType = "int"
		default:
			ref.GoType = "interface{}"
		}
		return ref
	}

	// Check for inline enum (before oneOf/anyOf since enum takes precedence)
	if len(schema.Enum) > 0 {
		// Check if we should extract inline enums
		if b.config.ExtractInlined && fieldName != "" {
			// Extract as separate named type
			enumTypeName := naming.ToPascalCase(fieldName)
			// Check if enum type already exists, otherwise create unique name
			enumTypeName = b.registry.EnsureUniqueName(enumTypeName)

			enumType := &Type{
				ID:          b.registry.NextID(),
				Name:        enumTypeName,
				Kind:        KindEnum,
				Description: getDescription(schema),
				EnumValues:  make([]EnumValue, 0),
			}

			// Determine enum type from first value
			switch schema.Enum[0].(type) {
			case string:
				enumType.EnumType = "string"
			case float64, int, int64:
				enumType.EnumType = "int"
			default:
				enumType.EnumType = "string"
			}

			// Extract enum values
			for _, val := range schema.Enum {
				enumVal := EnumValue{
					Name:  naming.ToConstantCase(fmt.Sprintf("%v", val)),
					Value: val,
				}
				enumType.EnumValues = append(enumType.EnumValues, enumVal)
			}

			b.registry.Add(enumType)

			// Return reference to extracted type
			ref.Kind = KindRef
			ref.TypeName = enumTypeName
			return ref
		}

		// Otherwise, keep inline
		ref.Kind = KindEnum
		ref.EnumValues = schema.Enum
		// Determine base type from first value
		if len(schema.Enum) > 0 {
			switch schema.Enum[0].(type) {
			case string:
				ref.GoType = "string"
			case float64, int, int64:
				ref.GoType = "int"
			default:
				ref.GoType = "interface{}"
			}
		}
		return ref
	}

	// Check for oneOf/anyOf (union types)
	if len(schema.OneOf) > 0 {
		ref.Kind = KindUnion
		ref.UnionMembers = make([]*TypeRef, 0, len(schema.OneOf))
		for i, memberSchema := range schema.OneOf {
			// Check if this is an inline object that should be extracted
			if b.ShouldExtractInlineObject(memberSchema) {
				// Extract as separate type with field-based name
				var typeName string
				if fieldName != "" {
					baseName := naming.ToPascalCase(fieldName)
					if i == 0 {
						typeName = baseName // First variant: Payload
					} else {
						typeName = fmt.Sprintf("%s%d", baseName, i) // Subsequent: Payload1, Payload2
					}
				} else {
					typeName = fmt.Sprintf("Variant%d", i) // Fallback
				}
				extractedType := b.ExtractInlineObjectType(typeName, memberSchema)
				// Add to types list
				b.registry.Add(extractedType)
				// Return reference to extracted type
				memberRef := &TypeRef{
					Kind:     KindRef,
					TypeName: extractedType.Name,
				}
				ref.UnionMembers = append(ref.UnionMembers, memberRef)
			} else {
				memberRef := b.BuildTypeRef(memberSchema, "")
				ref.UnionMembers = append(ref.UnionMembers, memberRef)
			}
		}
		return ref
	}

	if len(schema.AnyOf) > 0 {
		ref.Kind = KindUnion
		ref.UnionMembers = make([]*TypeRef, 0, len(schema.AnyOf))
		for i, memberSchema := range schema.AnyOf {
			// Check if this is an inline object that should be extracted
			if b.ShouldExtractInlineObject(memberSchema) {
				// Extract as separate type with field-based name
				var typeName string
				if fieldName != "" {
					baseName := naming.ToPascalCase(fieldName)
					if i == 0 {
						typeName = baseName // First variant: Payload
					} else {
						typeName = fmt.Sprintf("%s%d", baseName, i) // Subsequent: Payload1, Payload2
					}
				} else {
					typeName = fmt.Sprintf("Variant%d", i) // Fallback
				}
				extractedType := b.ExtractInlineObjectType(typeName, memberSchema)
				// Add to types list
				b.registry.Add(extractedType)
				// Return reference to extracted type
				memberRef := &TypeRef{
					Kind:     KindRef,
					TypeName: extractedType.Name,
				}
				ref.UnionMembers = append(ref.UnionMembers, memberRef)
			} else {
				memberRef := b.BuildTypeRef(memberSchema, "")
				ref.UnionMembers = append(ref.UnionMembers, memberRef)
			}
		}
		return ref
	}

	// Check for arrays
	if len(schema.Type) > 0 && schema.Type[0] == "array" {
		ref.Kind = KindArray
		if schema.Items != nil {
			// Pass field name context for array items to enable extraction
			itemFieldName := ""
			if fieldName != "" {
				itemFieldName = fieldName + "Item"
			}
			ref.ItemType = b.BuildTypeRef(schema.Items, itemFieldName)
		} else {
			ref.ItemType = &TypeRef{Kind: KindPrimitive, GoType: "interface{}"}
		}
		return ref
	}

	// Check for objects
	if len(schema.Type) > 0 && schema.Type[0] == "object" {
		hasProps := schema.Properties != nil && len(*schema.Properties) > 0
		if !hasProps {
			// Object without properties - treat as map
			ref.Kind = KindMap

			// Check if additionalProperties specifies a type
			if schema.AdditionalProperties != nil && schema.AdditionalProperties.Boolean == nil {
				// Has typed additionalProperties (e.g., {type: "string"})
				ref.ValueType = b.BuildTypeRef(schema.AdditionalProperties, "additionalProperty")
			} else {
				// No additionalProperties or just true/false - use any type
				ref.ValueType = &TypeRef{Kind: KindPrimitive, GoType: "interface{}"}
			}
			return ref
		}

		// Object with properties
		if b.config.ExtractInlined && fieldName != "" {
			// Extract as separate named type
			objectTypeName := naming.ToPascalCase(fieldName)
			objectTypeName = b.registry.EnsureUniqueName(objectTypeName)

			extractedType := b.ExtractInlineObjectType(objectTypeName, schema)
			b.registry.Add(extractedType)

			// Return reference to extracted type
			ref.Kind = KindRef
			ref.TypeName = extractedType.Name
			return ref
		}

		// Otherwise, keep inline - create inline object with fields
		ref.Kind = KindInterface
		ref.GoType = "map[string]interface{}"

		// Extract fields for the inline object
		ref.ObjectFields = b.BuildFieldsFromProperties(schema, "")
		return ref
	}

	// Primitive type
	ref.Kind = KindPrimitive
	ref.GoType = b.MapPrimitiveType(schema)

	// Extract format if present (email, uri, uuid, date-time, etc.)
	if schema.Format != nil && *schema.Format != "" {
		ref.Format = *schema.Format
	}

	return ref
}

// ShouldExtractInlineObject checks if a schema represents an inline object that should be extracted.
func (b *TypeRefBuilder) ShouldExtractInlineObject(schema *jsonschema.Schema) bool {
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

// ExtractInlineObjectType extracts an inline object as a separate Type.
func (b *TypeRefBuilder) ExtractInlineObjectType(baseName string, schema *jsonschema.Schema) *Type {
	typ := &Type{
		ID:          b.registry.NextID(),
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
				Type:        b.BuildTypeRef(propSchema, propName),
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

// BuildFieldsFromProperties extracts fields from schema properties for inline objects.
// This is used to populate ObjectFields in TypeRef for anonymous interfaces.
// Implements FieldBuilder interface.
func (b *TypeRefBuilder) BuildFieldsFromProperties(schema *jsonschema.Schema, orderPath string) []*Field {
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
			Type:        b.BuildTypeRef(propSchema, propName),
		}
		// Extract validation constraints
		b.extractConstraints(field, propSchema)
		fields = append(fields, field)
	}

	return fields
}

// MapPrimitiveType maps a JSON schema type to a Go type.
// Implements FieldBuilder interface.
func (b *TypeRefBuilder) MapPrimitiveType(schema *jsonschema.Schema) string {
	// Check for format-specific mappings first
	if schema.Format != nil {
		switch *schema.Format {
		case "uuid":
			return "uuid.UUID"
		case "date-time":
			return "time.Time"
		case "date":
			return "time.Time"
		case "time":
			return "string" // No standard Go type for time-only
		case "email", "uri", "hostname", "ipv4", "ipv6":
			return "string" // Validated strings
		}
	}

	// Check type
	if len(schema.Type) > 0 {
		switch schema.Type[0] {
		case "string":
			return "string"
		case "integer":
			// Check for specific integer formats
			if schema.Format != nil {
				switch *schema.Format {
				case "int32":
					return "int32"
				case "int64":
					return "int64"
				}
			}
			return "int"
		case "number":
			// Check for specific number formats
			if schema.Format != nil {
				switch *schema.Format {
				case "float":
					return "float32"
				case "double":
					return "float64"
				}
			}
			return "float64"
		case "boolean":
			return "bool"
		case "array":
			return "[]interface{}"
		case "object":
			return "map[string]interface{}"
		}
	}

	return "interface{}"
}

// getOrderedPropertyNames delegates to structBuilder.GetOrderedPropertyNames.
func (b *TypeRefBuilder) getOrderedPropertyNames(properties *jsonschema.SchemaMap, schemaPath string) []string {
	b.structBuilder.SetCurrentOrder(b.currentOrder)
	return b.structBuilder.GetOrderedPropertyNames(properties, schemaPath)
}

// extractConstraints delegates to structBuilder.ExtractConstraints.
func (b *TypeRefBuilder) extractConstraints(field *Field, schema *jsonschema.Schema) {
	b.structBuilder.ExtractConstraints(field, schema)
}

// extractAdditionalProperties delegates to structBuilder.ExtractAdditionalProperties.
func (b *TypeRefBuilder) extractAdditionalProperties(schema *jsonschema.Schema) *AdditionalPropsConfig {
	return b.structBuilder.ExtractAdditionalProperties(schema)
}
