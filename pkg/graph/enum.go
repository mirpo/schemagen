package graph

type EnumCategory struct {
	AllStrings bool
	AllNumbers bool
	HasMixed   bool
}

func AnalyzeEnumValues(values []EnumValue) EnumCategory {
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
			hasOther = true
		}
	}

	return EnumCategory{
		AllStrings: hasString && !hasNumber && !hasOther,
		AllNumbers: hasNumber && !hasString && !hasOther,
		HasMixed:   (hasString && hasNumber) || (hasString && hasOther) || (hasNumber && hasOther) || hasOther,
	}
}
