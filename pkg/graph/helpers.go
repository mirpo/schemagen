package graph

import (
	"fmt"
	"sort"

	"github.com/mirpo/schemagen/pkg/parse"
)

func extractConstraints(node *parse.SchemaNode, f *Field) {
	f.MinLength = node.MinLength
	f.MaxLength = node.MaxLength
	f.Pattern = node.Pattern
	f.Minimum = node.Minimum
	f.Maximum = node.Maximum
	f.ExclusiveMinimum = node.ExclusiveMinimum
	f.ExclusiveMaximum = node.ExclusiveMaximum
	f.MinItems = node.MinItems
	f.MaxItems = node.MaxItems
	f.MultipleOf = node.MultipleOf
}

func mapPrimitive(typ, format string) PrimitiveKind {
	if format != "" {
		switch format {
		case FormatDateTime:
			return PrimDateTime
		case FormatDate:
			return PrimDate
		case FormatTime:
			return PrimTime
		case FormatUUID:
			return PrimUUID
		case FormatEmail:
			return PrimEmail
		case FormatURI, FormatURL:
			return PrimURI
		case FormatHostname:
			return PrimHostname
		case FormatIPv4:
			return PrimIPv4
		case FormatIPv6:
			return PrimIPv6
		case FormatInt32:
			return PrimInt32
		case FormatInt64:
			return PrimInt64
		case FormatFloat:
			return PrimFloat32
		case FormatDouble:
			return PrimFloat64
		}
	}

	switch typ {
	case parse.TypeString:
		return PrimString
	case parse.TypeInteger:
		return PrimInt
	case parse.TypeNumber:
		return PrimFloat64
	case parse.TypeBoolean:
		return PrimBool
	default:
		return PrimUnknown
	}
}

func sortedProperties(props []parse.NamedSchema) []parse.NamedSchema {
	sorted := make([]parse.NamedSchema, len(props))
	copy(sorted, props)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

func isNullType(node *parse.SchemaNode) bool {
	return len(node.Type) == 1 && node.Type[0] == parse.TypeNull
}

func rawToEnumValues(values []any) []EnumValue {
	if values == nil {
		return nil
	}
	result := make([]EnumValue, len(values))
	for i, v := range values {
		result[i] = EnumValue{Value: v}
	}
	return result
}

func enumValueName(v any) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

func uniqueMemberName(baseName string, index int) string {
	if index == 0 {
		return baseName
	}
	return fmt.Sprintf("%s%d", baseName, index)
}
