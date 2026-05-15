package typegraph

import (
	"fmt"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/naming"
)

// TypeRefBuilder builds type references from JSON schemas.
// Implements FieldBuilder interface.
type TypeRefBuilder struct {
	registry *TypeRegistry
	resolver *RefResolver
	config   *BuildConfig
}

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

func (b *TypeRefBuilder) BuildTypeRef(ctx *BuildContext, schema *jsonschema.Schema, fieldName string) *TypeRef {
	ref := &TypeRef{Nullable: b.isNullable(schema)}

	// Dispatch to specific handlers based on schema structure
	switch {
	case schema.Ref != "":
		return b.buildRefTypeRef(ref, schema)
	case schema.Const != nil && schema.Const.IsSet:
		return b.buildConstTypeRef(ref, schema)
	case len(schema.Enum) > 0:
		return b.buildEnumTypeRef(ctx, ref, schema, fieldName)
	case len(schema.OneOf) > 0:
		ref.Kind = KindUnion
		ref.UnionMembers = b.buildUnionMembers(ctx, schema.OneOf, fieldName)
		return ref
	case len(schema.AnyOf) > 0:
		ref.Kind = KindUnion
		ref.UnionMembers = b.buildUnionMembers(ctx, schema.AnyOf, fieldName)
		return ref
	case b.hasType(schema, "array"):
		return b.buildArrayTypeRef(ctx, ref, schema, fieldName)
	case b.hasType(schema, "object"):
		return b.buildObjectTypeRef(ctx, ref, schema, fieldName)
	default:
		return b.buildPrimitiveTypeRef(ref, schema)
	}
}

func (b *TypeRefBuilder) isNullable(schema *jsonschema.Schema) bool {
	return b.hasType(schema, "null")
}

func (b *TypeRefBuilder) hasType(schema *jsonschema.Schema, typeName string) bool {
	for _, t := range schema.Type {
		if t == typeName {
			return true
		}
	}
	return false
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
	ref.Primitive = PrimUnknown
	return ref
}

// buildConstTypeRef handles const (single-value enum) schemas.
func (b *TypeRefBuilder) buildConstTypeRef(ref *TypeRef, schema *jsonschema.Schema) *TypeRef {
	constValue := schema.Const.Value
	ref.Kind = KindEnum
	ref.EnumValues = []interface{}{constValue}
	ref.Primitive = b.inferPrimitiveFromValue(constValue)

	return ref
}

// buildEnumTypeRef handles inline enum schemas.
func (b *TypeRefBuilder) buildEnumTypeRef(ctx *BuildContext, ref *TypeRef, schema *jsonschema.Schema, fieldName string) *TypeRef {
	// Check if we should extract as separate named type
	if b.config.ExtractInlined && fieldName != "" {
		return b.extractEnumType(ref, schema, fieldName)
	}
	// Keep inline
	ref.Kind = KindEnum
	ref.EnumValues = schema.Enum
	if len(schema.Enum) > 0 {
		ref.Primitive = b.inferPrimitiveFromValue(schema.Enum[0])
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
func (b *TypeRefBuilder) buildArrayTypeRef(ctx *BuildContext, ref *TypeRef, schema *jsonschema.Schema, fieldName string) *TypeRef {
	ref.Kind = KindArray
	if schema.Items != nil {
		itemFieldName := ""
		if fieldName != "" {
			itemFieldName = fieldName + "Item"
		}
		ref.ItemType = b.BuildTypeRef(ctx, schema.Items, itemFieldName)
	} else {
		ref.ItemType = &TypeRef{Kind: KindPrimitive, Primitive: PrimUnknown}
	}
	return ref
}

// buildObjectTypeRef handles object schemas.
func (b *TypeRefBuilder) buildObjectTypeRef(ctx *BuildContext, ref *TypeRef, schema *jsonschema.Schema, fieldName string) *TypeRef {
	hasProps := schema.Properties != nil && len(*schema.Properties) > 0

	if !hasProps {
		// Object without properties - treat as map
		ref.Kind = KindMap
		if schema.AdditionalProperties != nil && schema.AdditionalProperties.Boolean == nil {
			ref.ValueType = b.BuildTypeRef(ctx, schema.AdditionalProperties, "additionalProperty")
		} else {
			ref.ValueType = &TypeRef{Kind: KindPrimitive, Primitive: PrimUnknown}
		}
		return ref
	}

	// Object with properties
	if b.config.ExtractInlined && fieldName != "" {
		objectTypeName := b.registry.EnsureUniqueName(naming.ToPascalCase(fieldName))
		extractedType := b.ExtractInlineObjectType(ctx, objectTypeName, schema)
		b.registry.Add(extractedType)
		ref.Kind = KindRef
		ref.TypeName = extractedType.Name
		return ref
	}

	// Keep inline - create inline object with fields
	ref.Kind = KindInterface
	ref.Primitive = PrimUnknown
	ref.ObjectFields = b.BuildFieldsFromProperties(ctx, schema, "")
	return ref
}

// buildPrimitiveTypeRef handles primitive type schemas.
func (b *TypeRefBuilder) buildPrimitiveTypeRef(ref *TypeRef, schema *jsonschema.Schema) *TypeRef {
	ref.Kind = KindPrimitive
	ref.Primitive = MapPrimitiveSchema(schema)

	if schema.Format != nil && *schema.Format != "" {
		ref.Format = *schema.Format
	}
	return ref
}

func (b *TypeRefBuilder) inferPrimitiveFromValue(val interface{}) PrimitiveKind {
	switch val.(type) {
	case string:
		return PrimString
	case float64, int, int64:
		return PrimInt
	default:
		return PrimUnknown
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
func (b *TypeRefBuilder) ExtractInlineObjectType(ctx *BuildContext, baseName string, schema *jsonschema.Schema) *Type {
	// Pre-allocate based on expected sizes
	fieldCount := 0
	if schema.Properties != nil {
		fieldCount = len(*schema.Properties)
	}

	typ := &Type{
		ID:          b.registry.NextID(),
		Name:        baseName,
		Kind:        KindStruct,
		Description: getDescription(schema),
		Fields:      make([]*Field, 0, fieldCount),
	}

	// Build required map with pre-allocated capacity
	requiredMap := make(map[string]bool, len(schema.Required))
	for _, req := range schema.Required {
		requiredMap[req] = true
	}

	// Extract properties
	if schema.Properties != nil {
		// Inline objects don't have pre-extracted order info, so use empty path (falls back to alphabetical)
		for _, propName := range GetOrderedPropertyNames(schema.Properties, "", ctx.Order) {
			propSchema := (*schema.Properties)[propName]
			field := &Field{
				Name:        naming.ToPascalCase(propName),
				JSONName:    propName,
				Description: getDescription(propSchema),
				Required:    requiredMap[propName],
				OmitEmpty:   !requiredMap[propName],
				Type:        b.BuildTypeRef(ctx, propSchema, propName),
			}
			// Extract validation constraints
			ExtractConstraints(field, propSchema)
			typ.Fields = append(typ.Fields, field)
		}
	}

	// Capture additionalProperties configuration
	typ.AdditionalProps = ExtractAdditionalProperties(schema, func(s *jsonschema.Schema, fieldName string) *TypeRef { return b.BuildTypeRef(ctx, s, fieldName) })

	return typ
}

// BuildFieldsFromProperties extracts fields from schema properties for inline objects.
// This is used to populate ObjectFields in TypeRef for anonymous interfaces.
// Implements FieldBuilder interface.
func (b *TypeRefBuilder) BuildFieldsFromProperties(ctx *BuildContext, schema *jsonschema.Schema, orderPath string) []*Field {
	if schema.Properties == nil {
		return nil
	}

	// Build required map with pre-allocated capacity
	requiredMap := make(map[string]bool, len(schema.Required))
	for _, req := range schema.Required {
		requiredMap[req] = true
	}

	fields := make([]*Field, 0, len(*schema.Properties))

	// Iterate over properties in order
	for _, propName := range GetOrderedPropertyNames(schema.Properties, orderPath, ctx.Order) {
		propSchema := (*schema.Properties)[propName]
		field := &Field{
			Name:        naming.ToPascalCase(propName),
			JSONName:    propName,
			Description: getDescription(propSchema),
			Required:    requiredMap[propName],
			OmitEmpty:   !requiredMap[propName],
			Type:        b.BuildTypeRef(ctx, propSchema, propName),
		}
		// Extract validation constraints
		ExtractConstraints(field, propSchema)
		fields = append(fields, field)
	}

	return fields
}

// MapPrimitiveSchema maps a JSON schema to a PrimitiveKind.
func MapPrimitiveSchema(schema *jsonschema.Schema) PrimitiveKind {
	if schema.Format != nil {
		switch *schema.Format {
		case "uuid":
			return PrimUUID
		case "date-time":
			return PrimDateTime
		case "date":
			return PrimDate
		case "time":
			return PrimTime
		case "email":
			return PrimEmail
		case "uri":
			return PrimURI
		case "hostname":
			return PrimHostname
		case "ipv4":
			return PrimIPv4
		case "ipv6":
			return PrimIPv6
		case "int32":
			return PrimInt32
		case "int64":
			return PrimInt64
		case "float":
			return PrimFloat32
		case "double":
			return PrimFloat64
		}
	}

	if len(schema.Type) > 0 {
		switch schema.Type[0] {
		case "string":
			return PrimString
		case "integer":
			return PrimInt
		case "number":
			return PrimFloat64
		case "boolean":
			return PrimBool
		}
	}

	return PrimUnknown
}

// buildUnionMembers builds TypeRefs for union members (oneOf/anyOf).
// Extracts inline objects as separate types when applicable.
func (b *TypeRefBuilder) buildUnionMembers(ctx *BuildContext, members []*jsonschema.Schema, fieldName string) []*TypeRef {
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
			extractedType := b.ExtractInlineObjectType(ctx, typeName, memberSchema)
			b.registry.Add(extractedType)
			result = append(result, &TypeRef{
				Kind:     KindRef,
				TypeName: extractedType.Name,
			})
		} else {
			result = append(result, b.BuildTypeRef(ctx, memberSchema, ""))
		}
	}
	return result
}
