package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// PropertyOrder tracks the insertion order of properties in a JSON schema.
type PropertyOrder struct {
	// Map from schema path to ordered property names
	// Examples:
	//   "simple.json" -> ["id", "name", "email"]
	//   "simple.json#/allOf/0" -> ["createdAt", "updatedAt"]
	//   "simple.json#/$defs/Address" -> ["street", "city", "zip"]
	orders map[string][]string
}

// NewPropertyOrder creates a new PropertyOrder.
func NewPropertyOrder() *PropertyOrder {
	return &PropertyOrder{
		orders: make(map[string][]string),
	}
}

// GetOrder returns property names in order for a schema path.
// Returns nil if the path is not found.
func (po *PropertyOrder) GetOrder(path string) []string {
	return po.orders[path]
}

// OrderedField represents a JSON object field with its key and raw value.
type OrderedField struct {
	Key   string
	Value json.RawMessage
}

// DecodeOrderedObject decodes a JSON object preserving field order.
func DecodeOrderedObject(data []byte) ([]OrderedField, error) {
	dec := json.NewDecoder(bytes.NewReader(data))

	// Read opening `{`
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if tok != json.Delim('{') {
		return nil, fmt.Errorf("expected object")
	}

	var fields []OrderedField

	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}

		key := keyTok.(string)

		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}

		fields = append(fields, OrderedField{
			Key:   key,
			Value: raw,
		})
	}

	// Read closing `}`
	_, err = dec.Token()
	return fields, err
}

// ExtractPropertyOrder parses raw JSON to extract property order using recursive parsing.
// The basePath parameter is the schema file path (e.g., "simple.json").
func ExtractPropertyOrder(data []byte, basePath string) (*PropertyOrder, error) {
	po := NewPropertyOrder()

	// Decode the schema object preserving field order
	fields, err := DecodeOrderedObject(data)
	if err != nil {
		return nil, fmt.Errorf("decode schema: %w", err)
	}

	// Process the schema fields recursively
	extractFromFields(fields, basePath, po)

	return po, nil
}

// extractFromFields recursively extracts property order from schema fields.
func extractFromFields(fields []OrderedField, currentPath string, po *PropertyOrder) {
	for _, field := range fields {
		switch field.Key {
		case "properties":
			// Extract property names in order
			propFields, err := DecodeOrderedObject(field.Value)
			if err == nil {
				propNames := make([]string, len(propFields))
				for i, pf := range propFields {
					propNames[i] = pf.Key
				}
				po.orders[currentPath] = propNames
			}

		case "allOf", "oneOf", "anyOf":
			// Extract from array elements
			extractFromArray(field.Value, currentPath+"#/"+field.Key, po)

		case "$defs", "definitions":
			// Extract from definitions
			extractFromDefs(field.Value, currentPath+"#/"+field.Key, po)
		}
	}
}

// extractFromArray extracts property order from an array (allOf/oneOf/anyOf).
func extractFromArray(data json.RawMessage, arrayPath string, po *PropertyOrder) {
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return
	}

	for i, elem := range arr {
		elemPath := fmt.Sprintf("%s/%d", arrayPath, i)
		fields, err := DecodeOrderedObject(elem)
		if err != nil {
			continue
		}
		extractFromFields(fields, elemPath, po)
	}
}

// extractFromDefs extracts property order from $defs.
func extractFromDefs(data json.RawMessage, defsPath string, po *PropertyOrder) {
	fields, err := DecodeOrderedObject(data)
	if err != nil {
		return
	}

	for _, field := range fields {
		defPath := fmt.Sprintf("%s/%s", defsPath, field.Key)
		defFields, err := DecodeOrderedObject(field.Value)
		if err != nil {
			continue
		}
		extractFromFields(defFields, defPath, po)
	}
}
