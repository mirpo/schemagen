package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaptinlin/jsonschema"
	pkgerrors "github.com/mirpo/schemagen/pkg/errors"
	"github.com/mirpo/schemagen/pkg/naming"
	"gopkg.in/yaml.v3"
)

// Schema represents a loaded JSON Schema.
type Schema struct {
	Path          string
	RelativePath  string
	Name          string
	Compiled      *jsonschema.Schema
	PropertyOrder *PropertyOrder
}

// Loader loads JSON Schema files.
type Loader struct {
	compiler *jsonschema.Compiler
	baseDir  string
}

// NewLoader creates a new schema loader.
func NewLoader() *Loader {
	return &Loader{
		compiler: jsonschema.NewCompiler(),
	}
}

// Compiler returns the underlying JSON Schema compiler.
func (l *Loader) Compiler() *jsonschema.Compiler {
	return l.compiler
}

// Load loads schemas from a file or directory.
func (l *Loader) Load(path string) ([]*Schema, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	if info.IsDir() {
		l.baseDir = path
		return l.loadDirectory(path)
	}

	l.baseDir = filepath.Dir(path)
	s, err := l.loadFile(path)
	if err != nil {
		return nil, err
	}

	return []*Schema{s}, nil
}

func (l *Loader) loadDirectory(root string) ([]*Schema, error) {
	var schemas []*Schema

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !isSchemaFile(path) {
			return err
		}

		s, err := l.loadFile(path)
		if err != nil {
			return fmt.Errorf("loading %s: %w", path, err)
		}

		schemas = append(schemas, s)
		return nil
	})

	return schemas, err
}

func isSchemaFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func (l *Loader) loadFile(path string) (*Schema, error) {
	data, err := readAndNormalizeSchema(path)
	if err != nil {
		return nil, err
	}

	relPath, err := filepath.Rel(l.baseDir, path)
	if err != nil {
		return nil, schemaErr(path, "calculate relative path", err)
	}
	relPath = filepath.ToSlash(relPath)

	order, err := ExtractPropertyOrder(data, relPath)
	if err != nil {
		return nil, schemaErr(relPath, "extract property order", err)
	}

	compiled, err := l.compiler.Compile(data, relPath)
	if err != nil {
		return nil, schemaErr(relPath, "compile schema", err)
	}

	l.compiler.SetSchema(relPath, compiled)

	return &Schema{
		Path:          path,
		RelativePath:  relPath,
		Name:          deriveName(path, compiled),
		Compiled:      compiled,
		PropertyOrder: order,
	}, nil
}

func readAndNormalizeSchema(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, schemaErr(path, "read file", err)
	}

	if isYAMLFile(path) {
		raw, err = convertYAMLToJSON(raw)
		if err != nil {
			return nil, schemaErr(path, "convert YAML to JSON", err)
		}
	}

	return raw, nil
}

func schemaErr(path, msg string, err error) error {
	return &pkgerrors.SchemaError{
		Path:    path,
		Message: msg,
		Cause:   err,
	}
}

func deriveName(path string, schema *jsonschema.Schema) string {
	if schema.Title != nil && *schema.Title != "" {
		if naming.IsPascalCase(*schema.Title) {
			return *schema.Title
		}
		return naming.ToPascalCase(*schema.Title)
	}

	base := filepath.Base(path)
	return naming.ToPascalCase(strings.TrimSuffix(base, filepath.Ext(base)))
}

func isYAMLFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func convertYAMLToJSON(yamlData []byte) ([]byte, error) {
	var v interface{}
	if err := yaml.Unmarshal(yamlData, &v); err != nil {
		return nil, fmt.Errorf("unmarshal YAML: %w", err)
	}

	return json.Marshal(normalizeYAMLValue(v))
}

func normalizeYAMLValue(v interface{}) interface{} {
	switch x := v.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(x))
		for k, v := range x {
			if ks, ok := k.(string); ok {
				m[ks] = normalizeYAMLValue(v)
			}
		}
		return m

	case map[string]interface{}:
		m := make(map[string]interface{}, len(x))
		for k, v := range x {
			m[k] = normalizeYAMLValue(v)
		}
		return m

	case []interface{}:
		for i := range x {
			x[i] = normalizeYAMLValue(x[i])
		}
		return x

	default:
		return x
	}
}
