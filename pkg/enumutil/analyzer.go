package enumutil

import "github.com/mirpo/schemagen/pkg/typegraph"

// EnumCategory describes the composition of enum values.
type EnumCategory struct {
	AllStrings bool // All values are strings
	AllNumbers bool // All values are numbers (int/float)
	HasMixed   bool // Has mixed types or non-primitive values
}

// AnalyzeEnumValues analyzes enum values to determine their category.
// This centralizes the duplicate enum type analysis logic from generators.
func AnalyzeEnumValues(values []typegraph.EnumValue) EnumCategory {
	if len(values) == 0 {
		return EnumCategory{AllStrings: true}
	}

	hasString := false
	hasNumber := false
	hasOther := false

	for _, v := range values {
		switch v.Value.(type) {
		case string:
			hasString = true
		case float64, int, int64, int32:
			hasNumber = true
		default:
			// bool, nil, objects, arrays
			hasOther = true
		}
	}

	return EnumCategory{
		AllStrings: hasString && !hasNumber && !hasOther,
		AllNumbers: hasNumber && !hasString && !hasOther,
		HasMixed:   (hasString && hasNumber) || (hasString && hasOther) || (hasNumber && hasOther) || hasOther,
	}
}

// AnalyzeRawEnumValues analyzes raw interface{} enum values (for inline enums in TypeRef).
func AnalyzeRawEnumValues(values []interface{}) EnumCategory {
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
