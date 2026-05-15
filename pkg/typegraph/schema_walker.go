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
	BuildStruct(ctx *BuildContext, typ *Type, schema *jsonschema.Schema) error
	BuildEnum(typ *Type, schema *jsonschema.Schema) error
	BuildUnion(ctx *BuildContext, typ *Type, schema *jsonschema.Schema) error
}

// SchemaWalker handles schema traversal and type detection.
type SchemaWalker struct {
	registry    *TypeRegistry
	resolver    *RefResolver
	typeBuilder TypeBuilder
	config      *BuildConfig
}

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

// Process processes a single schema and extracts types.
func (w *SchemaWalker) Process(s *schema.Schema) error {
	ctx := &BuildContext{
		Order: s.PropertyOrder,
		Path:  s.RelativePath,
	}

	compiled := s.Compiled
	w.resolver.SetCurrentSchema(compiled)

	if compiled.Defs != nil {
		defNames := w.getOrderedDefNames(s, compiled.Defs)

		for _, defName := range defNames {
			defSchema := compiled.Defs[defName]
			defCtx := &BuildContext{
				Order: ctx.Order,
				Path:  fmt.Sprintf("%s#/$defs/%s", s.RelativePath, defName),
			}

			if err := w.extractDefinition(defCtx, defName, defSchema); err != nil {
				return fmt.Errorf("extracting $def %s: %w", defName, err)
			}
		}
	}

	typ := &Type{
		ID:          w.registry.NextID(),
		Name:        s.Name,
		Description: getDescription(compiled),
	}

	if isObject(compiled) {
		if err := w.typeBuilder.BuildStruct(ctx, typ, compiled); err != nil {
			return err
		}
	} else if isEnum(compiled) {
		if err := w.typeBuilder.BuildEnum(typ, compiled); err != nil {
			return err
		}
	} else if isUnion(compiled) {
		if err := w.typeBuilder.BuildUnion(ctx, typ, compiled); err != nil {
			return err
		}
	} else {
		typ.Kind = KindPrimitive
		typ.Primitive = MapPrimitiveSchema(compiled)
	}

	w.registry.Add(typ)
	return nil
}

func (w *SchemaWalker) extractDefinition(ctx *BuildContext, name string, schema *jsonschema.Schema) error {
	typ := &Type{
		ID:          w.registry.NextID(),
		Name:        naming.ToPascalCase(name),
		Description: getDescription(schema),
	}

	if isObject(schema) {
		if err := w.typeBuilder.BuildStruct(ctx, typ, schema); err != nil {
			return err
		}
	} else if isEnum(schema) {
		if err := w.typeBuilder.BuildEnum(typ, schema); err != nil {
			return err
		}
	} else if isUnion(schema) {
		if err := w.typeBuilder.BuildUnion(ctx, typ, schema); err != nil {
			return err
		}
	} else {
		typ.Kind = KindPrimitive
		typ.Primitive = MapPrimitiveSchema(schema)
	}

	w.registry.Add(typ)
	return nil
}
