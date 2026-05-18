package graph

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

type TypeKind string

const (
	KindStruct    TypeKind = "struct"
	KindEnum      TypeKind = "enum"
	KindPrimitive TypeKind = "primitive"
	KindArray     TypeKind = "array"
	KindMap       TypeKind = "map"
	KindRef       TypeKind = "ref"
	KindUnion     TypeKind = "union"
	KindInterface TypeKind = "interface"
)

type EnumKind string

const (
	EnumKindString EnumKind = "string"
	EnumKindInt    EnumKind = "int"
	EnumKindMixed  EnumKind = "mixed"
)

type Type struct {
	Name        string
	Kind        TypeKind
	Description string

	Fields          []*Field
	Extends         []string
	AdditionalProps *AdditionalPropsConfig

	EnumType   EnumKind
	EnumValues []EnumValue

	Primitive PrimitiveKind

	UnionMembers []*TypeRef
}

type AdditionalPropsConfig struct {
	Allowed bool
	Type    *TypeRef
}

type Field struct {
	JSONName    string
	Type        *TypeRef
	Description string
	Required    bool

	MinLength        *int
	MaxLength        *int
	Pattern          *string
	Minimum          *float64
	Maximum          *float64
	ExclusiveMinimum *float64
	ExclusiveMaximum *float64
	MinItems         *int
	MaxItems         *int
}

func (f *Field) HasConstraints() bool {
	return f.MinLength != nil || f.MaxLength != nil ||
		f.Pattern != nil ||
		f.Minimum != nil || f.Maximum != nil ||
		f.ExclusiveMinimum != nil || f.ExclusiveMaximum != nil ||
		f.MinItems != nil || f.MaxItems != nil
}

type TypeRef struct {
	Kind      TypeKind
	TypeName  string
	Primitive PrimitiveKind
	Nullable  bool
	Format    string

	ItemType     *TypeRef
	ValueType    *TypeRef
	UnionMembers []*TypeRef

	EnumValues []any

	ObjectFields []*Field
}

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
	for _, f := range ref.ObjectFields {
		f.Type.Walk(visitor)
	}
}

type EnumValue struct {
	Name  string
	Value any
}

type ImportSpec struct {
	ImportPath string
	TypeNames  []string
	FromPath   string
	ToPath     string
}

type Graph struct {
	Types     []*Type
	Warnings  []string
	typeIndex map[string]*Type
}

func NewGraph() *Graph {
	return &Graph{
		Types:     make([]*Type, 0),
		typeIndex: make(map[string]*Type),
	}
}

func (g *Graph) AddType(t *Type) {
	if _, exists := g.typeIndex[t.Name]; exists {
		return
	}
	g.Types = append(g.Types, t)
	g.typeIndex[t.Name] = t
}

func (g *Graph) GetType(name string) *Type {
	if g.typeIndex != nil {
		return g.typeIndex[name]
	}
	for _, t := range g.Types {
		if t.Name == name {
			return t
		}
	}
	return nil
}

type BuildConfig struct {
	ExtractInlined bool
}
