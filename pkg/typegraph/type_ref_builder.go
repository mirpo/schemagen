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
	ref := &TypeRef{Nullable: b.isNullable(schema)}

	// Dispatch to specific handlers based on schema structure
	switch {
	case schema.Ref != "":
		return b.buildRefTypeRef(ref, schema)
	case schema.Const != nil && schema.Const.IsSet:
		return b.buildConstTypeRef(ref, schema)
	case len(schema.Enum) > 0:
		return b.buildEnumTypeRef(ref, schema, fieldName)
	case len(schema.OneOf) > 0:
		ref.Kind = KindUnion
		ref.UnionMembers = b.buildUnionMembers(schema.OneOf, fieldName)
		return ref
	case len(schema.AnyOf) > 0:
		ref.Kind = KindUnion
		ref.UnionMembers = b.buildUnionMembers(schema.AnyOf, fieldName)
		return ref
	case b.hasType(schema, "array"):
		return b.buildArrayTypeRef(ref, schema, fieldName)
	case b.hasType(schema, "object"):
		return b.buildObjectTypeRef(ref, schema, fieldName)
	default:
		return b.buildPrimitiveTypeRef(ref, schema)
	}
}

// isNullable checks if the schema allows null values.
func (b *TypeRefBuilder) isNullable(schema *jsonschema.Schema) bool {
	for _, t := range schema.Type {
		if t == "null" {
			return true
		}
	}
	return false
}

// hasType checks if the schema has the specified primary type.
func (b *TypeRefBuilder) hasType(schema *jsonschema.Schema, typeName string) bool {
	return len(schema.Type) > 0 && schema.Type[0] == typeName
}

// buildRefTypeRef handles $ref schemas.
func (b *TypeRefBuilder) buildRefTypeRef(ref *TypeRef, schema *jsonschema.Schema) *TypeRef {
	typeName := b.resolver.ExtractTypeName(schema.Ref)
	if typeName != "" {
		ref.Kind = KindRef
		ref.TypeName = typeName
		return ref
	}
	// Unresolved ref - fall back to interface{}
	ref.Kind = KindInterface
	ref.GoType = "interface{}"
	return ref
}

// buildConstTypeRef handles const (single-value enum) schemas.
func (b *TypeRefBuilder) buildConstTypeRef(ref *TypeRef, schema *jsonschema.Schema) *TypeRef {
	constValue := schema.Const.Value
	ref.Kind = KindEnum
	ref.EnumValues = []interface{}{constValue}
	ref.GoType = b.inferGoTypeFromValue(constValue)
	return ref
}

// buildEnumTypeRef handles inline enum schemas.
func (b *TypeRefBuilder) buildEnumTypeRef(ref *TypeRef, schema *jsonschema.Schema, fieldName string) *TypeRef {
	// Check if we should extract as separate named type
	if b.config.ExtractInlined && fieldName != "" {
		return b.extractEnumType(ref, schema, fieldName)
	}
	// Keep inline
	ref.Kind = KindEnum
	ref.EnumValues = schema.Enum
	if len(schema.Enum) > 0 {
		ref.GoType = b.inferGoTypeFromValue(schema.Enum[0])
	}
	return ref
}

// extractEnumType extracts an inline enum as a separate named type.
func (b *TypeRefBuilder) extractEnumType(ref *TypeRef, schema *jsonschema.Schema, fieldName string) *TypeRef {
	enumTypeName := b.registry.EnsureUniqueName(naming.ToPascalCase(fieldName))

	enumType := &Type{
		ID:          b.registry.NextID(),
		Name:        enumTypeName,
		Kind:        KindEnum,
		Description: getDescription(schema),
		EnumValues:  make([]EnumValue, 0, len(schema.Enum)),
		EnumType:    b.inferEnumType(schema.Enum[0]),
	}

	for _, val := range schema.Enum {
		enumType.EnumValues = append(enumType.EnumValues, EnumValue{
			Name:  naming.ToConstantCase(fmt.Sprintf("%v", val)),
			Value: val,
		})
	}

	b.registry.Add(enumType)
	ref.Kind = KindRef
	ref.TypeName = enumTypeName
	return ref
}

// buildArrayTypeRef handles array schemas.
func (b *TypeRefBuilder) buildArrayTypeRef(ref *TypeRef, schema *jsonschema.Schema, fieldName string) *TypeRef {
	ref.Kind = KindArray
	if schema.Items != nil {
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

// buildObjectTypeRef handles object schemas.
func (b *TypeRefBuilder) buildObjectTypeRef(ref *TypeRef, schema *jsonschema.Schema, fieldName string) *TypeRef {
	hasProps := schema.Properties != nil && len(*schema.Properties) > 0

	if !hasProps {
		// Object without properties - treat as map
		ref.Kind = KindMap
		if schema.AdditionalProperties != nil && schema.AdditionalProperties.Boolean == nil {
			ref.ValueType = b.BuildTypeRef(schema.AdditionalProperties, "additionalProperty")
		} else {
			ref.ValueType = &TypeRef{Kind: KindPrimitive, GoType: "interface{}"}
		}
		return ref
	}

	// Object with properties
	if b.config.ExtractInlined && fieldName != "" {
		objectTypeName := b.registry.EnsureUniqueName(naming.ToPascalCase(fieldName))
		extractedType := b.ExtractInlineObjectType(objectTypeName, schema)
		b.registry.Add(extractedType)
		ref.Kind = KindRef
		ref.TypeName = extractedType.Name
		return ref
	}

	// Keep inline - create inline object with fields
	ref.Kind = KindInterface
	ref.GoType = "map[string]interface{}"
	ref.ObjectFields = b.BuildFieldsFromProperties(schema, "")
	return ref
}

// buildPrimitiveTypeRef handles primitive type schemas.
func (b *TypeRefBuilder) buildPrimitiveTypeRef(ref *TypeRef, schema *jsonschema.Schema) *TypeRef {
	ref.Kind = KindPrimitive
	ref.GoType = b.MapPrimitiveType(schema)
	if schema.Format != nil && *schema.Format != "" {
		ref.Format = *schema.Format
	}
	return ref
}

// inferGoTypeFromValue infers Go type from a value.
func (b *TypeRefBuilder) inferGoTypeFromValue(val interface{}) string {
	switch val.(type) {
	case string:
		return "string"
	case float64, int, int64:
		return "int"
	default:
		return "interface{}"
	}
}

// inferEnumType infers enum type (string/int) from first value.
func (b *TypeRefBuilder) inferEnumType(val interface{}) string {
	switch val.(type) {
	case string:
		return "string"
	case float64, int, int64:
		return "int"
	default:
		return "string"
	}
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

// buildUnionMembers builds TypeRefs for union members (oneOf/anyOf).
// Extracts inline objects as separate types when applicable.
func (b *TypeRefBuilder) buildUnionMembers(members []*jsonschema.Schema, fieldName string) []*TypeRef {
	result := make([]*TypeRef, 0, len(members))
	for i, memberSchema := range members {
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
			b.registry.Add(extractedType)
			result = append(result, &TypeRef{
				Kind:     KindRef,
				TypeName: extractedType.Name,
			})
		} else {
			result = append(result, b.BuildTypeRef(memberSchema, ""))
		}
	}
	return result
}
