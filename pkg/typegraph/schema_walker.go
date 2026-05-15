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
func (w *schemaWalker) getOrderedDefNames(s *schema.Schema, defs map[string]*jsonschema.Schema) []string {
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

type typeBuilder interface {
	BuildStruct(ctx *buildContext, typ *Type, schema *jsonschema.Schema) error
	BuildEnum(typ *Type, schema *jsonschema.Schema) error
	BuildUnion(ctx *buildContext, typ *Type, schema *jsonschema.Schema) error
}

type schemaWalker struct {
	registry    *typeRegistry
	resolver    *refResolver
	typeBuilder typeBuilder
	config      *BuildConfig
}

func newSchemaWalker(registry *typeRegistry, resolver *refResolver, tb typeBuilder, config *BuildConfig) *schemaWalker {
	if config == nil {
		config = &BuildConfig{}
	}
	return &schemaWalker{
		registry:    registry,
		resolver:    resolver,
		typeBuilder: tb,
		config:      config,
	}
}

// Process processes a single schema and extracts types.
func (w *schemaWalker) Process(s *schema.Schema) error {
	ctx := &buildContext{
		Order: s.PropertyOrder,
		Path:  s.RelativePath,
	}

	compiled := s.Compiled
	w.resolver.setCurrentSchema(compiled)

	if compiled.Defs != nil {
		defNames := w.getOrderedDefNames(s, compiled.Defs)

		for _, defName := range defNames {
			defSchema := compiled.Defs[defName]
			defCtx := &buildContext{
				Order: ctx.Order,
				Path:  fmt.Sprintf("%s#/$defs/%s", s.RelativePath, defName),
			}

			if err := w.extractDefinition(defCtx, defName, defSchema); err != nil {
				return fmt.Errorf("extracting $def %s: %w", defName, err)
			}
		}
	}

	typ := &Type{
		Name:        s.Name,
		Description: getDescription(compiled),
	}

	if err := w.buildType(ctx, typ, compiled); err != nil {
		return err
	}

	w.registry.add(typ)
	return nil
}

func (w *schemaWalker) extractDefinition(ctx *buildContext, name string, schema *jsonschema.Schema) error {
	typ := &Type{
		Name:        naming.ToPascalCase(name),
		Description: getDescription(schema),
	}

	if err := w.buildType(ctx, typ, schema); err != nil {
		return err
	}

	w.registry.add(typ)
	return nil
}

func (w *schemaWalker) buildType(ctx *buildContext, typ *Type, schema *jsonschema.Schema) error {
	if isObject(schema) {
		return w.typeBuilder.BuildStruct(ctx, typ, schema)
	}
	if isEnum(schema) {
		return w.typeBuilder.BuildEnum(typ, schema)
	}
	if isUnion(schema) {
		return w.typeBuilder.BuildUnion(ctx, typ, schema)
	}
	typ.Kind = KindPrimitive
	typ.Primitive = mapPrimitiveSchema(schema)
	return nil
}
