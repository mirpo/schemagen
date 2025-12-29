package typegraph

import (
	"path/filepath"
	"strings"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/naming"
)

// RefResolver handles $ref resolution and type name derivation.
type RefResolver struct {
	compiler      *jsonschema.Compiler
	currentSchema *jsonschema.Schema
}

// NewRefResolver creates a new ref resolver.
func NewRefResolver(compiler *jsonschema.Compiler) *RefResolver {
	return &RefResolver{
		compiler: compiler,
	}
}

// SetCurrentSchema sets the current root schema for self-reference resolution.
func (r *RefResolver) SetCurrentSchema(schema *jsonschema.Schema) {
	r.currentSchema = schema
}

// ExtractTypeName extracts a type name from a $ref string.
func (r *RefResolver) ExtractTypeName(ref string) string {
	// Handle root self-reference "#"
	if ref == "#" {
		if r.currentSchema != nil {
			if r.currentSchema.Title != nil && *r.currentSchema.Title != "" {
				return *r.currentSchema.Title
			}
		}
		return "Schema"
	}

	// Handle internal $defs references
	if strings.HasPrefix(ref, "#/$defs/") {
		defName := strings.TrimPrefix(ref, "#/$defs/")
		return naming.ToPascalCase(defName)
	}

	// Handle external file references - try multiple normalized variants
	for _, variant := range normalizeRefPath(ref) {
		if refSchema, err := r.compiler.GetSchema(variant); err == nil && refSchema != nil {
			return r.DeriveTypeName(refSchema, ref)
		}
	}

	// Fall back to extracting type name from filename
	return extractTypeNameFromFilename(ref)
}

// DeriveTypeName derives a type name from a referenced schema.
func (r *RefResolver) DeriveTypeName(refSchema *jsonschema.Schema, refURI string) string {
	// Try to get the title from the schema - use as-is if already PascalCase
	if refSchema.Title != nil && *refSchema.Title != "" {
		title := *refSchema.Title
		if naming.IsPascalCase(title) {
			return title
		}
		return naming.ToPascalCase(title)
	}

	// Fall back to deriving from the URI
	if refURI != "" {
		name := refURI
		if lastSlash := len(refURI) - 1; lastSlash >= 0 {
			for i := len(refURI) - 1; i >= 0; i-- {
				if refURI[i] == '/' {
					name = refURI[i+1:]
					break
				}
			}
		}

		// Remove .json extension
		if len(name) > 5 && name[len(name)-5:] == ".json" {
			name = name[:len(name)-5]
		}

		return naming.ToPascalCase(name)
	}

	return "Unknown"
}

// normalizeRefPath normalizes a $ref path for schema lookup.
// Handles various relative path formats: "./", "../", "../../", etc.
// Returns multiple normalized variants to try during schema lookup.
func normalizeRefPath(ref string) []string {
	variants := []string{ref} // Always try original first

	// Clean the path to normalize "./" "../" etc using Go's filepath
	cleaned := filepath.Clean(ref)
	if cleaned != ref {
		variants = append(variants, cleaned)
	}

	// Convert to forward slashes (schemas registered with ToSlash)
	slashed := filepath.ToSlash(cleaned)
	if slashed != cleaned && slashed != ref {
		variants = append(variants, slashed)
	}

	// Try without "./" prefix if present
	if strings.HasPrefix(ref, "./") {
		withoutDot := strings.TrimPrefix(ref, "./")
		variants = append(variants, withoutDot)

		// Also try cleaned version without "./"
		cleanedWithoutDot := filepath.Clean(withoutDot)
		if cleanedWithoutDot != withoutDot {
			variants = append(variants, cleanedWithoutDot)
		}
	}

	// Try stripping "../" and "../../" prefixes
	stripped := ref
	for strings.HasPrefix(stripped, "../") {
		stripped = strings.TrimPrefix(stripped, "../")
	}
	if stripped != ref && stripped != "" {
		variants = append(variants, stripped)

		// Also try cleaned version of stripped
		cleanedStripped := filepath.Clean(stripped)
		if cleanedStripped != stripped {
			variants = append(variants, cleanedStripped)
		}
	}

	// Try adding "./" prefix if not present and not starting with "../"
	if !strings.HasPrefix(ref, "./") && !strings.HasPrefix(ref, "../") {
		variants = append(variants, "./"+ref)
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
