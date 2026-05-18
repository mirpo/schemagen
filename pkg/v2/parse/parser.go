package parse

import (
	"encoding/json"
	"fmt"
	"io"
)

func ParseJSON(r io.Reader) (*SchemaNode, error) {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return parseNode(dec)
}

func parseNode(dec *json.Decoder) (*SchemaNode, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}

	switch v := tok.(type) {
	case json.Delim:
		if v != '{' {
			return nil, fmt.Errorf("expected '{', got %v", v)
		}
		return parseObjectBody(dec)
	case bool:
		node := &SchemaNode{}
		if !v {
			node.Type = StringOrSlice{"never"}
		}
		return node, nil
	default:
		return nil, fmt.Errorf("unexpected token %T: %v", tok, tok)
	}
}

func decodeAdditionalProperties(dec *json.Decoder) (*AdditionalProperties, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}

	switch v := tok.(type) {
	case bool:
		return &AdditionalProperties{Allowed: v}, nil
	case json.Delim:
		if v != '{' {
			return nil, fmt.Errorf("expected '{' or bool for additionalProperties, got %v", v)
		}
		node, err := parseObjectBody(dec)
		if err != nil {
			return nil, err
		}
		return &AdditionalProperties{Allowed: true, Schema: node}, nil
	default:
		return nil, fmt.Errorf("expected bool or object for additionalProperties, got %T", tok)
	}
}

func parseObjectBody(dec *json.Decoder) (*SchemaNode, error) {
	node := &SchemaNode{}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("expected string key, got %T", keyTok)
		}

		switch key {
		case "$schema":
			node.Schema, err = decodeString(dec)
		case "$id":
			node.ID, err = decodeString(dec)
		case "$ref":
			node.Ref, err = decodeString(dec)
		case "title":
			node.Title, err = decodeString(dec)
		case "description":
			node.Description, err = decodeString(dec)
		case "type":
			node.Type, err = decodeStringOrSlice(dec)
		case "format":
			node.Format, err = decodeString(dec)
		case "properties":
			node.Properties, err = decodeNamedSchemas(dec)
		case "required":
			node.Required, err = decodeStringSlice(dec)
		case "additionalProperties":
			node.AdditionalProperties, err = decodeAdditionalProperties(dec)
		case "items":
			node.Items, err = parseNode(dec)
		case "allOf":
			node.AllOf, err = decodeSchemaSlice(dec)
		case "anyOf":
			node.AnyOf, err = decodeSchemaSlice(dec)
		case "oneOf":
			node.OneOf, err = decodeSchemaSlice(dec)
		case "enum":
			node.Enum, err = decodeAnySlice(dec)
		case "const":
			node.Const, err = decodeAny(dec)
			node.HasConst = true
		case "prefixItems":
			node.PrefixItems, err = decodeSchemaSlice(dec)
		case "$defs", "definitions":
			node.Defs, err = decodeNamedSchemas(dec)
		case "minLength":
			node.MinLength, err = decodeIntPtr(dec)
		case "maxLength":
			node.MaxLength, err = decodeIntPtr(dec)
		case "pattern":
			var s string
			s, err = decodeString(dec)
			if err == nil {
				node.Pattern = &s
			}
		case "minimum":
			node.Minimum, err = decodeFloatPtr(dec)
		case "maximum":
			node.Maximum, err = decodeFloatPtr(dec)
		case "exclusiveMinimum":
			node.ExclusiveMinimum, err = decodeExclusiveLimit(dec)
		case "exclusiveMaximum":
			node.ExclusiveMaximum, err = decodeExclusiveLimit(dec)
		case "minItems":
			node.MinItems, err = decodeIntPtr(dec)
		case "maxItems":
			node.MaxItems, err = decodeIntPtr(dec)
		case "multipleOf":
			node.MultipleOf, err = decodeFloatPtr(dec)
		default:
			err = skipValue(dec)
		}
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", key, err)
		}
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return node, nil
}

func decodeExclusiveLimit(dec *json.Decoder) (*float64, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch v := tok.(type) {
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return nil, err
		}
		return &f, nil
	case bool:
		return nil, nil
	default:
		return nil, fmt.Errorf("expected number or bool for exclusive limit, got %T", tok)
	}
}

func decodeString(dec *json.Decoder) (string, error) {
	tok, err := dec.Token()
	if err != nil {
		return "", err
	}
	s, ok := tok.(string)
	if !ok {
		return "", fmt.Errorf("expected string, got %T", tok)
	}
	return s, nil
}

func decodeStringOrSlice(dec *json.Decoder) (StringOrSlice, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch v := tok.(type) {
	case string:
		return StringOrSlice{v}, nil
	case json.Delim:
		if v != '[' {
			return nil, fmt.Errorf("expected '[' for type array, got %v", v)
		}
		var result StringOrSlice
		for dec.More() {
			s, err := decodeString(dec)
			if err != nil {
				return nil, err
			}
			result = append(result, s)
		}
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, fmt.Errorf("expected string or array for type, got %T", tok)
	}
}

func decodeStringSlice(dec *json.Decoder) ([]string, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '[' {
		return nil, fmt.Errorf("expected '[', got %v", tok)
	}
	var result []string
	for dec.More() {
		s, err := decodeString(dec)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return result, nil
}

func decodeNamedSchemas(dec *json.Decoder) ([]NamedSchema, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("expected '{', got %v", tok)
	}
	var result []NamedSchema
	for dec.More() {
		name, err := decodeString(dec)
		if err != nil {
			return nil, err
		}
		schema, err := parseNode(dec)
		if err != nil {
			return nil, err
		}
		result = append(result, NamedSchema{Name: name, Schema: schema})
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return result, nil
}

func decodeSchemaSlice(dec *json.Decoder) ([]*SchemaNode, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '[' {
		return nil, fmt.Errorf("expected '[', got %v", tok)
	}
	var result []*SchemaNode
	for dec.More() {
		n, err := parseNode(dec)
		if err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return result, nil
}

func decodeAnySlice(dec *json.Decoder) ([]any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '[' {
		return nil, fmt.Errorf("expected '[', got %v", tok)
	}
	var result []any
	for dec.More() {
		v, err := decodeAny(dec)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return result, nil
}

func decodeAny(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch v := tok.(type) {
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i, nil
		}
		return v.Float64()
	case string:
		return v, nil
	case bool:
		return v, nil
	case nil:
		return nil, nil
	case json.Delim:
		return decodeComplexValue(dec, v)
	default:
		return tok, nil
	}
}

func decodeComplexValue(dec *json.Decoder, d json.Delim) (any, error) {
	switch d {
	case '{':
		obj := make(map[string]any)
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyTok.(string)
			if !ok {
				return nil, fmt.Errorf("expected string key, got %T", keyTok)
			}
			val, err := decodeAny(dec)
			if err != nil {
				return nil, err
			}
			obj[key] = val
		}
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
		return obj, nil
	case '[':
		var arr []any
		for dec.More() {
			val, err := decodeAny(dec)
			if err != nil {
				return nil, err
			}
			arr = append(arr, val)
		}
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
		return arr, nil
	}
	return nil, nil
}

func decodeIntPtr(dec *json.Decoder) (*int, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	n, ok := tok.(json.Number)
	if !ok {
		return nil, fmt.Errorf("expected number, got %T", tok)
	}
	i, err := n.Int64()
	if err != nil {
		return nil, err
	}
	v := int(i)
	return &v, nil
}

func decodeFloatPtr(dec *json.Decoder) (*float64, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	n, ok := tok.(json.Number)
	if !ok {
		return nil, fmt.Errorf("expected number, got %T", tok)
	}
	f, err := n.Float64()
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func skipValue(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := tok.(json.Delim); ok {
		return skipAfterDelim(dec, d)
	}
	return nil
}

func skipAfterDelim(dec *json.Decoder, d json.Delim) error {
	switch d {
	case '{':
		for dec.More() {
			if _, err := dec.Token(); err != nil {
				return err
			}
			if err := skipValue(dec); err != nil {
				return err
			}
		}
		_, err := dec.Token()
		return err
	case '[':
		for dec.More() {
			if err := skipValue(dec); err != nil {
				return err
			}
		}
		_, err := dec.Token()
		return err
	}
	return nil
}
