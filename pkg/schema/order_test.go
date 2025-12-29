package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPropertyOrder tests creating a new PropertyOrder
func TestNewPropertyOrder(t *testing.T) {
	po := NewPropertyOrder()

	require.NotNil(t, po, "NewPropertyOrder should not return nil")
	assert.NotNil(t, po.orders, "orders map should be initialized")
	assert.Empty(t, po.orders, "orders map should be empty")
}

// TestPropertyOrder_GetOrder tests getting property order
func TestPropertyOrder_GetOrder(t *testing.T) {
	po := NewPropertyOrder()
	po.orders["test.json"] = []string{"id", "name", "email"}

	result := po.GetOrder("test.json")

	require.Len(t, result, 3, "should return 3 properties")
	assert.Equal(t, "id", result[0], "first property should be id")
	assert.Equal(t, "name", result[1], "second property should be name")
	assert.Equal(t, "email", result[2], "third property should be email")
}

// TestPropertyOrder_GetOrder_NotFound tests getting order for non-existent path
func TestPropertyOrder_GetOrder_NotFound(t *testing.T) {
	po := NewPropertyOrder()

	result := po.GetOrder("nonexistent.json")

	assert.Nil(t, result, "should return nil for non-existent path")
}

// TestDecodeOrderedObject tests decoding JSON object with field order
func TestDecodeOrderedObject(t *testing.T) {
	data := []byte(`{"name": "John", "age": 30, "email": "john@example.com"}`)

	fields, err := DecodeOrderedObject(data)

	require.NoError(t, err, "DecodeOrderedObject should not return error")
	require.Len(t, fields, 3, "should decode 3 fields")

	assert.Equal(t, "name", fields[0].Key, "first field should be name")
	assert.Equal(t, "age", fields[1].Key, "second field should be age")
	assert.Equal(t, "email", fields[2].Key, "third field should be email")

	// Verify values
	var nameValue string
	err = json.Unmarshal(fields[0].Value, &nameValue)
	require.NoError(t, err, "should unmarshal name value")
	assert.Equal(t, "John", nameValue, "name value should match")
}

// TestDecodeOrderedObject_EmptyObject tests decoding empty object
func TestDecodeOrderedObject_EmptyObject(t *testing.T) {
	data := []byte(`{}`)

	fields, err := DecodeOrderedObject(data)

	require.NoError(t, err, "DecodeOrderedObject should not return error")
	assert.Empty(t, fields, "should decode 0 fields")
}

// TestDecodeOrderedObject_NestedObject tests decoding object with nested values
func TestDecodeOrderedObject_NestedObject(t *testing.T) {
	data := []byte(`{"user": {"name": "John"}, "active": true}`)

	fields, err := DecodeOrderedObject(data)

	require.NoError(t, err, "DecodeOrderedObject should not return error")
	require.Len(t, fields, 2, "should decode 2 fields")

	assert.Equal(t, "user", fields[0].Key, "first field should be user")
	assert.Equal(t, "active", fields[1].Key, "second field should be active")
}

// TestDecodeOrderedObject_InvalidJSON tests decoding invalid JSON
func TestDecodeOrderedObject_InvalidJSON(t *testing.T) {
	data := []byte(`{invalid}`)

	_, err := DecodeOrderedObject(data)

	assert.Error(t, err, "DecodeOrderedObject should return error for invalid JSON")
}

// TestDecodeOrderedObject_NotAnObject tests decoding non-object JSON
func TestDecodeOrderedObject_NotAnObject(t *testing.T) {
	data := []byte(`["array", "not", "object"]`)

	_, err := DecodeOrderedObject(data)

	assert.Error(t, err, "DecodeOrderedObject should return error for non-object")
}

// TestExtractPropertyOrder_SimpleObject tests extracting property order from simple object
func TestExtractPropertyOrder_SimpleObject(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"name": {"type": "string"},
			"email": {"type": "string"}
		}
	}`)

	po, err := ExtractPropertyOrder(schemaJSON, "user.json")

	require.NoError(t, err, "ExtractPropertyOrder should not return error")
	require.NotNil(t, po, "PropertyOrder should not be nil")

	order := po.GetOrder("user.json")
	require.Len(t, order, 3, "should extract 3 properties")
	assert.Equal(t, "id", order[0], "first property should be id")
	assert.Equal(t, "name", order[1], "second property should be name")
	assert.Equal(t, "email", order[2], "third property should be email")
}

// TestExtractPropertyOrder_WithDefs tests extracting property order with $defs
func TestExtractPropertyOrder_WithDefs(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"user": {"type": "string"}
		},
		"$defs": {
			"Address": {
				"type": "object",
				"properties": {
					"street": {"type": "string"},
					"city": {"type": "string"},
					"zip": {"type": "string"}
				}
			}
		}
	}`)

	po, err := ExtractPropertyOrder(schemaJSON, "user.json")

	require.NoError(t, err, "ExtractPropertyOrder should not return error")

	// Check main schema properties
	mainOrder := po.GetOrder("user.json")
	require.Len(t, mainOrder, 1, "main schema should have 1 property")
	assert.Equal(t, "user", mainOrder[0], "property should be user")

	// Check $defs/Address properties
	addressOrder := po.GetOrder("user.json#/$defs/Address")
	require.Len(t, addressOrder, 3, "Address should have 3 properties")
	assert.Equal(t, "street", addressOrder[0], "first property should be street")
	assert.Equal(t, "city", addressOrder[1], "second property should be city")
	assert.Equal(t, "zip", addressOrder[2], "third property should be zip")
}

// TestExtractPropertyOrder_WithDefinitions tests extracting with definitions keyword
func TestExtractPropertyOrder_WithDefinitions(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"definitions": {
			"Person": {
				"type": "object",
				"properties": {
					"firstName": {"type": "string"},
					"lastName": {"type": "string"}
				}
			}
		}
	}`)

	po, err := ExtractPropertyOrder(schemaJSON, "schema.json")

	require.NoError(t, err, "ExtractPropertyOrder should not return error")

	// Check definitions/Person properties
	personOrder := po.GetOrder("schema.json#/definitions/Person")
	require.Len(t, personOrder, 2, "Person should have 2 properties")
	assert.Equal(t, "firstName", personOrder[0], "first property should be firstName")
	assert.Equal(t, "lastName", personOrder[1], "second property should be lastName")
}

// TestExtractPropertyOrder_WithAllOf tests extracting property order with allOf
func TestExtractPropertyOrder_WithAllOf(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"allOf": [
			{
				"properties": {
					"id": {"type": "string"},
					"name": {"type": "string"}
				}
			},
			{
				"properties": {
					"createdAt": {"type": "string"},
					"updatedAt": {"type": "string"}
				}
			}
		]
	}`)

	po, err := ExtractPropertyOrder(schemaJSON, "model.json")

	require.NoError(t, err, "ExtractPropertyOrder should not return error")

	// Check first allOf element
	order0 := po.GetOrder("model.json#/allOf/0")
	require.Len(t, order0, 2, "allOf[0] should have 2 properties")
	assert.Equal(t, "id", order0[0], "first property should be id")
	assert.Equal(t, "name", order0[1], "second property should be name")

	// Check second allOf element
	order1 := po.GetOrder("model.json#/allOf/1")
	require.Len(t, order1, 2, "allOf[1] should have 2 properties")
	assert.Equal(t, "createdAt", order1[0], "first property should be createdAt")
	assert.Equal(t, "updatedAt", order1[1], "second property should be updatedAt")
}

// TestExtractPropertyOrder_WithOneOf tests extracting property order with oneOf
func TestExtractPropertyOrder_WithOneOf(t *testing.T) {
	schemaJSON := []byte(`{
		"oneOf": [
			{
				"properties": {
					"type": {"type": "string"},
					"value": {"type": "number"}
				}
			},
			{
				"properties": {
					"type": {"type": "string"},
					"text": {"type": "string"}
				}
			}
		]
	}`)

	po, err := ExtractPropertyOrder(schemaJSON, "union.json")

	require.NoError(t, err, "ExtractPropertyOrder should not return error")

	// Check first oneOf element
	order0 := po.GetOrder("union.json#/oneOf/0")
	require.Len(t, order0, 2, "oneOf[0] should have 2 properties")
	assert.Equal(t, "type", order0[0], "first property should be type")
	assert.Equal(t, "value", order0[1], "second property should be value")

	// Check second oneOf element
	order1 := po.GetOrder("union.json#/oneOf/1")
	require.Len(t, order1, 2, "oneOf[1] should have 2 properties")
	assert.Equal(t, "type", order1[0], "first property should be type")
	assert.Equal(t, "text", order1[1], "second property should be text")
}

// TestExtractPropertyOrder_WithAnyOf tests extracting property order with anyOf
func TestExtractPropertyOrder_WithAnyOf(t *testing.T) {
	schemaJSON := []byte(`{
		"anyOf": [
			{
				"properties": {
					"a": {"type": "string"},
					"b": {"type": "string"}
				}
			},
			{
				"properties": {
					"c": {"type": "string"},
					"d": {"type": "string"}
				}
			}
		]
	}`)

	po, err := ExtractPropertyOrder(schemaJSON, "any.json")

	require.NoError(t, err, "ExtractPropertyOrder should not return error")

	// Check anyOf elements
	order0 := po.GetOrder("any.json#/anyOf/0")
	require.Len(t, order0, 2, "anyOf[0] should have 2 properties")

	order1 := po.GetOrder("any.json#/anyOf/1")
	require.Len(t, order1, 2, "anyOf[1] should have 2 properties")
}

// TestExtractPropertyOrder_ComplexNested tests extracting from complex nested schema
func TestExtractPropertyOrder_ComplexNested(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"data": {"type": "object"}
		},
		"$defs": {
			"Metadata": {
				"allOf": [
					{
						"properties": {
							"created": {"type": "string"},
							"modified": {"type": "string"}
						}
					}
				]
			}
		}
	}`)

	po, err := ExtractPropertyOrder(schemaJSON, "complex.json")

	require.NoError(t, err, "ExtractPropertyOrder should not return error")

	// Check main properties
	mainOrder := po.GetOrder("complex.json")
	require.Len(t, mainOrder, 2, "main schema should have 2 properties")
	assert.Equal(t, "id", mainOrder[0], "first property should be id")
	assert.Equal(t, "data", mainOrder[1], "second property should be data")

	// Check nested allOf in $defs
	metadataOrder := po.GetOrder("complex.json#/$defs/Metadata#/allOf/0")
	require.Len(t, metadataOrder, 2, "Metadata allOf[0] should have 2 properties")
	assert.Equal(t, "created", metadataOrder[0], "first property should be created")
	assert.Equal(t, "modified", metadataOrder[1], "second property should be modified")
}

// TestExtractPropertyOrder_NoProperties tests schema without properties
func TestExtractPropertyOrder_NoProperties(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "string",
		"enum": ["a", "b", "c"]
	}`)

	po, err := ExtractPropertyOrder(schemaJSON, "enum.json")

	require.NoError(t, err, "ExtractPropertyOrder should not return error")

	order := po.GetOrder("enum.json")
	assert.Nil(t, order, "should not extract properties from non-object schema")
}

// TestExtractPropertyOrder_EmptyProperties tests schema with empty properties
func TestExtractPropertyOrder_EmptyProperties(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {}
	}`)

	po, err := ExtractPropertyOrder(schemaJSON, "empty.json")

	require.NoError(t, err, "ExtractPropertyOrder should not return error")

	order := po.GetOrder("empty.json")
	require.NotNil(t, order, "should extract empty property list")
	assert.Empty(t, order, "property order should be empty")
}

// TestExtractPropertyOrder_MultipleDefsLevels tests multiple levels of $defs
func TestExtractPropertyOrder_MultipleDefsLevels(t *testing.T) {
	schemaJSON := []byte(`{
		"$defs": {
			"User": {
				"properties": {
					"name": {"type": "string"},
					"age": {"type": "integer"}
				}
			},
			"Admin": {
				"properties": {
					"role": {"type": "string"},
					"permissions": {"type": "array"}
				}
			}
		}
	}`)

	po, err := ExtractPropertyOrder(schemaJSON, "users.json")

	require.NoError(t, err, "ExtractPropertyOrder should not return error")

	// Check User definition
	userOrder := po.GetOrder("users.json#/$defs/User")
	require.Len(t, userOrder, 2, "User should have 2 properties")
	assert.Equal(t, "name", userOrder[0], "first property should be name")
	assert.Equal(t, "age", userOrder[1], "second property should be age")

	// Check Admin definition
	adminOrder := po.GetOrder("users.json#/$defs/Admin")
	require.Len(t, adminOrder, 2, "Admin should have 2 properties")
	assert.Equal(t, "role", adminOrder[0], "first property should be role")
	assert.Equal(t, "permissions", adminOrder[1], "second property should be permissions")
}

// TestExtractPropertyOrder_InvalidJSON tests extracting from invalid JSON
func TestExtractPropertyOrder_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{invalid json`)

	_, err := ExtractPropertyOrder(invalidJSON, "invalid.json")

	assert.Error(t, err, "ExtractPropertyOrder should return error for invalid JSON")
}

// TestExtractPropertyOrder_OrderPreserved tests that actual insertion order is preserved
func TestExtractPropertyOrder_OrderPreserved(t *testing.T) {
	// This JSON has properties in a specific order: z, a, m
	schemaJSON := []byte(`{"type":"object","properties":{"z":{"type":"string"},"a":{"type":"string"},"m":{"type":"string"}}}`)

	po, err := ExtractPropertyOrder(schemaJSON, "order.json")

	require.NoError(t, err, "ExtractPropertyOrder should not return error")

	order := po.GetOrder("order.json")
	require.Len(t, order, 3, "should extract 3 properties")
	// Properties should be in insertion order (z, a, m), NOT alphabetical
	assert.Equal(t, "z", order[0], "first property should be z (insertion order)")
	assert.Equal(t, "a", order[1], "second property should be a (insertion order)")
	assert.Equal(t, "m", order[2], "third property should be m (insertion order)")
}
