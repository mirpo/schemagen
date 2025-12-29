package loader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

/*
Public types
*/
type ExternalRef struct {
	FilePath string
	Fragment string
}

/*
Extract external $ref references
*/
func ExtractExternalRefs(schemaPath string) ([]ExternalRef, error) {
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	base := filepath.Dir(schemaPath)
	refs := map[string]ExternalRef{}

	extractRefsRecursive(root, base, refs)

	out := make([]ExternalRef, 0, len(refs))
	for _, r := range refs {
		out = append(out, r)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].FilePath == out[j].FilePath {
			return out[i].Fragment < out[j].Fragment
		}
		return out[i].FilePath < out[j].FilePath
	})

	return out, nil
}

func extractRefsRecursive(node any, basePath string, acc map[string]ExternalRef) {
	switch v := node.(type) {
	case map[string]any:
		if raw, ok := v["$ref"]; ok {
			if s, ok := raw.(string); ok {
				if ref := parseRef(s, basePath); ref != nil {
					key := ref.FilePath + "#" + ref.Fragment
					acc[key] = *ref
				}
			}
		}
		for _, val := range v {
			extractRefsRecursive(val, basePath, acc)
		}

	case []any:
		for _, item := range v {
			extractRefsRecursive(item, basePath, acc)
		}
	}
}

func parseRef(refStr, basePath string) *ExternalRef {
	if strings.HasPrefix(refStr, "#") {
		return nil
	}

	file, frag, _ := strings.Cut(refStr, "#")
	if file == "" {
		return nil
	}

	abs, err := ResolveRefPath(file, basePath)
	if err != nil {
		return nil
	}

	return &ExternalRef{
		FilePath: abs,
		Fragment: frag,
	}
}

func ResolveRefPath(refPath, basePath string) (string, error) {
	if filepath.IsAbs(refPath) {
		return filepath.Clean(refPath), nil
	}

	full := filepath.Join(basePath, refPath)
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", fmt.Errorf("failed to resolve ref path: %w", err)
	}
	return filepath.Clean(abs), nil
}

/*
Recursive schema loading
*/

func LoadSchemasRecursive(initialPath string) ([]DiscoveredSchema, error) {
	abs, err := filepath.Abs(initialPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve initial path: %w", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("failed to stat initial path: %w", err)
	}

	if info.IsDir() {
		return DiscoverSchemas(DiscoveryOptions{Input: abs})
	}

	return loadSingleFileWithDeps(abs)
}

func loadSingleFileWithDeps(entry string) ([]DiscoveredSchema, error) {
	root := filepath.Dir(entry)

	queue := []string{entry}
	seen := map[string]bool{}
	out := map[string]DiscoveredSchema{}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if seen[cur] {
			continue
		}
		seen[cur] = true

		rel, err := filepath.Rel(root, cur)
		if err != nil {
			rel = filepath.Base(cur)
		}

		out[cur] = DiscoveredSchema{
			AbsolutePath: cur,
			RelativePath: rel,
			SchemaRoot:   root,
		}

		refs, err := ExtractExternalRefs(cur)
		if err != nil {
			continue // tolerate broken schemas
		}

		for _, r := range refs {
			if !seen[r.FilePath] {
				queue = append(queue, r.FilePath)
			}
		}
	}

	result := make([]DiscoveredSchema, 0, len(out))
	for _, s := range out {
		result = append(result, s)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].RelativePath < result[j].RelativePath
	})

	return result, nil
}
