package typegraph

import (
	"fmt"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/schema"
)

// Builder builds a type graph from JSON schemas.
type Builder struct {
	registry       *TypeRegistry
	resolver       *RefResolver
	walker         *SchemaWalker
	structBuilder  *StructBuilder
	typeRefBuilder *TypeRefBuilder
	compiler       *jsonschema.Compiler // For resolving $refs
	config         *BuildConfig         // Build configuration
}

// NewBuilder creates a new type graph builder with default configuration.
func NewBuilder(compiler *jsonschema.Compiler) *Builder {
	return NewBuilderWithConfig(compiler, nil)
}

// NewBuilderWithConfig creates a new type graph builder with custom configuration.
func NewBuilderWithConfig(compiler *jsonschema.Compiler, cfg *BuildConfig) *Builder {
	if cfg == nil {
		cfg = &BuildConfig{}
	}
	registry := NewTypeRegistry()
	resolver := NewRefResolver(compiler)

	b := &Builder{
		registry: registry,
		resolver: resolver,
		compiler: compiler,
		config:   cfg,
	}

	// Create TypeRefBuilder
	b.typeRefBuilder = NewTypeRefBuilder(registry, resolver, cfg)

	// Create StructBuilder and wire TypeRefBuilder as FieldBuilder
	b.structBuilder = NewStructBuilder(registry, resolver)
	b.structBuilder.SetFieldBuilder(b.typeRefBuilder)

	// Create walker with builder as TypeBuilder (breaks circular dependency via interface)
	b.walker = NewSchemaWalker(registry, resolver, b, cfg)

	return b
}

// Build builds a type graph from loaded schemas.
func (b *Builder) Build(schemas []*schema.Schema) (*Graph, error) {
	for _, s := range schemas {
		if err := b.walker.Process(s); err != nil {
			return nil, fmt.Errorf("processing %s: %w", s.Path, err)
		}
	}

	graph := NewGraph()
	for _, t := range b.registry.All() {
		graph.AddType(t)
	}
	return graph, nil
}

// MapPrimitiveType maps a JSON schema type to a Go type (implements TypeBuilder).
// Delegates to typeRefBuilder.
func (b *Builder) MapPrimitiveType(schema *jsonschema.Schema) string {
	return b.typeRefBuilder.MapPrimitiveType(schema)
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

// BuildTypeRef builds a TypeRef from a schema property (implements FieldBuilder).
// Delegates to typeRefBuilder.
func (b *Builder) BuildTypeRef(schema *jsonschema.Schema, fieldName string) *TypeRef {
	// Sync walker's currentOrder to typeRefBuilder
	b.typeRefBuilder.SetCurrentOrder(b.walker.CurrentOrder())
	return b.typeRefBuilder.BuildTypeRef(schema, fieldName)
}

// getOrderedPropertyNames delegates to structBuilder.GetOrderedPropertyNames.
func (b *Builder) getOrderedPropertyNames(properties *jsonschema.SchemaMap, schemaPath string) []string {
	return b.structBuilder.GetOrderedPropertyNames(properties, schemaPath)
}
