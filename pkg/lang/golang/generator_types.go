package golang

import (
	"github.com/mirpo/schemagen/pkg/typegraph"
)

// fieldGoType determines the Go type for a field.
func (g *Generator) fieldGoType(field *typegraph.Field) string {
	typeStr := g.typeRefToGoType(field.Type)

	// Use pointer for optional fields if configured
	if !field.Required && g.config.UsePointers {
		return "*" + typeStr
	}

	return typeStr
}

// typeRefToGoType converts a TypeRef to a Go type string.
func (g *Generator) typeRefToGoType(ref *typegraph.TypeRef) string {
	if ref.GoType != "" {
		return g.primitiveToGo(ref.GoType)
	}

	if ref.TypeName != "" {
		// Reference to named type
		return ref.TypeName
	}

	switch ref.Kind {
	case typegraph.KindUnion:
		// If this is a named union type, use the type name
		// Otherwise fall back to any
		if ref.TypeName != "" {
			return ref.TypeName
		}
		return "any"
	case typegraph.KindArray:
		itemType := g.typeRefToGoType(ref.ItemType)
		return "[]" + itemType
	case typegraph.KindMap:
		valueType := g.typeRefToGoType(ref.ValueType)
		return "map[string]" + valueType
	case typegraph.KindInterface:
		// Go always extracts inline types, so this should never be reached
		// If it is, fall back to interface{} as a safety measure
		return "interface{}"
	default:
		return "interface{}"
	}
}

// primitiveToGo validates and returns Go primitive types using centralized mapping.
// This ensures consistency across all generators.
func (g *Generator) primitiveToGo(goType string) string {
	// Try centralized mapping for validation
	if mappedType := typegraph.MapGoType(goType, "go"); mappedType != "" {
		return mappedType
	}

	// If not in mapping, return as-is (could be a complex type)
	return goType
}

// scanTypeForImports scans a type to determine needed imports.
func (g *Generator) scanTypeForImports(typ *typegraph.Type) {
	switch typ.Kind {
	case typegraph.KindStruct:
		for _, field := range typ.Fields {
			g.scanTypeRefForImports(field.Type)
			// Check if validation tags are needed
			if field.Required || field.MinLength != nil || field.MaxLength != nil ||
				field.Pattern != nil || field.Minimum != nil || field.Maximum != nil ||
				field.MinItems != nil || field.MaxItems != nil ||
				(field.Type != nil && field.Type.Format != "") {
				g.imports["github.com/go-playground/validator/v10"] = true
			}
		}
	case typegraph.KindPrimitive:
		g.checkGoTypeForImport(typ.GoType)
	}
}

// scanTypeRefForImports scans a TypeRef recursively using Walk.
func (g *Generator) scanTypeRefForImports(ref *typegraph.TypeRef) {
	ref.Walk(func(r *typegraph.TypeRef) {
		g.checkGoTypeForImport(r.GoType)
	})
}

// checkGoTypeForImport adds imports based on Go type.
func (g *Generator) checkGoTypeForImport(goType string) {
	switch goType {
	case "uuid.UUID":
		g.imports["github.com/google/uuid"] = true
	case "time.Time":
		g.imports["time"] = true
	}
}

// resetImports clears the imports map.
func (g *Generator) resetImports() {
	g.imports = make(map[string]bool)
}
