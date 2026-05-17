package enumutil

import "github.com/mirpo/schemagen/pkg/typegraph"

type EnumCategory struct {
	AllStrings bool
	AllNumbers bool
	HasMixed   bool
}

// AnalyzeEnumValues analyzes enum values to determine their category.
// This centralizes the duplicate enum type analysis logic from generators.
func AnalyzeEnumValues(values []typegraph.EnumValue) EnumCategory {
	if len(values) == 0 {
		return EnumCategory{AllStrings: true}
	}

	raw := make([]interface{}, len(values))
	for i, v := range values {
		raw[i] = v.Value
	}
	return AnalyzeRawValues(raw)
}

// AnalyzeRawValues analyzes raw enum values ([]interface{}) to determine their category.
// Used for inline enums in TypeRef where values aren't wrapped in EnumValue.
func AnalyzeRawValues(values []interface{}) EnumCategory {
	if len(values) == 0 {
		return EnumCategory{AllStrings: true}
	}

	hasString := false
	hasNumber := false
	hasOther := false

	for _, v := range values {
		switch v.(type) {
		case string:
			hasString = true
		case float64, int, int64, int32:
			hasNumber = true
		default:
			hasOther = true
		}
	}

	return EnumCategory{
		AllStrings: hasString && !hasNumber && !hasOther,
		AllNumbers: hasNumber && !hasString && !hasOther,
		HasMixed:   (hasString && hasNumber) || (hasString && hasOther) || (hasNumber && hasOther) || hasOther,
	}
}
