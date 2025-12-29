package typegraph

// TypeKind represents the kind of type.
type TypeKind string

const (
	KindStruct    TypeKind = "struct"
	KindEnum      TypeKind = "enum"
	KindPrimitive TypeKind = "primitive"
	KindArray     TypeKind = "array"
	KindMap       TypeKind = "map"
	KindAlias     TypeKind = "alias"
	KindRef       TypeKind = "ref"       // Reference to another type
	KindUnion     TypeKind = "union"     // Union type (oneOf/anyOf)
	KindInterface TypeKind = "interface" // For empty objects or any
)

// Type represents a type in the type graph.
type Type struct {
	ID          string   // Unique identifier
	Name        string   // Type name (PascalCase)
	Kind        TypeKind // Type kind
	Description string   // From schema description

	// For structs
	Fields          []*Field
	Extends         []string               // For allOf composition (list of base type names)
	AdditionalProps *AdditionalPropsConfig // Additional properties configuration

	// For enums
	EnumType         string // "string" or "int"
	EnumValues       []EnumValue
	HasComplexValues bool // true if enum contains non-primitive values (objects/arrays)

	// For primitives
	GoType string // e.g., "string", "int", "float64"

	// For arrays
	ItemType *TypeRef

	// For maps
	ValueType *TypeRef

	// For aliases
	TargetType *TypeRef

	// For unions
	UnionMembers []*TypeRef // Union members (anyOf/oneOf)
}

// AdditionalPropsConfig captures additionalProperties from JSON Schema.
type AdditionalPropsConfig struct {
	Allowed bool     // true if additionalProperties is true or an object
	Type    *TypeRef // Type of additional properties (nil if any type allowed)
}

// Field represents a struct field.
type Field struct {
	Name        string   // Field name (PascalCase)
	JSONName    string   // JSON tag name
	Type        *TypeRef // Field type
	Description string   // From schema
	Required    bool     // Is this field required?
	OmitEmpty   bool     // Should we add omitempty?

	// Validation constraints
	MinLength        *int     // minLength for strings
	MaxLength        *int     // maxLength for strings
	Pattern          *string  // pattern for strings
	Minimum          *float64 // minimum for numbers
	Maximum          *float64 // maximum for numbers
	ExclusiveMinimum *float64 // exclusiveMinimum for numbers
	ExclusiveMaximum *float64 // exclusiveMaximum for numbers
	MinItems         *int     // minItems for arrays
	MaxItems         *int     // maxItems for arrays
}

// TypeRef is a reference to another type.
type TypeRef struct {
	Kind     TypeKind // Kind of referenced type
	TypeName string   // Name of type (if named)
	GoType   string   // Direct Go type (if primitive)
	Nullable bool     // Is this nullable?
	Format   string   // JSON Schema format (email, uri, uuid, date-time, etc.)

	// For composite types
	ItemType     *TypeRef   // For arrays
	ValueType    *TypeRef   // For maps
	UnionMembers []*TypeRef // For unions (oneOf/anyOf)

	// For inline enums (not extracted as separate types)
	EnumValues []interface{} // Enum literal values

	// For inline objects (objects with properties not extracted as separate types)
	ObjectFields []*Field // Object fields for anonymous interfaces
}

// EnumValue represents an enum constant.
type EnumValue struct {
	Name  string      // Constant name (UPPER_SNAKE_CASE)
	Value interface{} // Actual value (string or int)
}

// ImportSpec represents an import statement in generated code.
type ImportSpec struct {
	ImportPath string   // The path/module to import from
	TypeNames  []string // The specific types to import
}

// Graph holds all types in the schema.
type Graph struct {
	Types []*Type // All types
}

// GetType finds a type by name.
func (g *Graph) GetType(name string) *Type {
	for _, t := range g.Types {
		if t.Name == name {
			return t
		}
	}
	return nil
}

// BuildConfig controls type graph building behavior.
type BuildConfig struct {
	ExtractInlined bool // Extract inline enums/objects to named types
}
