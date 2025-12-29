package typegraph

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kaptinlin/jsonschema"
	"github.com/mirpo/schemagen/pkg/naming"
	"github.com/mirpo/schemagen/pkg/schema"
)

// Builder builds a type graph from JSON schemas.
type Builder struct {
	types         []*Type
	nextID        int
	compiler      *jsonschema.Compiler  // For resolving $refs
	currentSchema *jsonschema.Schema    // Current root schema being processed
	currentOrder  *schema.PropertyOrder // Current schema's property order
	currentPath   string                // Current schema path for order lookups
	config        *BuildConfig          // Build configuration
}

// NewBuilder creates a new type graph builder with default configuration.
func NewBuilder(compiler *jsonschema.Compiler) *Builder {
	return NewBuilderWithConfig(compiler, nil)
}

// NewBuilderWithConfig creates a new type graph builder with custom configuration.
func NewBuilderWithConfig(compiler *jsonschema.Compiler, cfg *BuildConfig) *Builder {
	if cfg == nil {
		cfg = &BuildConfig{}
	}
	return &Builder{
		types:    make([]*Type, 0),
		compiler: compiler,
		config:   cfg,
	}
}

// Build builds a type graph from loaded schemas.
func (b *Builder) Build(schemas []*schema.Schema) (*Graph, error) {
	for _, s := range schemas {
		if err := b.processSchema(s); err != nil {
			return nil, fmt.Errorf("processing %s: %w", s.Path, err)
		}
	}

	return &Graph{
		Types: b.types,
	}, nil
}

// processSchema processes a single schema and extracts types.
func (b *Builder) processSchema(s *schema.Schema) error {
	// Store current schema's property order for lookups
	b.currentOrder = s.PropertyOrder
	b.currentPath = s.RelativePath

	// Get the compiled schema
	compiled := s.Compiled
	// Store current root schema for self-reference resolution
	b.currentSchema = compiled

	// First, extract $defs as separate types
	if compiled.Defs != nil {
		// Sort $defs keys for deterministic iteration
		defNames := make([]string, 0, len(compiled.Defs))
		for defName := range compiled.Defs {
			defNames = append(defNames, defName)
		}
		sort.Strings(defNames)

		// Process each $def with correct path for order lookup
		for _, defName := range defNames {
			defSchema := compiled.Defs[defName]

			// Set currentPath to the $def path for property order lookup
			// Format: "tricky.json#/$defs/Node"
			previousPath := b.currentPath
			b.currentPath = fmt.Sprintf("%s#/$defs/%s", s.RelativePath, defName)

			if err := b.extractDefinition(defName, defSchema); err != nil {
				return fmt.Errorf("extracting $def %s: %w", defName, err)
			}

			// Restore path
			b.currentPath = previousPath
		}
	}

	// Determine type kind based on schema
	typ := &Type{
		ID:          b.nextTypeID(),
		Name:        s.Name,
		Description: getDescription(compiled),
	}

	// Handle different schema types
	if isObject(compiled) {
		if err := b.buildStruct(typ, compiled); err != nil {
			return err
		}
	} else if isEnum(compiled) {
		if err := b.buildEnum(typ, compiled); err != nil {
			return err
		}
	} else if isUnion(compiled) {
		if err := b.buildUnion(typ, compiled); err != nil {
			return err
		}
	} else {
		// Primitive or other type
		typ.Kind = KindPrimitive
		typ.GoType = b.mapPrimitiveType(compiled)
	}

	b.types = append(b.types, typ)
	return nil
}

// extractDefinition extracts a $def as a separate type.
func (b *Builder) extractDefinition(name string, schema *jsonschema.Schema) error {
	typ := &Type{
		ID:          b.nextTypeID(),
		Name:        naming.ToPascalCase(name),
		Description: getDescription(schema),
	}

	// Handle different definition types
	if isObject(schema) {
		if err := b.buildStruct(typ, schema); err != nil {
			return err
		}
	} else if isEnum(schema) {
		if err := b.buildEnum(typ, schema); err != nil {
			return err
		}
	} else {
		// Primitive or other type
		typ.Kind = KindPrimitive
		typ.GoType = b.mapPrimitiveType(schema)
	}

	b.types = append(b.types, typ)
	return nil
}

// mapPrimitiveType maps a JSON schema type to a Go type.
func (b *Builder) mapPrimitiveType(schema *jsonschema.Schema) string {
	// Check for format-specific mappings first
	if schema.Format != nil {
		switch *schema.Format {
		case "uuid":
			return "uuid.UUID"
		case "date-time":
			return "time.Time"
		case "date":
			return "time.Time"
		case "time":
			return "string" // No standard Go type for time-only
		case "email", "uri", "hostname", "ipv4", "ipv6":
			return "string" // Validated strings
		}
	}

	// Check type
	if len(schema.Type) > 0 {
		switch schema.Type[0] {
		case "string":
			return "string"
		case "integer":
			// Check for specific integer formats
			if schema.Format != nil {
				switch *schema.Format {
				case "int32":
					return "int32"
				case "int64":
					return "int64"
				}
			}
			return "int"
		case "number":
			// Check for specific number formats
			if schema.Format != nil {
				switch *schema.Format {
				case "float":
					return "float32"
				case "double":
					return "float64"
				}
			}
			return "float64"
		case "boolean":
			return "bool"
		case "array":
			// Should not reach here (handled separately)
			return "[]interface{}"
		case "object":
			// Should not reach here (handled as struct)
			return "map[string]interface{}"
		}
	}

	return "interface{}"
}

// Helper functions

func (b *Builder) nextTypeID() string {
	b.nextID++
	return fmt.Sprintf("type_%d", b.nextID)
}

func getDescription(schema *jsonschema.Schema) string {
	if schema.Description != nil {
		return *schema.Description
	}
	return ""
}

func isObject(schema *jsonschema.Schema) bool {
	// Check if schema defines an object with properties
	if schema.Properties != nil && len(*schema.Properties) > 0 {
		return true
	}
	// Check if schema uses allOf composition (even without direct properties)
	if len(schema.AllOf) > 0 {
		return true
	}
	// Check if schema defines property name constraints (even without direct properties)
	if schema.PropertyNames != nil {
		return true
	}
	return false
}

func isEnum(schema *jsonschema.Schema) bool {
	// Check if schema defines enum values
	return len(schema.Enum) > 0
}

func isUnion(schema *jsonschema.Schema) bool {
	return len(schema.AnyOf) > 0 || len(schema.OneOf) > 0
}

// buildTypeRef builds a TypeRef from a schema property.
// fieldName is used for naming extracted inline types (can be empty).
func (b *Builder) buildTypeRef(schema *jsonschema.Schema, fieldName string) *TypeRef {
	ref := &TypeRef{
		Nullable: false,
	}

	// Check for nullable types (type array containing "null")
	if len(schema.Type) > 0 {
		for _, t := range schema.Type {
			if t == "null" {
				ref.Nullable = true
				// Continue processing other types in the array
				// The nullable flag will be used by language generators
			}
		}
	}

	// Check for $ref (external or internal reference)
	if schema.Ref != "" {
		refURI := schema.Ref

		// Extract type name from $ref (handles both #/$defs/Type and external refs)
		typeName := b.extractTypeNameFromRef(refURI)
		if typeName != "" {
			ref.Kind = KindRef
			ref.TypeName = typeName
			return ref
		}

		// If we couldn't extract a type name, fall back to interface{}
		ref.Kind = KindInterface
		ref.GoType = "interface{}" // Unresolved ref
		return ref
	}

	// Check for const (single-value enum)
	if schema.Const != nil && schema.Const.IsSet {
		// Const is like a single-value enum
		constValue := schema.Const.Value
		ref.Kind = KindEnum
		ref.EnumValues = []interface{}{constValue}
		// Determine base type from const value
		switch constValue.(type) {
		case string:
			ref.GoType = "string"
		case float64, int, int64:
			ref.GoType = "int"
		default:
			ref.GoType = "interface{}"
		}
		return ref
	}

	// Check for inline enum (before oneOf/anyOf since enum takes precedence)
	if len(schema.Enum) > 0 {
		// Check if we should extract inline enums
		if b.config.ExtractInlined && fieldName != "" {
			// Extract as separate named type
			enumTypeName := naming.ToPascalCase(fieldName)
			// Check if enum type already exists, otherwise create unique name
			enumTypeName = b.ensureUniqueTypeName(enumTypeName)

			enumType := &Type{
				ID:          b.nextTypeID(),
				Name:        enumTypeName,
				Kind:        KindEnum,
				Description: getDescription(schema),
				EnumValues:  make([]EnumValue, 0),
			}

			// Determine enum type from first value
			switch schema.Enum[0].(type) {
			case string:
				enumType.EnumType = "string"
			case float64, int, int64:
				enumType.EnumType = "int"
			default:
				enumType.EnumType = "string"
			}

			// Extract enum values
			for _, val := range schema.Enum {
				enumVal := EnumValue{
					Name:  naming.ToConstantCase(fmt.Sprintf("%v", val)),
					Value: val,
				}
				enumType.EnumValues = append(enumType.EnumValues, enumVal)
			}

			b.types = append(b.types, enumType)

			// Return reference to extracted type
			ref.Kind = KindRef
			ref.TypeName = enumTypeName
			return ref
		}

		// Otherwise, keep inline
		// Inline enum - not extracted as separate type
		// Will be rendered as Literal in Python, literal union in TypeScript
		ref.Kind = KindEnum
		ref.EnumValues = schema.Enum
		// Determine base type from first value
		if len(schema.Enum) > 0 {
			switch schema.Enum[0].(type) {
			case string:
				ref.GoType = "string"
			case float64, int, int64:
				ref.GoType = "int"
			default:
				ref.GoType = "interface{}"
			}
		}
		return ref
	}

	// Check for oneOf/anyOf (union types)
	if len(schema.OneOf) > 0 {
		ref.Kind = KindUnion
		ref.UnionMembers = make([]*TypeRef, 0, len(schema.OneOf))
		for i, memberSchema := range schema.OneOf {
			// Check if this is an inline object that should be extracted
			if b.shouldExtractInlineObject(memberSchema) {
				// Extract as separate type with field-based name
				var typeName string
				if fieldName != "" {
					baseName := naming.ToPascalCase(fieldName)
					if i == 0 {
						typeName = baseName // First variant: Payload
					} else {
						typeName = fmt.Sprintf("%s%d", baseName, i) // Subsequent: Payload1, Payload2
					}
				} else {
					typeName = fmt.Sprintf("Variant%d", i) // Fallback
				}
				extractedType := b.extractInlineObjectType(typeName, memberSchema)
				// Add to types list
				b.types = append(b.types, extractedType)
				// Return reference to extracted type
				memberRef := &TypeRef{
					Kind:     KindRef,
					TypeName: extractedType.Name,
				}
				ref.UnionMembers = append(ref.UnionMembers, memberRef)
			} else {
				memberRef := b.buildTypeRef(memberSchema, "")
				ref.UnionMembers = append(ref.UnionMembers, memberRef)
			}
		}
		return ref
	}

	if len(schema.AnyOf) > 0 {
		ref.Kind = KindUnion
		ref.UnionMembers = make([]*TypeRef, 0, len(schema.AnyOf))
		for i, memberSchema := range schema.AnyOf {
			// Check if this is an inline object that should be extracted
			if b.shouldExtractInlineObject(memberSchema) {
				// Extract as separate type with field-based name
				var typeName string
				if fieldName != "" {
					baseName := naming.ToPascalCase(fieldName)
					if i == 0 {
						typeName = baseName // First variant: Payload
					} else {
						typeName = fmt.Sprintf("%s%d", baseName, i) // Subsequent: Payload1, Payload2
					}
				} else {
					typeName = fmt.Sprintf("Variant%d", i) // Fallback
				}
				extractedType := b.extractInlineObjectType(typeName, memberSchema)
				// Add to types list
				b.types = append(b.types, extractedType)
				// Return reference to extracted type
				memberRef := &TypeRef{
					Kind:     KindRef,
					TypeName: extractedType.Name,
				}
				ref.UnionMembers = append(ref.UnionMembers, memberRef)
			} else {
				memberRef := b.buildTypeRef(memberSchema, "")
				ref.UnionMembers = append(ref.UnionMembers, memberRef)
			}
		}
		return ref
	}

	// Check for arrays
	if len(schema.Type) > 0 && schema.Type[0] == "array" {
		ref.Kind = KindArray
		if schema.Items != nil {
			// Pass field name context for array items to enable extraction
			itemFieldName := ""
			if fieldName != "" {
				itemFieldName = fieldName + "Item"
			}
			ref.ItemType = b.buildTypeRef(schema.Items, itemFieldName)
		} else {
			ref.ItemType = &TypeRef{Kind: KindPrimitive, GoType: "interface{}"}
		}
		return ref
	}

	// Check for objects
	if len(schema.Type) > 0 && schema.Type[0] == "object" {
		hasProps := schema.Properties != nil && len(*schema.Properties) > 0
		if !hasProps {
			// Object without properties - treat as map
			ref.Kind = KindMap

			// Check if additionalProperties specifies a type
			if schema.AdditionalProperties != nil && schema.AdditionalProperties.Boolean == nil {
				// Has typed additionalProperties (e.g., {type: "string"})
				ref.ValueType = b.buildTypeRef(schema.AdditionalProperties, "additionalProperty")
			} else {
				// No additionalProperties or just true/false - use any type
				ref.ValueType = &TypeRef{Kind: KindPrimitive, GoType: "interface{}"}
			}
			return ref
		}

		// Object with properties
		if b.config.ExtractInlined && fieldName != "" {
			// Extract as separate named type
			objectTypeName := naming.ToPascalCase(fieldName)
			objectTypeName = b.ensureUniqueTypeName(objectTypeName)

			extractedType := b.extractInlineObjectType(objectTypeName, schema)
			b.types = append(b.types, extractedType)

			// Return reference to extracted type
			ref.Kind = KindRef
			ref.TypeName = extractedType.Name
			return ref
		}

		// Otherwise, keep inline - create inline object with fields
		ref.Kind = KindInterface
		ref.GoType = "map[string]interface{}"

		// Extract fields for the inline object
		ref.ObjectFields = b.buildFieldsFromProperties(schema, "")
		return ref
	}

	// Primitive type
	ref.Kind = KindPrimitive
	ref.GoType = b.mapPrimitiveType(schema)

	// Extract format if present (email, uri, uuid, date-time, etc.)
	if schema.Format != nil && *schema.Format != "" {
		ref.Format = *schema.Format
	}

	return ref
}

// deriveTypeName derives a type name from a referenced schema.
func (b *Builder) deriveTypeName(refSchema *jsonschema.Schema, refURI string) string {
	// Try to get the title from the schema - use as-is if already PascalCase
	if refSchema.Title != nil && *refSchema.Title != "" {
		title := *refSchema.Title
		// If title is already in PascalCase, use it directly
		if naming.IsPascalCase(title) {
			return title
		}
		return naming.ToPascalCase(title)
	}

	// Fall back to deriving from the URI
	// For "header.json" -> "Header"
	// For "payloads/subscribe.json" -> "Subscribe"
	if refURI != "" {
		// Extract filename without extension
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
//
// Examples:
//
//	"./settings.yaml" -> ["./settings.yaml", "settings.yaml"]
//	"../events/config.json" -> ["../events/config.json", "events/config.json"]
//	"events/part1.json" -> ["events/part1.json", "./events/part1.json"]
//	"./nested/../simple.json" -> ["./nested/../simple.json", "simple.json", "nested/../simple.json"]
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
	// "../events/config.json" -> "events/config.json"
	// "../../shared/types.json" -> "shared/types.json"
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
//
// Examples:
//
//	"./settings.yaml" -> "Settings"
//	"nested/config.json" -> "Config"
//	"./user-settings.yaml" -> "UserSettings"
//	"../events/config.json" -> "Config"
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

// extractTypeNameFromRef extracts a type name from a $ref string.
// Examples:
//
//	"#/$defs/IdObject" -> "IdObject"
//	"header.json" -> "EventHeader" (requires resolution)
//	"#/$defs/Person" -> "Person"
func (b *Builder) extractTypeNameFromRef(ref string) string {
	// Handle root self-reference "#"
	if ref == "#" {
		if b.currentSchema != nil {
			// Use the root schema's title if available
			if b.currentSchema.Title != nil && *b.currentSchema.Title != "" {
				return *b.currentSchema.Title
			}
		}
		// Default to "Schema" if no title is set
		return "Schema"
	}

	// Handle internal $defs references
	// "#/$defs/IdObject" -> "IdObject"
	if strings.HasPrefix(ref, "#/$defs/") {
		defName := strings.TrimPrefix(ref, "#/$defs/")
		return naming.ToPascalCase(defName)
	}

	// Handle external file references - try multiple normalized variants
	for _, variant := range normalizeRefPath(ref) {
		if refSchema, err := b.compiler.GetSchema(variant); err == nil && refSchema != nil {
			return b.deriveTypeName(refSchema, ref)
		}
	}

	// Fall back to extracting type name from filename
	return extractTypeNameFromFilename(ref)
}

// getOrderedPropertyNames returns property names in schema file order.
// Falls back to alphabetical sorting if order information is not available.
func (b *Builder) getOrderedPropertyNames(properties *jsonschema.SchemaMap, schemaPath string) []string {
	if properties == nil {
		return nil
	}

	// Try to get order from extracted property order
	if b.currentOrder != nil {
		if ordered := b.currentOrder.GetOrder(schemaPath); len(ordered) > 0 {
			// Filter to only include keys that exist in the properties map
			// (defensive programming - should always match)
			mapKeys := make(map[string]bool)
			for key := range *properties {
				mapKeys[key] = true
			}

			result := make([]string, 0, len(ordered))
			for _, key := range ordered {
				if mapKeys[key] {
					result = append(result, key)
					delete(mapKeys, key)
				}
			}

			// Add any keys not in order (shouldn't happen, but be safe)
			if len(mapKeys) > 0 {
				extra := make([]string, 0, len(mapKeys))
				for key := range mapKeys {
					extra = append(extra, key)
				}
				sort.Strings(extra) // Sort extras for determinism
				result = append(result, extra...)
			}

			return result
		}
	}

	// Fallback to alphabetical sorting for backward compatibility
	names := make([]string, 0, len(*properties))
	for name := range *properties {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ensureUniqueTypeName ensures a type name is unique by appending numbers if needed.
func (b *Builder) ensureUniqueTypeName(baseName string) string {
	name := baseName
	counter := 1

	// Check if name already exists
	for {
		exists := false
		for _, typ := range b.types {
			if typ.Name == name {
				exists = true
				break
			}
		}
		if !exists {
			return name
		}
		// Try next name
		counter++
		name = fmt.Sprintf("%s%d", baseName, counter)
	}
}
