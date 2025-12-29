package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoveryOptions controls schema discovery.
type DiscoveryOptions struct {
	Input      string
	Extensions []string
}

// DiscoveredSchema represents a discovered schema file.
//
// SchemaRoot is the directory used to resolve $ref paths:
//   - for directory discovery: the discovery root
//   - for single-file discovery: the file’s parent directory
type DiscoveredSchema struct {
	AbsolutePath string
	RelativePath string
	SchemaRoot   string
}

var defaultExtensions = []string{".json", ".yaml", ".yml"}

func DiscoverSchemas(opts DiscoveryOptions) ([]DiscoveredSchema, error) {
	exts := normalizeExtensions(opts.Extensions)
	if len(exts) == 0 {
		exts = defaultExtensions
	}

	absPath, err := filepath.Abs(opts.Input)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat input path: %w", err)
	}

	if info.IsDir() {
		return discoverFromDirectory(absPath, exts)
	}

	return discoverSingleFile(absPath, exts)
}

func discoverSingleFile(absPath string, extensions []string) ([]DiscoveredSchema, error) {
	if !hasValidExtension(absPath, extensions) {
		return nil, fmt.Errorf("file %s does not match allowed extensions", absPath)
	}

	return []DiscoveredSchema{
		{
			AbsolutePath: absPath,
			RelativePath: filepath.Base(absPath),
			SchemaRoot:   filepath.Dir(absPath),
		},
	}, nil
}

func discoverFromDirectory(root string, extensions []string) ([]DiscoveredSchema, error) {
	var schemas []DiscoveredSchema

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !hasValidExtension(path, extensions) {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path: %w", err)
		}

		schemas = append(schemas, DiscoveredSchema{
			AbsolutePath: path,
			RelativePath: filepath.ToSlash(rel),
			SchemaRoot:   root,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].RelativePath < schemas[j].RelativePath
	})

	return schemas, nil
}

func hasValidExtension(path string, extensions []string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, e := range extensions {
		if ext == e {
			return true
		}
	}
	return false
}

func normalizeExtensions(exts []string) []string {
	if len(exts) == 0 {
		return nil
	}
	out := make([]string, len(exts))
	for i, e := range exts {
		out[i] = strings.ToLower(e)
	}
	return out
}
