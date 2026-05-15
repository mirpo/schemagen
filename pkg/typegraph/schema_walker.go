package typegraph

import (
	"fmt"
	"maps"
	"slices"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/naming"
	"github.com/mirpo/schemagen/pkg/schema"
)

// getOrderedDefNames returns $defs names in original schema order.
func (w *SchemaWalker) getOrderedDefNames(s *schema.Schema, defs map[string]*jsonschema.Schema) []string {
	// PropertyOrder may be nil in tests that create schemas manually
	if s.PropertyOrder == nil {
		return slices.Sorted(maps.Keys(defs))
	}

	defsPath := s.RelativePath + "#/$defs"
	ordered := s.PropertyOrder.GetDefsOrder(defsPath)

	// Filter to only include keys that exist in defs (defensive)
	mapKeys := make(map[string]bool)
	for key := range defs {
		mapKeys[key] = true
	}

	result := make([]string, 0, len(ordered))
	for _, key := range ordered {
		if mapKeys[key] {
			result = append(result, key)
			delete(mapKeys, key)
		}
	}

	// Add any keys not in order (shouldn't happen, but be safe)
	if len(mapKeys) > 0 {
		extra := slices.Sorted(maps.Keys(mapKeys))
		result = append(result, extra...)
	}

	return result
}

// TypeBuilder interface for building specific type kinds.
// Used to break circular dependency between SchemaWalker and type builders.
type TypeBuilder interface {
	BuildStruct(typ *Type, schema *jsonschema.Schema) error
	BuildEnum(typ *Type, schema *jsonschema.Schema) error
	BuildUnion(typ *Type, schema *jsonschema.Schema) error
	MapPrimitiveType(schema *jsonschema.Schema) string
}

// SchemaWalker handles schema traversal and type detection.
type SchemaWalker struct {
	registry     *TypeRegistry
	resolver     *RefResolver
	typeBuilder  TypeBuilder
	config       *BuildConfig
	currentOrder *schema.PropertyOrder
	currentPath  string
}

// NewSchemaWalker creates a new schema walker.
func NewSchemaWalker(registry *TypeRegistry, resolver *RefResolver, tb TypeBuilder, config *BuildConfig) *SchemaWalker {
	if config == nil {
		config = &BuildConfig{}
	}
	return &SchemaWalker{
		registry:    registry,
		resolver:    resolver,
		typeBuilder: tb,
		config:      config,
	}
}

// SetCurrentOrder sets the current property order for lookups.
func (w *SchemaWalker) SetCurrentOrder(order *schema.PropertyOrder) {
	w.currentOrder = order
}

// SetCurrentPath sets the current schema path for order lookups.
func (w *SchemaWalker) SetCurrentPath(path string) {
	w.currentPath = path
}

// CurrentOrder returns the current property order.
func (w *SchemaWalker) CurrentOrder() *schema.PropertyOrder {
	return w.currentOrder
}

// CurrentPath returns the current schema path.
func (w *SchemaWalker) CurrentPath() string {
	return w.currentPath
}

// Process processes a single schema and extracts types.
func (w *SchemaWalker) Process(s *schema.Schema) error {
	// Store current schema's property order for lookups
	w.currentOrder = s.PropertyOrder
	w.currentPath = s.RelativePath

	// Get the compiled schema
	compiled := s.Compiled
	// Store current root schema for self-reference resolution
	w.resolver.SetCurrentSchema(compiled)

	// First, extract $defs as separate types
	if compiled.Defs != nil {
		// Get $defs in original schema order (preserves JSON file order)
		defNames := w.getOrderedDefNames(s, compiled.Defs)

		// Process each $def with correct path for order lookup
		for _, defName := range defNames {
			defSchema := compiled.Defs[defName]

			// Set currentPath to the $def path for property order lookup
			previousPath := w.currentPath
			w.currentPath = fmt.Sprintf("%s#/$defs/%s", s.RelativePath, defName)

			if err := w.ExtractDefinition(defName, defSchema); err != nil {
				return fmt.Errorf("extracting $def %s: %w", defName, err)
			}

			// Restore path
			w.currentPath = previousPath
		}
	}

	// Determine type kind based on schema
	typ := &Type{
		ID:          w.registry.NextID(),
		Name:        s.Name,
		Description: getDescription(compiled),
	}

	// Handle different schema types
	if isObject(compiled) {
		if err := w.typeBuilder.BuildStruct(typ, compiled); err != nil {
			return err
		}
	} else if isEnum(compiled) {
		if err := w.typeBuilder.BuildEnum(typ, compiled); err != nil {
			return err
		}
	} else if isUnion(compiled) {
		if err := w.typeBuilder.BuildUnion(typ, compiled); err != nil {
			return err
		}
	} else {
		// Primitive or other type
		typ.Kind = KindPrimitive
		typ.GoType = w.typeBuilder.MapPrimitiveType(compiled)
	}

	w.registry.Add(typ)
	return nil
}

// ExtractDefinition extracts a $def as a separate type.
func (w *SchemaWalker) ExtractDefinition(name string, schema *jsonschema.Schema) error {
	typ := &Type{
		ID:          w.registry.NextID(),
		Name:        naming.ToPascalCase(name),
		Description: getDescription(schema),
	}

	if isObject(schema) {
		if err := w.typeBuilder.BuildStruct(typ, schema); err != nil {
			return err
		}
	} else if isEnum(schema) {
		if err := w.typeBuilder.BuildEnum(typ, schema); err != nil {
			return err
		}
	} else if isUnion(schema) {
		if err := w.typeBuilder.BuildUnion(typ, schema); err != nil {
			return err
		}
	} else {
		typ.Kind = KindPrimitive
		typ.GoType = w.typeBuilder.MapPrimitiveType(schema)
	}

	w.registry.Add(typ)
	return nil
}
