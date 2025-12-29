package typegraph

// PrimitiveMapping maps a Go type to language-specific types.
type PrimitiveMapping struct {
	TypeScript string
	Python     string
	Go         string
}

// PrimitiveMappings defines the mapping from Go types to language-specific types.
var PrimitiveMappings = map[string]PrimitiveMapping{
	"string":      {TypeScript: "string", Python: "str", Go: "string"},
	"int":         {TypeScript: "number", Python: "int", Go: "int"},
	"int32":       {TypeScript: "number", Python: "int", Go: "int32"},
	"int64":       {TypeScript: "number", Python: "int", Go: "int64"},
	"float32":     {TypeScript: "number", Python: "float", Go: "float32"},
	"float64":     {TypeScript: "number", Python: "float", Go: "float64"},
	"bool":        {TypeScript: "boolean", Python: "bool", Go: "bool"},
	"time.Time":   {TypeScript: "string", Python: "datetime", Go: "time.Time"},
	"uuid.UUID":   {TypeScript: "string", Python: "UUID", Go: "uuid.UUID"},
	"interface{}": {TypeScript: "unknown", Python: "Any", Go: "interface{}"},
}

// ForLanguage returns the type string for a given language.
// lang should be "typescript", "python", or "go".
func (m PrimitiveMapping) ForLanguage(lang string) string {
	switch lang {
	case "typescript", "ts":
		return m.TypeScript
	case "python", "py":
		return m.Python
	case "go", "golang":
		return m.Go
	default:
		return ""
	}
}

// MapGoType maps a Go type to a target language type.
// Returns empty string if the type is not found.
func MapGoType(goType, targetLang string) string {
	if mapping, exists := PrimitiveMappings[goType]; exists {
		return mapping.ForLanguage(targetLang)
	}
	return ""
}
