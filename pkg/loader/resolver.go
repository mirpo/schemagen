package loader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ExternalRef struct {
	FilePath string
	Fragment string
}

func ExtractExternalRefs(schemaPath string) ([]ExternalRef, error) {
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	basePath := filepath.Dir(schemaPath)
	refs := make(map[string]ExternalRef)

	extractRefsRecursive(schema, basePath, refs)

	result := make([]ExternalRef, 0, len(refs))
	for _, ref := range refs {
		result = append(result, ref)
	}

	return result, nil
}

func extractRefsRecursive(node interface{}, basePath string, refs map[string]ExternalRef) {
	switch v := node.(type) {
	case map[string]interface{}:
		if refValue, ok := v["$ref"]; ok {
			if refStr, ok := refValue.(string); ok {
				if ref := parseRef(refStr, basePath); ref != nil {
					key := ref.FilePath
					if ref.Fragment != "" {
						key = ref.FilePath + "#" + ref.Fragment
					}
					refs[key] = *ref
				}
			}
		}

		for _, value := range v {
			extractRefsRecursive(value, basePath, refs)
		}

	case []interface{}:
		for _, item := range v {
			extractRefsRecursive(item, basePath, refs)
		}
	}
}

func parseRef(refStr string, basePath string) *ExternalRef {
	if strings.HasPrefix(refStr, "#") {
		return nil
	}

	parts := strings.SplitN(refStr, "#", 2)
	filePart := parts[0]
	fragment := ""
	if len(parts) > 1 {
		fragment = parts[1]
	}

	if filePart == "" {
		return nil
	}

	absPath, err := ResolveRefPath(filePart, basePath)
	if err != nil {
		return nil
	}

	return &ExternalRef{
		FilePath: absPath,
		Fragment: fragment,
	}
}

func ResolveRefPath(refPath string, basePath string) (string, error) {
	if filepath.IsAbs(refPath) {
		return filepath.Clean(refPath), nil
	}

	fullPath := filepath.Join(basePath, refPath)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve ref path: %w", err)
	}

	return filepath.Clean(absPath), nil
}

func LoadSchemasRecursive(initialPath string) ([]DiscoveredSchema, error) {
	absPath, err := filepath.Abs(initialPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve initial path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat initial path: %w", err)
	}

	if info.IsDir() {
		return DiscoverSchemas(DiscoveryOptions{Input: absPath})
	}

	return loadSingleFileWithDeps(absPath)
}

func loadSingleFileWithDeps(initialPath string) ([]DiscoveredSchema, error) {
	schemaRoot := filepath.Dir(initialPath)
	discovered := make(map[string]DiscoveredSchema)
	queue := []string{initialPath}
	processed := make(map[string]bool)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if processed[current] {
			continue
		}
		processed[current] = true

		relPath, err := filepath.Rel(schemaRoot, current)
		if err != nil {
			relPath = filepath.Base(current)
		}

		discovered[current] = DiscoveredSchema{
			AbsolutePath: current,
			RelativePath: relPath,
			SchemaRoot:   schemaRoot,
		}

		refs, err := ExtractExternalRefs(current)
		if err != nil {
			continue
		}

		for _, ref := range refs {
			if !processed[ref.FilePath] {
				queue = append(queue, ref.FilePath)
			}
		}
	}

	result := make([]DiscoveredSchema, 0, len(discovered))
	for _, schema := range discovered {
		result = append(result, schema)
	}

	return result, nil
}
