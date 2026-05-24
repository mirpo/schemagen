package parse

type SchemaNode struct {
	Schema      string
	ID          string
	Ref         string
	Title       string
	Description string

	Type   StringOrSlice
	Format string

	Properties           []NamedSchema
	Required             []string
	AdditionalProperties *AdditionalProperties

	Items       *SchemaNode
	PrefixItems []*SchemaNode

	AllOf []*SchemaNode
	AnyOf []*SchemaNode
	OneOf []*SchemaNode

	Enum     []any
	Const    any
	HasConst bool

	Defs []NamedSchema

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

type NamedSchema struct {
	Name   string
	Path   string
	Schema *SchemaNode
}

type AdditionalProperties struct {
	Allowed bool
	Schema  *SchemaNode
}

const (
	TypeString  = "string"
	TypeInteger = "integer"
	TypeNumber  = "number"
	TypeBoolean = "boolean"
	TypeObject  = "object"
	TypeArray   = "array"
	TypeNull    = "null"
)

type StringOrSlice []string

func (s StringOrSlice) Has(t string) bool {
	for _, v := range s {
		if v == t {
			return true
		}
	}
	return false
}

func (s StringOrSlice) Single() string {
	for _, v := range s {
		if v != TypeNull {
			return v
		}
	}
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

func (s StringOrSlice) IsNullable() bool {
	return s.Has(TypeNull) && len(s) > 1
}

func (n *SchemaNode) IsObject() bool { return n.Type.Has(TypeObject) || len(n.Properties) > 0 }
func (n *SchemaNode) IsArray() bool  { return n.Type.Has(TypeArray) }
func (n *SchemaNode) IsEnum() bool   { return len(n.Enum) > 0 }
func (n *SchemaNode) IsRef() bool    { return n.Ref != "" }
func (n *SchemaNode) IsAllOf() bool  { return len(n.AllOf) > 0 }
func (n *SchemaNode) IsAnyOf() bool  { return len(n.AnyOf) > 0 }
func (n *SchemaNode) IsOneOf() bool  { return len(n.OneOf) > 0 }
func (n *SchemaNode) IsUnion() bool  { return n.IsAnyOf() || n.IsOneOf() }
func (n *SchemaNode) IsConst() bool  { return n.HasConst }

func (n *SchemaNode) UnionMembers() []*SchemaNode {
	if len(n.AnyOf) > 0 {
		return n.AnyOf
	}
	return n.OneOf
}

func (n *SchemaNode) IsPrimitive() bool {
	t := n.Type.Single()
	return t == TypeString || t == TypeInteger || t == TypeNumber || t == TypeBoolean
}

func (n *SchemaNode) IsRequired(field string) bool {
	for _, r := range n.Required {
		if r == field {
			return true
		}
	}
	return false
}
