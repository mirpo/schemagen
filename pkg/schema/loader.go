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
	Path          string             // Original file path
	RelativePath  string             // Relative path (for $id and refs)
	Name          string             // Derived name (from file or title)
	Compiled      *jsonschema.Schema // Compiled schema
	PropertyOrder *PropertyOrder     // Property order extracted from JSON
}

// Loader loads JSON Schema files.
type Loader struct {
	compiler *jsonschema.Compiler
	baseDir  string // Base directory for relative path calculation
}

// NewLoader creates a new schema loader.
func NewLoader() *Loader {
	return &Loader{
		compiler: jsonschema.NewCompiler(),
	}
}

// Compiler returns the underlying JSON Schema compiler.
// This is used by the typegraph builder to resolve $refs.
func (l *Loader) Compiler() *jsonschema.Compiler {
	return l.compiler
}

// Load loads schemas from a file or directory.
// Returns a slice of compiled schemas.
func (l *Loader) Load(path string) ([]*Schema, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	// Set base directory for relative path calculation
	if info.IsDir() {
		l.baseDir = path
		return l.loadDirectory(path)
	}

	// For single file, use parent directory as base
	l.baseDir = filepath.Dir(path)
	schema, err := l.loadFile(path)
	if err != nil {
		return nil, err
	}

	return []*Schema{schema}, nil
}

// loadDirectory loads all .json files from a directory.
func (l *Loader) loadDirectory(dir string) ([]*Schema, error) {
	var schemas []*Schema

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only load schema files (.json, .yaml, .yml)
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
			return nil
		}

		schema, err := l.loadFile(path)
		if err != nil {
			return fmt.Errorf("loading %s: %w", path, err)
		}

		schemas = append(schemas, schema)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return schemas, nil
}

// loadFile loads and compiles a single JSON Schema file.
func (l *Loader) loadFile(path string) (*Schema, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &pkgerrors.SchemaError{Path: path, Message: "read file", Cause: err}
	}

	// Convert YAML to JSON if needed
	if isYAMLFile(path) {
		data, err = convertYAMLToJSON(data)
		if err != nil {
			return nil, &pkgerrors.SchemaError{Path: path, Message: "convert YAML to JSON", Cause: err}
		}
	}

	// Calculate relative path from base directory
	relPath, err := filepath.Rel(l.baseDir, path)
	if err != nil {
		return nil, &pkgerrors.SchemaError{Path: path, Message: "calculate relative path", Cause: err}
	}

	// Normalize path separators to forward slashes (for cross-platform consistency)
	relPath = filepath.ToSlash(relPath)

	// Extract property order from raw JSON (before jsonschema library processes it)
	propertyOrder, err := ExtractPropertyOrder(data, relPath)
	if err != nil {
		return nil, &pkgerrors.SchemaError{Path: relPath, Message: "extract property order", Cause: err}
	}

	// Parse JSON to inject $id field
	var schemaObj map[string]interface{}
	if err := json.Unmarshal(data, &schemaObj); err != nil {
		return nil, &pkgerrors.SchemaError{Path: relPath, Message: "parse JSON", Cause: err}
	}

	// Inject $id if not present (for $ref resolution)
	if _, hasID := schemaObj["$id"]; !hasID {
		schemaObj["$id"] = relPath
	}

	// Re-marshal with $id
	data, err = json.Marshal(schemaObj)
	if err != nil {
		return nil, &pkgerrors.SchemaError{Path: relPath, Message: "marshal JSON", Cause: err}
	}

	// Compile schema with URI (for registration)
	compiled, err := l.compiler.Compile(data, relPath)
	if err != nil {
		return nil, &pkgerrors.SchemaError{Path: relPath, Message: "compile schema", Cause: err}
	}

	// Register schema for $ref resolution
	l.compiler.SetSchema(relPath, compiled)

	// Derive name from filename or schema title
	name := deriveName(path, compiled)

	return &Schema{
		Path:          path,
		RelativePath:  relPath,
		Name:          name,
		Compiled:      compiled,
		PropertyOrder: propertyOrder,
	}, nil
}

// deriveName extracts a suitable type name from the schema.
func deriveName(path string, schema *jsonschema.Schema) string {
	// Try title first - use as-is if already properly formatted
	if schema.Title != nil && *schema.Title != "" {
		title := *schema.Title
		// If title looks like it's already in PascalCase, use it directly
		if naming.IsPascalCase(title) {
			return title
		}
		return naming.ToPascalCase(title)
	}

	// Fall back to filename without extension
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return naming.ToPascalCase(name)
}

// isYAMLFile checks if a file path represents a YAML file based on extension.
func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

// convertYAMLToJSON converts YAML data to JSON format.
func convertYAMLToJSON(yamlData []byte) ([]byte, error) {
	var data interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal YAML: %w", err)
	}

	// Convert YAML types to JSON-compatible types
	data = convertYAMLValue(data)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal to JSON: %w", err)
	}

	return jsonData, nil
}

// convertYAMLValue recursively converts YAML-specific types to JSON-compatible types.
// This handles the case where yaml.v3 returns map[interface{}]interface{} which JSON doesn't support.
func convertYAMLValue(val interface{}) interface{} {
	switch v := val.(type) {
	case map[interface{}]interface{}:
		// Convert map[interface{}]interface{} to map[string]interface{}
		m := make(map[string]interface{})
		for key, value := range v {
			if keyStr, ok := key.(string); ok {
				m[keyStr] = convertYAMLValue(value)
			}
		}
		return m
	case map[string]interface{}:
		// Recursively convert values
		m := make(map[string]interface{})
		for key, value := range v {
			m[key] = convertYAMLValue(value)
		}
		return m
	case []interface{}:
		// Recursively convert array elements
		arr := make([]interface{}, len(v))
		for i, elem := range v {
			arr[i] = convertYAMLValue(elem)
		}
		return arr
	default:
		// Primitive types (string, int, float, bool) pass through
		return v
	}
}
