package typegraph

import "github.com/mirpo/schemagen/pkg/schema"

// PrimitiveKind identifies a language-neutral primitive type.
type PrimitiveKind int

const (
	PrimUnknown PrimitiveKind = iota
	PrimString
	PrimInt
	PrimInt32
	PrimInt64
	PrimFloat32
	PrimFloat64
	PrimBool
	PrimDateTime
	PrimDate
	PrimTime
	PrimUUID
	PrimEmail
	PrimURI
	PrimHostname
	PrimIPv4
	PrimIPv6
)

// TypeKind represents the kind of type.
type TypeKind string

const (
	KindStruct    TypeKind = "struct"
	KindEnum      TypeKind = "enum"
	KindPrimitive TypeKind = "primitive"
	KindArray     TypeKind = "array"
	KindMap       TypeKind = "map"
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
	EnumType   string // "string" or "int"
	EnumValues []EnumValue

	// For primitives
	Primitive PrimitiveKind

	// For arrays
	ItemType *TypeRef

	// For maps
	ValueType *TypeRef

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
	Kind      TypeKind      // Kind of referenced type
	TypeName  string        // Name of type (if named)
	Primitive PrimitiveKind // Language-neutral primitive kind
	Nullable  bool          // Is this nullable?
	Format    string        // JSON Schema format (email, uri, uuid, date-time, etc.)

	// For composite types
	ItemType     *TypeRef   // For arrays
	ValueType    *TypeRef   // For maps
	UnionMembers []*TypeRef // For unions (oneOf/anyOf)

	// For inline enums (not extracted as separate types)
	EnumValues []interface{} // Enum literal values

	// For inline objects (objects with properties not extracted as separate types)
	ObjectFields []*Field // Object fields for anonymous interfaces
}

// Walk traverses all nested TypeRefs, calling the visitor function for each.
// This enables generic traversal without duplicating recursion logic.
func (ref *TypeRef) Walk(visitor func(*TypeRef)) {
	if ref == nil {
		return
	}
	visitor(ref)
	ref.ItemType.Walk(visitor)
	ref.ValueType.Walk(visitor)
	for _, m := range ref.UnionMembers {
		m.Walk(visitor)
	}
}

// EnumValue represents an enum constant.
type EnumValue struct {
	Name  string      // Constant name (UPPER_SNAKE_CASE)
	Value interface{} // Actual value (string or int)
}

// ImportSpec represents an import statement in generated code.
type ImportSpec struct {
	ImportPath string   // The path/module to import from (language-specific format)
	TypeNames  []string // The specific types to import
	FromPath   string   // Source file path (for computing relative imports)
	ToPath     string   // Target file path (for computing relative imports)
}

// Graph holds all types in the schema.
type Graph struct {
	Types     []*Type          // All types
	typeIndex map[string]*Type // O(1) type lookup by name
}

// NewGraph creates a new Graph with initialized index.
func NewGraph() *Graph {
	return &Graph{
		Types:     make([]*Type, 0),
		typeIndex: make(map[string]*Type),
	}
}

// AddType adds a type to the graph and updates the index.
func (g *Graph) AddType(t *Type) {
	g.Types = append(g.Types, t)
	if g.typeIndex == nil {
		g.typeIndex = make(map[string]*Type)
	}
	g.typeIndex[t.Name] = t
}

// GetType finds a type by name using O(1) lookup.
func (g *Graph) GetType(name string) *Type {
	if g.typeIndex != nil {
		return g.typeIndex[name]
	}
	// Fallback for graphs created without index
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

// BuildContext carries per-schema state through the build pipeline.
type BuildContext struct {
	Order *schema.PropertyOrder
	Path  string
}
