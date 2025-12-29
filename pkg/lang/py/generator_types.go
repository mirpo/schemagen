package py

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mirpo/schemagen/pkg/typegraph"
)

// checkTypeRefForImports recursively checks a TypeRef for import requirements.
func (g *Generator) checkTypeRefForImports(ref *typegraph.TypeRef) {
	if ref == nil {
		g.needsAny = true
		return
	}

	// Check format for special Pydantic types
	if ref.Format != "" {
		switch ref.Format {
		case "email":
			g.imports["pydantic_email"] = true
		case "uri", "url":
			g.imports["pydantic_url"] = true
		case "uuid":
			g.imports["uuid"] = true
		case "date-time":
			g.imports["datetime"] = true
		}
	}

	// Check Go type for imports
	switch ref.GoType {
	case "uuid.UUID":
		g.imports["uuid"] = true
	case "time.Time":
		g.imports["datetime"] = true
	}

	// Check for inline enum (needs Literal)
	if ref.Kind == typegraph.KindEnum && len(ref.EnumValues) > 0 {
		g.imports["typing_literal"] = true
	}

	// Check for types that need Any
	switch ref.Kind {
	case typegraph.KindInterface:
		g.needsAny = true
	case typegraph.KindPrimitive:
		// Check if it's interface{} which becomes Any
		if ref.GoType == "interface{}" {
			g.needsAny = true
		}
	case typegraph.KindMap:
		// Check if map value type needs Any
		g.checkTypeRefForImports(ref.ValueType)
	case typegraph.KindUnion:
		// Check all union members
		for _, member := range ref.UnionMembers {
			g.checkTypeRefForImports(member)
		}
	case typegraph.KindArray:
		g.checkTypeRefForImports(ref.ItemType)
	}
}

// typeRefToPython converts a TypeRef to a Python type annotation.
func (g *Generator) typeRefToPython(ref *typegraph.TypeRef, optional bool) string {
	if ref == nil {
		return "Any"
	}

	var pyType string

	switch ref.Kind {
	case typegraph.KindRef:
		// Reference to another type
		if ref.TypeName != "" {
			pyType = ref.TypeName
		} else {
			pyType = "Any"
		}
	case typegraph.KindEnum:
		// Inline enum - generate Literal type
		if len(ref.EnumValues) > 0 {
			g.imports["typing_literal"] = true
			literals := make([]string, 0, len(ref.EnumValues))
			for _, val := range ref.EnumValues {
				switch v := val.(type) {
				case string:
					literals = append(literals, fmt.Sprintf("%q", v))
				case float64, int, int64:
					literals = append(literals, fmt.Sprintf("%v", v))
				case bool:
					if v {
						literals = append(literals, "True")
					} else {
						literals = append(literals, "False")
					}
				case nil:
					literals = append(literals, "None")
				default:
					// Skip other types (objects, arrays)
				}
			}
			if len(literals) > 0 {
				pyType = fmt.Sprintf("Literal[%s]", strings.Join(literals, ", "))
			} else {
				pyType = "Any"
			}
		} else {
			pyType = "Any"
		}
	case typegraph.KindUnion:
		// Union type (oneOf/anyOf) - Python 3.10+ has native union support
		if len(ref.UnionMembers) > 0 {
			memberTypes := make([]string, len(ref.UnionMembers))
			for i, member := range ref.UnionMembers {
				memberTypes[i] = g.typeRefToPython(member, false)
			}
			pyType = strings.Join(memberTypes, " | ")
		} else {
			pyType = "Any"
		}
	case typegraph.KindPrimitive:
		pyType = g.primitiveToPython(ref.GoType, ref.Format)
	case typegraph.KindArray:
		itemType := g.typeRefToPython(ref.ItemType, false)
		pyType = fmt.Sprintf("list[%s]", itemType)
	case typegraph.KindMap:
		valueType := g.typeRefToPython(ref.ValueType, false)
		pyType = fmt.Sprintf("dict[str, %s]", valueType)
	case typegraph.KindInterface:
		// Check if this is an inline object with fields
		if len(ref.ObjectFields) > 0 {
			// For Python, we use dict[str, Any] but it's better than just dict
			// TODO: Extract as separate BaseModel class for full type safety
			pyType = "dict[str, Any]"
			g.needsAny = true
		} else {
			// Generic dict without structure
			pyType = "dict[str, Any]"
			g.needsAny = true
		}
	default:
		pyType = "Any"
	}

	// Add Optional wrapper if field is optional (unless already nullable)
	if optional && !strings.Contains(pyType, " | None") {
		pyType += " | None"
	}

	return pyType
}

// primitiveToPython maps Go primitive types to Python types.
func (g *Generator) primitiveToPython(goType string, format string) string {
	// Check for format-specific types first
	if format != "" {
		switch format {
		case "email":
			g.imports["pydantic_email"] = true
			return "EmailStr"
		case "uri", "url":
			g.imports["pydantic_url"] = true
			return "AnyUrl"
		case "uuid":
			g.imports["uuid"] = true
			return "UUID"
		case "date-time":
			g.imports["datetime"] = true
			return "datetime"
		}
	}

	// Fall back to centralized Go type mapping
	if pyType := typegraph.MapGoType(goType, "python"); pyType != "" {
		// Handle special imports for mapped types
		switch goType {
		case "uuid.UUID":
			g.imports["uuid"] = true
		case "time.Time":
			g.imports["datetime"] = true
		}
		return pyType
	}

	// Default to Any for unmapped types
	return "Any"
}

// collectTypeDependencies recursively collects all type names referenced by a TypeRef.
func collectTypeDependencies(ref *typegraph.TypeRef, deps map[string]bool) {
	if ref == nil {
		return
	}

	// Add direct type reference
	if ref.TypeName != "" {
		deps[ref.TypeName] = true
	}

	// Recursively collect from union members
	for _, member := range ref.UnionMembers {
		collectTypeDependencies(member, deps)
	}

	// Recursively collect from array item type
	collectTypeDependencies(ref.ItemType, deps)

	// Recursively collect from map value type
	collectTypeDependencies(ref.ValueType, deps)

	// Recursively collect from inline object fields
	for _, field := range ref.ObjectFields {
		collectTypeDependencies(field.Type, deps)
	}
}

// topologicalSort sorts types so that dependencies come before dependents.
// This ensures that base classes are defined before subclasses.
func topologicalSort(types []*typegraph.Type) []*typegraph.Type {
	// Build dependency graph: map from type name to types it depends on
	deps := make(map[string]map[string]bool)
	typeMap := make(map[string]*typegraph.Type)

	for _, typ := range types {
		typeMap[typ.Name] = typ
		deps[typ.Name] = make(map[string]bool)

		// Add dependencies from base classes (Extends)
		for _, base := range typ.Extends {
			deps[typ.Name][base] = true
		}

		// Add dependencies from field types (recursively collect all type refs)
		for _, field := range typ.Fields {
			collectTypeDependencies(field.Type, deps[typ.Name])
		}

		// Add dependencies from target type (for aliases and unions)
		collectTypeDependencies(typ.TargetType, deps[typ.Name])

		// Remove self-dependencies (forward references are handled by Python's `from __future__ import annotations`)
		delete(deps[typ.Name], typ.Name)
	}

	// Kahn's algorithm for topological sort
	inDegree := make(map[string]int)
	for name := range typeMap {
		inDegree[name] = 0
	}

	// Calculate in-degrees (only count dependencies that exist in our type set)
	for name, depSet := range deps {
		for dep := range depSet {
			if _, exists := typeMap[dep]; exists {
				inDegree[name]++
			}
		}
	}

	// Queue all types with no dependencies
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	// Sort the initial queue alphabetically for deterministic output
	sort.Strings(queue)

	// Process queue
	var sorted []*typegraph.Type
	for len(queue) > 0 {
		// Pop from queue
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, typeMap[current])

		// Reduce in-degree for all types that depend on current
		for name, depSet := range deps {
			if depSet[current] {
				inDegree[name]--
				if inDegree[name] == 0 {
					queue = append(queue, name)
					// Keep queue sorted for determinism
					sort.Strings(queue)
				}
			}
		}
	}

	// If we couldn't sort all types (cycle detected), fall back to original order
	if len(sorted) != len(types) {
		return types
	}

	return sorted
}
