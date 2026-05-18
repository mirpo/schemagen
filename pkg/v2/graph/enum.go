package graph

import "github.com/mirpo/schemagen/pkg/enumutil"

func AnalyzeEnumValues(values []EnumValue) enumutil.EnumCategory {
	raw := make([]any, len(values))
	for i, v := range values {
		raw[i] = v.Value
	}
	return enumutil.AnalyzeRawValues(raw)
}
