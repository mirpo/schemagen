package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DiscoveryOptions struct {
	Input      string
	Extensions []string
}

type DiscoveredSchema struct {
	AbsolutePath string
	RelativePath string
	SchemaRoot   string
}

var defaultExtensions = []string{".json", ".yaml", ".yml"}

func DiscoverSchemas(opts DiscoveryOptions) ([]DiscoveredSchema, error) {
	if len(opts.Extensions) == 0 {
		opts.Extensions = defaultExtensions
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
		return discoverFromDirectory(absPath, opts.Extensions)
	}
	return discoverSingleFile(absPath, opts.Extensions)
}

func discoverSingleFile(absPath string, extensions []string) ([]DiscoveredSchema, error) {
	if !hasValidExtension(absPath, extensions) {
		return nil, fmt.Errorf("file %s does not have a valid schema extension (.json, .yaml, .yml)", absPath)
	}

	schemaRoot := filepath.Dir(absPath)
	fileName := filepath.Base(absPath)

	return []DiscoveredSchema{
		{
			AbsolutePath: absPath,
			RelativePath: fileName,
			SchemaRoot:   schemaRoot,
		},
	}, nil
}

func discoverFromDirectory(rootPath string, extensions []string) ([]DiscoveredSchema, error) {
	var schemas []DiscoveredSchema
	seenPaths := make(map[string]bool)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !hasValidExtension(path, extensions) {
			return nil
		}

		if seenPaths[path] {
			return nil
		}
		seenPaths[path] = true

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path: %w", err)
		}

		schemas = append(schemas, DiscoveredSchema{
			AbsolutePath: path,
			RelativePath: relPath,
			SchemaRoot:   rootPath,
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
	for _, validExt := range extensions {
		if ext == validExt {
			return true
		}
	}
	return false
}
