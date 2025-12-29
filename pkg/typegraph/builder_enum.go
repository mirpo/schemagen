package typegraph

import (
	"fmt"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/naming"
)

// hasComplexEnumValues checks if enum contains non-primitive values (objects, arrays)
func hasComplexEnumValues(values []any) bool {
	for _, val := range values {
		switch val.(type) {
		case string, float64, int, int64, bool, nil:
			// Primitive types are OK
			continue
		case map[string]any:
			// Objects are complex
			return true
		case []any:
			// Arrays are complex
			return true
		default:
			// Unknown type, treat as complex
			return true
		}
	}
	return false
}

// BuildEnum builds an enum type from a schema (implements TypeBuilder).
func (b *Builder) BuildEnum(typ *Type, schema *jsonschema.Schema) error {
	typ.Kind = KindEnum
	typ.EnumValues = make([]EnumValue, 0)

	// Check if enum has complex values
	typ.HasComplexValues = hasComplexEnumValues(schema.Enum)

	// Determine enum type
	if len(schema.Enum) > 0 {
		if typ.HasComplexValues {
			typ.EnumType = "any" // Complex enums use any
		} else {
			// Determine from first value
			switch schema.Enum[0].(type) {
			case string:
				typ.EnumType = "string"
			case float64, int, int64:
				typ.EnumType = "int"
			default:
				typ.EnumType = "string"
			}
		}
	}

	// Extract enum values
	for _, val := range schema.Enum {
		enumVal := EnumValue{
			Name:  naming.ToConstantCase(fmt.Sprintf("%v", val)),
			Value: val,
		}
		typ.EnumValues = append(typ.EnumValues, enumVal)
	}

	return nil
}
