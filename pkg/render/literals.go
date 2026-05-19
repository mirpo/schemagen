package render

import "fmt"

type LiteralFormatter struct {
	NullValue  string
	TrueValue  string
	FalseValue string
}

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

func (f LiteralFormatter) FormatValue(val any) string {
	switch v := val.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	case float64:
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
