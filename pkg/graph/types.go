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

type Type struct {
	Name        string
	Kind        TypeKind
	Description string
	SourceFile  string

	Fields          []*Field
	Extends         []string
	AdditionalProps *AdditionalPropsConfig

	EnumValues []EnumValue

	Primitive PrimitiveKind

	UnionMembers []*TypeRef
}

type AdditionalPropsConfig struct {
	Allowed bool
	Type    *TypeRef
}

type Constraints struct {
	MinLength        *int
	MaxLength        *int
	Pattern          *string
	Minimum          *float64
	Maximum          *float64
	ExclusiveMinimum *float64
	ExclusiveMaximum *float64
	MinItems         *int
	MaxItems         *int
	MultipleOf       *float64
}

func (c *Constraints) HasConstraints() bool {
	return c.MinLength != nil || c.MaxLength != nil ||
		c.Pattern != nil ||
		c.Minimum != nil || c.Maximum != nil ||
		c.ExclusiveMinimum != nil || c.ExclusiveMaximum != nil ||
		c.MinItems != nil || c.MaxItems != nil ||
		c.MultipleOf != nil
}

type Field struct {
	JSONName    string
	Type        *TypeRef
	Description string
	Required    bool
	Constraints
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

	EnumValues []EnumValue

	ObjectFields []*Field
}

func (ref *TypeRef) Walk(visitor func(*TypeRef)) {
	visited := make(map[*TypeRef]struct{})
	ref.walk(visitor, visited)
}

func (ref *TypeRef) walk(visitor func(*TypeRef), visited map[*TypeRef]struct{}) {
	if ref == nil {
		return
	}
	if _, seen := visited[ref]; seen {
		return
	}
	visited[ref] = struct{}{}
	visitor(ref)
	ref.ItemType.walk(visitor, visited)
	ref.ValueType.walk(visitor, visited)
	for _, m := range ref.UnionMembers {
		m.walk(visitor, visited)
	}
	for _, f := range ref.ObjectFields {
		f.Type.walk(visitor, visited)
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
