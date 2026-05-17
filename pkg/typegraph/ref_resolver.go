package typegraph

import (
	"path/filepath"
	"strings"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/constants"
	"github.com/mirpo/schemagen/pkg/naming"
	"github.com/mirpo/schemagen/pkg/schema"
)

type refResolver struct {
	compiler      *jsonschema.Compiler
	currentSchema *jsonschema.Schema
}

func newRefResolver(compiler *jsonschema.Compiler) *refResolver {
	return &refResolver{
		compiler: compiler,
	}
}

func (r *refResolver) setCurrentSchema(schema *jsonschema.Schema) {
	r.currentSchema = schema
}

func (r *refResolver) extractTypeName(ref string) string {
	// Handle root self-reference "#"
	if ref == schema.SelfRef {
		if r.currentSchema != nil {
			if r.currentSchema.Title != nil && *r.currentSchema.Title != "" {
				return *r.currentSchema.Title
			}
		}
		return "Schema"
	}

	// Handle internal $defs references
	if strings.HasPrefix(ref, schema.DefsPrefix) {
		defName := strings.TrimPrefix(ref, schema.DefsPrefix)
		return naming.ToPascalCase(defName)
	}

	// Handle external file references - try multiple normalized variants
	for _, variant := range normalizeRefPath(ref) {
		if refSchema, err := r.compiler.Schema(variant); err == nil && refSchema != nil {
			return r.deriveTypeName(refSchema, ref)
		}
	}

	// Fall back to extracting type name from filename
	return extractTypeNameFromFilename(ref)
}

func (r *refResolver) deriveTypeName(refSchema *jsonschema.Schema, refURI string) string {
	if refSchema.Title != nil && *refSchema.Title != "" {
		title := *refSchema.Title
		if naming.IsPascalCase(title) {
			return title
		}
		return naming.ToPascalCase(title)
	}

	if refURI != "" {
		name := filepath.Base(refURI)
		name = strings.TrimSuffix(name, constants.ExtJSON)
		return naming.ToPascalCase(name)
	}

	return "Unknown"
}

// normalizeRefPath normalizes a $ref path for schema lookup.
// Returns deduplicated variants to try during schema lookup.
func normalizeRefPath(ref string) []string {
	seen := make(map[string]bool)
	variants := []string{ref}
	seen[ref] = true

	add := func(v string) {
		if v != "" && !seen[v] {
			seen[v] = true
			variants = append(variants, v)
		}
	}

	cleaned := filepath.Clean(ref)
	add(cleaned)
	add(filepath.ToSlash(cleaned))

	if strings.HasPrefix(ref, "./") {
		add(strings.TrimPrefix(ref, "./"))
	}

	stripped := ref
	for strings.HasPrefix(stripped, "../") {
		stripped = strings.TrimPrefix(stripped, "../")
	}
	add(stripped)

	if !strings.HasPrefix(ref, "./") && !strings.HasPrefix(ref, "../") {
		add("./" + ref)
	}

	return variants
}

// extractTypeNameFromFilename extracts a type name from a file path reference.
// Used as a fallback when the schema cannot be resolved.
func extractTypeNameFromFilename(ref string) string {
	// Strip "./" prefix
	cleaned := strings.TrimPrefix(ref, "./")

	// Get base filename without path
	base := filepath.Base(cleaned)

	// Remove extension
	name := strings.TrimSuffix(base, filepath.Ext(base))

	// Convert to PascalCase
	return naming.ToPascalCase(name)
}
