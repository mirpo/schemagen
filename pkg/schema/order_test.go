package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPropertyOrder(t *testing.T) {
	t.Run("new", func(t *testing.T) {
		po := newPropertyOrder()
		require.NotNil(t, po)
		assert.Empty(t, po.orders)
	})

	t.Run("get order", func(t *testing.T) {
		po := newPropertyOrder()
		po.orders["test.json"] = []string{"id", "name", "email"}
		result := po.GetOrder("test.json")
		require.Len(t, result, 3)
		assert.Equal(t, []string{"id", "name", "email"}, result)
	})

	t.Run("not found", func(t *testing.T) {
		po := newPropertyOrder()
		assert.Nil(t, po.GetOrder("nonexistent.json"))
	})
}

func TestDecodeOrderedObject(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		expectLen int
		expectErr bool
		firstKey  string
	}{
		{"simple object", []byte(`{"name": "John", "age": 30}`), 2, false, "name"},
		{"empty object", []byte(`{}`), 0, false, ""},
		{"invalid JSON", []byte(`{invalid}`), 0, true, ""},
		{"not an object", []byte(`["array"]`), 0, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields, err := decodeOrderedObject(tt.data)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, fields, tt.expectLen)
			if tt.firstKey != "" && len(fields) > 0 {
				assert.Equal(t, tt.firstKey, fields[0].Key)
			}
		})
	}
}

func TestExtractPropertyOrder(t *testing.T) {
	t.Run("simple object", func(t *testing.T) {
		schemaJSON := []byte(`{"type":"object","properties":{"id":{},"name":{},"email":{}}}`)
		po, err := extractPropertyOrder(schemaJSON, "user.json")
		require.NoError(t, err)
		order := po.GetOrder("user.json")
		assert.Equal(t, []string{"id", "name", "email"}, order)
	})

	t.Run("with $defs", func(t *testing.T) {
		schemaJSON := []byte(`{"type":"object","properties":{"user":{}},"$defs":{"Address":{"properties":{"street":{},"city":{}}}}}`)
		po, err := extractPropertyOrder(schemaJSON, "user.json")
		require.NoError(t, err)
		assert.Equal(t, []string{"user"}, po.GetOrder("user.json"))
		assert.Equal(t, []string{"street", "city"}, po.GetOrder("user.json#/$defs/Address"))
	})

	t.Run("with definitions", func(t *testing.T) {
		schemaJSON := []byte(`{"definitions":{"Person":{"properties":{"firstName":{},"lastName":{}}}}}`)
		po, err := extractPropertyOrder(schemaJSON, "schema.json")
		require.NoError(t, err)
		assert.Equal(t, []string{"firstName", "lastName"}, po.GetOrder("schema.json#/definitions/Person"))
	})

	t.Run("with allOf", func(t *testing.T) {
		schemaJSON := []byte(`{"allOf":[{"properties":{"id":{},"name":{}}},{"properties":{"createdAt":{}}}]}`)
		po, err := extractPropertyOrder(schemaJSON, "model.json")
		require.NoError(t, err)
		assert.Equal(t, []string{"id", "name"}, po.GetOrder("model.json#/allOf/0"))
		assert.Equal(t, []string{"createdAt"}, po.GetOrder("model.json#/allOf/1"))
	})

	t.Run("complex nested", func(t *testing.T) {
		schemaJSON := []byte(`{"properties":{"id":{},"data":{}},"$defs":{"Metadata":{"allOf":[{"properties":{"created":{},"modified":{}}}]}}}`)
		po, err := extractPropertyOrder(schemaJSON, "complex.json")
		require.NoError(t, err)
		assert.Equal(t, []string{"id", "data"}, po.GetOrder("complex.json"))
		assert.Equal(t, []string{"created", "modified"}, po.GetOrder("complex.json#/$defs/Metadata#/allOf/0"))
	})

	t.Run("no properties", func(t *testing.T) {
		schemaJSON := []byte(`{"type":"string","enum":["a","b"]}`)
		po, err := extractPropertyOrder(schemaJSON, "enum.json")
		require.NoError(t, err)
		assert.Nil(t, po.GetOrder("enum.json"))
	})

	t.Run("empty properties", func(t *testing.T) {
		schemaJSON := []byte(`{"type":"object","properties":{}}`)
		po, err := extractPropertyOrder(schemaJSON, "empty.json")
		require.NoError(t, err)
		assert.Empty(t, po.GetOrder("empty.json"))
	})

	t.Run("multiple defs", func(t *testing.T) {
		schemaJSON := []byte(`{"$defs":{"User":{"properties":{"name":{},"age":{}}},"Admin":{"properties":{"role":{}}}}}`)
		po, err := extractPropertyOrder(schemaJSON, "users.json")
		require.NoError(t, err)
		assert.Equal(t, []string{"name", "age"}, po.GetOrder("users.json#/$defs/User"))
		assert.Equal(t, []string{"role"}, po.GetOrder("users.json#/$defs/Admin"))
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := extractPropertyOrder([]byte(`{invalid`), "invalid.json")
		require.Error(t, err)
	})

}
