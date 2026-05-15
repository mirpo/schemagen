package typegraph

import (
	"fmt"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/schema"
)

type Builder struct {
	registry       *typeRegistry
	resolver       *refResolver
	walker         *schemaWalker
	structBuilder  *structBuilder
	typeRefBuilder *typeRefBuilder
	compiler       *jsonschema.Compiler
	config         *BuildConfig
}

func newBuilder(compiler *jsonschema.Compiler) *Builder {
	return NewBuilderWithConfig(compiler, nil)
}

// NewBuilderWithConfig creates a new type graph builder with custom configuration.
func NewBuilderWithConfig(compiler *jsonschema.Compiler, cfg *BuildConfig) *Builder {
	if cfg == nil {
		cfg = &BuildConfig{}
	}
	registry := newTypeRegistry()
	resolver := newRefResolver(compiler)

	b := &Builder{
		registry: registry,
		resolver: resolver,
		compiler: compiler,
		config:   cfg,
	}

	b.typeRefBuilder = newTypeRefBuilder(registry, resolver, cfg)

	b.structBuilder = newStructBuilder(registry, resolver)
	b.structBuilder.setFieldBuilder(b.typeRefBuilder)

	b.walker = newSchemaWalker(registry, resolver, b, cfg)

	return b
}

// Build builds a type graph from loaded schemas.
func (b *Builder) Build(schemas []*schema.Schema) (*Graph, error) {
	for _, s := range schemas {
		if err := b.walker.Process(s); err != nil {
			return nil, fmt.Errorf("processing %s: %w", s.Path, err)
		}
	}

	graph := newGraph()
	for _, t := range b.registry.all() {
		graph.AddType(t)
	}
	return graph, nil
}

// Helper functions

func getDescription(schema *jsonschema.Schema) string {
	if schema.Description != nil {
		return *schema.Description
	}
	return ""
}

func isObject(schema *jsonschema.Schema) bool {
	// Check if schema defines an object with properties
	if schema.Properties != nil && len(*schema.Properties) > 0 {
		return true
	}
	// Check if schema uses allOf composition (even without direct properties)
	if len(schema.AllOf) > 0 {
		return true
	}
	// Check if schema defines property name constraints (even without direct properties)
	if schema.PropertyNames != nil {
		return true
	}
	return false
}

func isEnum(schema *jsonschema.Schema) bool {
	// Check if schema defines enum values
	return len(schema.Enum) > 0
}

func isUnion(schema *jsonschema.Schema) bool {
	return len(schema.AnyOf) > 0 || len(schema.OneOf) > 0
}

func (b *Builder) BuildTypeRef(ctx *buildContext, schema *jsonschema.Schema, fieldName string) *TypeRef {
	return b.typeRefBuilder.BuildTypeRef(ctx, schema, fieldName)
}
