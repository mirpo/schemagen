package typegraph

import (
	"testing"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/schema"
	"github.com/stretchr/testify/assert"
)

func TestGetOrderedPropertyNames(t *testing.T) {
	t.Run("with order preserves schema order", func(t *testing.T) {
		order, _ := schema.ExtractPropertyOrder([]byte(`{"properties": {"z": {}, "a": {}, "m": {}}}`), "test.json")

		names := GetOrderedPropertyNames(&jsonschema.SchemaMap{
			"z": &jsonschema.Schema{}, "a": &jsonschema.Schema{}, "m": &jsonschema.Schema{},
		}, "test.json", order)

		assert.Equal(t, []string{"z", "a", "m"}, names)
	})

	t.Run("alphabetical fallback when no order", func(t *testing.T) {
		properties := jsonschema.SchemaMap{
			"zebra": &jsonschema.Schema{},
			"apple": &jsonschema.Schema{},
			"mango": &jsonschema.Schema{},
		}
		result := GetOrderedPropertyNames(&properties, "test.json", nil)
		assert.Equal(t, []string{"apple", "mango", "zebra"}, result)
	})

	t.Run("nil properties returns nil", func(t *testing.T) {
		result := GetOrderedPropertyNames(nil, "test.json", nil)
		assert.Nil(t, result)
	})
}

func TestExtractConstraints(t *testing.T) {
	t.Run("string constraints", func(t *testing.T) {
		minLen, maxLen, pattern := float64(5), float64(100), "^[a-z]+$"
		field := &Field{}
		ExtractConstraints(field, &jsonschema.Schema{MinLength: &minLen, MaxLength: &maxLen, Pattern: &pattern})

		assert.Equal(t, 5, *field.MinLength)
		assert.Equal(t, 100, *field.MaxLength)
		assert.Equal(t, "^[a-z]+$", *field.Pattern)
	})

	t.Run("number constraints", func(t *testing.T) {
		field := &Field{}
		ExtractConstraints(field, &jsonschema.Schema{
			Minimum: jsonschema.NewRat(0),
			Maximum: jsonschema.NewRat(100),
		})

		assert.InDelta(t, 0.0, *field.Minimum, 0.0001)
		assert.InDelta(t, 100.0, *field.Maximum, 0.0001)
	})
}

func TestExtractAdditionalProperties(t *testing.T) {
	t.Run("nil schema returns nil", func(t *testing.T) {
		config := ExtractAdditionalProperties(&jsonschema.Schema{}, nil)
		assert.Nil(t, config)
	})

	t.Run("boolean true", func(t *testing.T) {
		boolTrue := true
		config := ExtractAdditionalProperties(&jsonschema.Schema{
			AdditionalProperties: &jsonschema.Schema{Boolean: &boolTrue},
		}, func(s *jsonschema.Schema, name string) *TypeRef { return nil })

		assert.True(t, config.Allowed)
		assert.Nil(t, config.Type)
	})

	t.Run("typed additional properties", func(t *testing.T) {
		buildTypeRef := func(s *jsonschema.Schema, name string) *TypeRef {
			return &TypeRef{Kind: KindPrimitive, Primitive: PrimString}
		}

		config := ExtractAdditionalProperties(&jsonschema.Schema{
			AdditionalProperties: &jsonschema.Schema{Type: []string{"string"}},
		}, buildTypeRef)

		assert.True(t, config.Allowed)
		assert.Equal(t, KindPrimitive, config.Type.Kind)
	})
}
