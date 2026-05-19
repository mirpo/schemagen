package parse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

func ParseYAML(r io.Reader) (*SchemaNode, error) {
	var raw any
	dec := yaml.NewDecoder(r)
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	normalized := normalizeYAML(raw)

	jsonBytes, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("convert yaml to json: %w", err)
	}

	return ParseJSON(bytes.NewReader(jsonBytes))
}

func normalizeYAML(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, v := range val {
			out[k] = normalizeYAML(v)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(val))
		for k, v := range val {
			out[fmt.Sprintf("%v", k)] = normalizeYAML(v)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, v := range val {
			out[i] = normalizeYAML(v)
		}
		return out
	default:
		return v
	}
}
