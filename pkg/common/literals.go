package common

import "fmt"

// LiteralFormatter provides language-specific literal value formatting.
type LiteralFormatter struct {
	NullValue  string // "nil", "null", "None"
	TrueValue  string // "true", "True"
	FalseValue string // "false", "False"
}

// Language-specific literal formatters.
var (
	GoLiterals = LiteralFormatter{
		NullValue:  "nil",
		TrueValue:  "true",
		FalseValue: "false",
	}
	TSLiterals = LiteralFormatter{
		NullValue:  "null",
		TrueValue:  "true",
		FalseValue: "false",
	}
	PyLiterals = LiteralFormatter{
		NullValue:  "None",
		TrueValue:  "True",
		FalseValue: "False",
	}
)

// FormatValue formats a literal value according to the language's conventions.
func (f LiteralFormatter) FormatValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	case float64:
		// Check if it's a whole number
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case int32:
		return fmt.Sprintf("%d", v)
	case bool:
		if v {
			return f.TrueValue
		}
		return f.FalseValue
	case nil:
		return f.NullValue
	default:
		return fmt.Sprintf("%v", v)
	}
}
