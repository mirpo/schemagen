package schema

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
)

// PropertyOrder tracks insertion order of "properties" and "$defs" blocks in a schema.
type PropertyOrder struct {
	// key: schema path (e.g. "a.json#/allOf/0")
	// val: ordered property names
	orders map[string][]string

	// key: schema path (e.g. "a.json")
	// val: ordered $defs names
	defsOrders map[string][]string
}

// newPropertyOrder creates an empty PropertyOrder.
func newPropertyOrder() *PropertyOrder {
	return &PropertyOrder{
		orders:     make(map[string][]string),
		defsOrders: make(map[string][]string),
	}
}

// NewPropertyOrderWith creates a PropertyOrder pre-populated with the given property orders.
func NewPropertyOrderWith(orders map[string][]string) *PropertyOrder {
	po := newPropertyOrder()
	for k, v := range orders {
		po.orders[k] = v
	}
	return po
}

// GetOrder returns ordered property names for a schema path.
func (po *PropertyOrder) GetOrder(path string) []string {
	return po.orders[path]
}

// GetDefsOrder returns ordered $defs names for a schema path.
func (po *PropertyOrder) GetDefsOrder(path string) []string {
	return po.defsOrders[path]
}

// orderedField represents one JSON object field with preserved order.
type orderedField struct {
	Key   string
	Value json.RawMessage
}

// decodeOrderedObject decodes a JSON object while preserving field order.
func decodeOrderedObject(data []byte) ([]orderedField, error) {
	dec := json.NewDecoder(bytes.NewReader(data))

	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if tok != json.Delim('{') {
		return nil, fmt.Errorf("expected JSON object")
	}

	var fields []orderedField

	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}

		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("expected string key")
		}

		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}

		fields = append(fields, orderedField{
			Key:   key,
			Value: raw,
		})
	}

	_, err = dec.Token() // closing '}'
	return fields, err
}

// extractPropertyOrder extracts property order from raw JSON schema.
func extractPropertyOrder(data []byte, basePath string) (*PropertyOrder, error) {
	fields, err := decodeOrderedObject(data)
	if err != nil {
		return nil, fmt.Errorf("decode schema: %w", err)
	}

	po := newPropertyOrder()
	extractFromFields(fields, basePath, po)

	return po, nil
}

func extractFromFields(fields []orderedField, path string, po *PropertyOrder) {
	for _, f := range fields {
		switch f.Key {
		case "properties":
			extractProperties(f.Value, path, po)

		case "allOf", "oneOf", "anyOf":
			extractFromArray(f.Value, path+"#/"+f.Key, po)

		case "$defs", "definitions":
			extractFromDefinitions(f.Value, path+"#/"+f.Key, po)
		}
	}
}

func extractProperties(data json.RawMessage, path string, po *PropertyOrder) {
	fields, err := decodeOrderedObject(data)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("failed to decode properties")
		return
	}

	names := make([]string, 0, len(fields))
	for _, f := range fields {
		names = append(names, f.Key)
	}

	po.orders[path] = names
}

func extractFromArray(data json.RawMessage, basePath string, po *PropertyOrder) {
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		log.Debug().Err(err).Str("path", basePath).Msg("failed to unmarshal array")
		return
	}

	for i, elem := range arr {
		elemPath := fmt.Sprintf("%s/%d", basePath, i)

		fields, err := decodeOrderedObject(elem)
		if err != nil {
			log.Debug().Err(err).Str("path", elemPath).Msg("failed to decode array element")
			continue
		}

		extractFromFields(fields, elemPath, po)
	}
}

func extractFromDefinitions(data json.RawMessage, basePath string, po *PropertyOrder) {
	fields, err := decodeOrderedObject(data)
	if err != nil {
		log.Debug().Err(err).Str("path", basePath).Msg("failed to decode definitions")
		return
	}

	// Store the order of $defs names
	defNames := make([]string, 0, len(fields))
	for _, def := range fields {
		defNames = append(defNames, def.Key)
	}
	po.defsOrders[basePath] = defNames

	for _, def := range fields {
		defPath := fmt.Sprintf("%s/%s", basePath, def.Key)

		defFields, err := decodeOrderedObject(def.Value)
		if err != nil {
			log.Debug().Err(err).Str("path", defPath).Msg("failed to decode definition")
			continue
		}

		extractFromFields(defFields, defPath, po)
	}
}
