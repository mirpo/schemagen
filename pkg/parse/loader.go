package parse

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ParseFile(path string) (*NamedSchema, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))

	var node *SchemaNode
	switch ext {
	case ".json":
		node, err = ParseJSON(f)
	case ".yaml", ".yml":
		node, err = ParseYAML(f)
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	name := node.Title
	if name == "" {
		base := filepath.Base(path)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	return &NamedSchema{
		Name:   name,
		Schema: node,
		Path:   filepath.Base(path),
	}, nil
}

func ParseDir(dir string) ([]*NamedSchema, error) {
	var schemas []*NamedSchema

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
			return nil
		}

		ns, err := ParseFile(path)
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("relative path for %s: %w", path, err)
		}
		ns.Path = relPath

		schemas = append(schemas, ns)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return schemas, nil
}
