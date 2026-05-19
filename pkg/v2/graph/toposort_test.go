package graph

import (
	"path/filepath"
	"testing"

	"github.com/mirpo/schemagen/pkg/v2/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopoSort_NoDeps(t *testing.T) {
	g := NewGraph()
	g.AddType(&Type{Name: "A", Kind: KindStruct})
	g.AddType(&Type{Name: "B", Kind: KindStruct})
	g.AddType(&Type{Name: "C", Kind: KindStruct})

	sorted := TopoSort(g)
	require.Len(t, sorted, 3)
	assert.Equal(t, "A", sorted[0].Name)
	assert.Equal(t, "B", sorted[1].Name)
	assert.Equal(t, "C", sorted[2].Name)
}

func TestTopoSort_SimpleDepBBeforeA(t *testing.T) {
	g := NewGraph()
	g.AddType(&Type{
		Name: "A",
		Kind: KindStruct,
		Fields: []*Field{
			{JSONName: "b", Type: &TypeRef{Kind: KindRef, TypeName: "B"}},
		},
	})
	g.AddType(&Type{Name: "B", Kind: KindStruct})

	sorted := TopoSort(g)
	require.Len(t, sorted, 2)
	assert.Equal(t, "B", sorted[0].Name)
	assert.Equal(t, "A", sorted[1].Name)
}

func TestTopoSort_DiamondDependency(t *testing.T) {
	// D depends on B, C
	// B depends on A
	// C depends on A
	g := NewGraph()
	g.AddType(&Type{
		Name: "D",
		Kind: KindStruct,
		Fields: []*Field{
			{JSONName: "b", Type: &TypeRef{Kind: KindRef, TypeName: "B"}},
			{JSONName: "c", Type: &TypeRef{Kind: KindRef, TypeName: "C"}},
		},
	})
	g.AddType(&Type{
		Name: "B",
		Kind: KindStruct,
		Fields: []*Field{
			{JSONName: "a", Type: &TypeRef{Kind: KindRef, TypeName: "A"}},
		},
	})
	g.AddType(&Type{
		Name: "C",
		Kind: KindStruct,
		Fields: []*Field{
			{JSONName: "a", Type: &TypeRef{Kind: KindRef, TypeName: "A"}},
		},
	})
	g.AddType(&Type{Name: "A", Kind: KindStruct})

	sorted := TopoSort(g)
	require.Len(t, sorted, 4)

	indexOf := make(map[string]int)
	for i, typ := range sorted {
		indexOf[typ.Name] = i
	}

	assert.Less(t, indexOf["A"], indexOf["B"])
	assert.Less(t, indexOf["A"], indexOf["C"])
	assert.Less(t, indexOf["B"], indexOf["D"])
	assert.Less(t, indexOf["C"], indexOf["D"])
}

func TestTopoSort_CyclicReference(t *testing.T) {
	g := NewGraph()
	g.AddType(&Type{
		Name: "A",
		Kind: KindStruct,
		Fields: []*Field{
			{JSONName: "b", Type: &TypeRef{Kind: KindRef, TypeName: "B"}},
		},
	})
	g.AddType(&Type{
		Name: "B",
		Kind: KindStruct,
		Fields: []*Field{
			{JSONName: "a", Type: &TypeRef{Kind: KindRef, TypeName: "A"}},
		},
	})

	sorted := TopoSort(g)
	require.Len(t, sorted, 2)

	names := make(map[string]bool)
	for _, typ := range sorted {
		names[typ.Name] = true
	}
	assert.True(t, names["A"])
	assert.True(t, names["B"])
}

func TestTopoSort_SelfReference(t *testing.T) {
	g := NewGraph()
	g.AddType(&Type{
		Name: "Node",
		Kind: KindStruct,
		Fields: []*Field{
			{JSONName: "parent", Type: &TypeRef{Kind: KindRef, TypeName: "Node"}},
		},
	})

	sorted := TopoSort(g)
	require.Len(t, sorted, 1)
	assert.Equal(t, "Node", sorted[0].Name)
}

func TestTopoSort_ExtendsDeps(t *testing.T) {
	g := NewGraph()
	g.AddType(&Type{
		Name:    "Child",
		Kind:    KindStruct,
		Extends: []string{"Parent"},
	})
	g.AddType(&Type{Name: "Parent", Kind: KindStruct})

	sorted := TopoSort(g)
	require.Len(t, sorted, 2)
	assert.Equal(t, "Parent", sorted[0].Name)
	assert.Equal(t, "Child", sorted[1].Name)
}

func TestTopoSort_UnionMemberDeps(t *testing.T) {
	g := NewGraph()
	g.AddType(&Type{
		Name: "Union",
		Kind: KindUnion,
		UnionMembers: []*TypeRef{
			{Kind: KindRef, TypeName: "TypeA"},
			{Kind: KindRef, TypeName: "TypeB"},
		},
	})
	g.AddType(&Type{Name: "TypeA", Kind: KindStruct})
	g.AddType(&Type{Name: "TypeB", Kind: KindStruct})

	sorted := TopoSort(g)
	require.Len(t, sorted, 3)

	indexOf := make(map[string]int)
	for i, typ := range sorted {
		indexOf[typ.Name] = i
	}
	assert.Less(t, indexOf["TypeA"], indexOf["Union"])
	assert.Less(t, indexOf["TypeB"], indexOf["Union"])
}

func TestTopoSort_ArrayItemDeps(t *testing.T) {
	g := NewGraph()
	g.AddType(&Type{
		Name: "Container",
		Kind: KindStruct,
		Fields: []*Field{
			{JSONName: "items", Type: &TypeRef{
				Kind:     KindArray,
				ItemType: &TypeRef{Kind: KindRef, TypeName: "Item"},
			}},
		},
	})
	g.AddType(&Type{Name: "Item", Kind: KindStruct})

	sorted := TopoSort(g)
	require.Len(t, sorted, 2)
	assert.Equal(t, "Item", sorted[0].Name)
	assert.Equal(t, "Container", sorted[1].Name)
}

func TestTopoSort_MapValueDeps(t *testing.T) {
	g := NewGraph()
	g.AddType(&Type{
		Name: "Registry",
		Kind: KindStruct,
		Fields: []*Field{
			{JSONName: "entries", Type: &TypeRef{
				Kind:      KindMap,
				ValueType: &TypeRef{Kind: KindRef, TypeName: "Entry"},
			}},
		},
	})
	g.AddType(&Type{Name: "Entry", Kind: KindStruct})

	sorted := TopoSort(g)
	require.Len(t, sorted, 2)
	assert.Equal(t, "Entry", sorted[0].Name)
	assert.Equal(t, "Registry", sorted[1].Name)
}

func TestTopoSort_ExternalDepIgnored(t *testing.T) {
	g := NewGraph()
	g.AddType(&Type{
		Name: "A",
		Kind: KindStruct,
		Fields: []*Field{
			{JSONName: "ext", Type: &TypeRef{Kind: KindRef, TypeName: "External"}},
		},
	})

	sorted := TopoSort(g)
	require.Len(t, sorted, 1)
	assert.Equal(t, "A", sorted[0].Name)
}

func TestTopoSort_EmptyGraph(t *testing.T) {
	g := NewGraph()
	sorted := TopoSort(g)
	assert.Empty(t, sorted)
}

func TestTopoSort_AdditionalPropsDeps(t *testing.T) {
	g := NewGraph()
	g.AddType(&Type{
		Name: "A",
		Kind: KindStruct,
		AdditionalProps: &AdditionalPropsConfig{
			Allowed: true,
			Type:    &TypeRef{Kind: KindRef, TypeName: "B"},
		},
	})
	g.AddType(&Type{Name: "B", Kind: KindStruct})

	sorted := TopoSort(g)
	require.Len(t, sorted, 2)

	indexOf := make(map[string]int)
	for i, typ := range sorted {
		indexOf[typ.Name] = i
	}
	assert.Less(t, indexOf["B"], indexOf["A"])
}

func TestTopoSort_InlineObjectFieldDeps(t *testing.T) {
	g := NewGraph()
	g.AddType(&Type{
		Name: "Parent",
		Kind: KindStruct,
		Fields: []*Field{
			{JSONName: "data", Type: &TypeRef{
				Kind: KindInterface,
				ObjectFields: []*Field{
					{JSONName: "nested", Type: &TypeRef{Kind: KindRef, TypeName: "Child"}},
				},
			}},
		},
	})
	g.AddType(&Type{Name: "Child", Kind: KindStruct})

	sorted := TopoSort(g)
	require.Len(t, sorted, 2)

	indexOf := make(map[string]int)
	for i, typ := range sorted {
		indexOf[typ.Name] = i
	}
	assert.Less(t, indexOf["Child"], indexOf["Parent"])
}

// ==================== collectDeps ====================

func TestCollectDeps(t *testing.T) {
	typ := &Type{
		Name:    "X",
		Kind:    KindStruct,
		Extends: []string{"Base"},
		Fields: []*Field{
			{JSONName: "ref", Type: &TypeRef{Kind: KindRef, TypeName: "Other"}},
			{JSONName: "arr", Type: &TypeRef{Kind: KindArray, ItemType: &TypeRef{Kind: KindRef, TypeName: "ArrayItem"}}},
			{JSONName: "prim", Type: &TypeRef{Kind: KindPrimitive, Primitive: PrimString}},
		},
		UnionMembers: []*TypeRef{
			{Kind: KindRef, TypeName: "UnionMember"},
		},
	}

	deps := collectDeps(typ)
	assert.True(t, deps["Base"])
	assert.True(t, deps["Other"])
	assert.True(t, deps["ArrayItem"])
	assert.True(t, deps["UnionMember"])
	assert.Len(t, deps, 4)
}

// ==================== Integration: EcommerceOrder toposort ====================

func TestTopoSort_Integration_EcommerceOrder(t *testing.T) {
	ns, err := parse.ParseFile(filepath.Join(testdataDir(), "schemas", "complex", "ecommerce_order.json"))
	require.NoError(t, err)

	g, err := Build([]*parse.NamedSchema{ns}, BuildConfig{})
	require.NoError(t, err)

	sorted := TopoSort(g)
	require.Len(t, sorted, 4)

	indexOf := make(map[string]int)
	for i, typ := range sorted {
		indexOf[typ.Name] = i
	}

	assert.Less(t, indexOf["Money"], indexOf["LineItem"])
	assert.Less(t, indexOf["Address"], indexOf["EcommerceOrder"])
	assert.Less(t, indexOf["LineItem"], indexOf["EcommerceOrder"])
	assert.Less(t, indexOf["Money"], indexOf["EcommerceOrder"])
}

func TestTopoSort_Integration_CyclicRef(t *testing.T) {
	ns, err := parse.ParseFile(filepath.Join(testdataDir(), "schemas", "edge-cases", "cyclic-ref.json"))
	require.NoError(t, err)

	g, err := Build([]*parse.NamedSchema{ns}, BuildConfig{})
	require.NoError(t, err)

	sorted := TopoSort(g)
	require.Len(t, sorted, 1)
	assert.Equal(t, "CyclicRef", sorted[0].Name)
}

func TestTopoSort_Integration_Document(t *testing.T) {
	ns, err := parse.ParseFile(filepath.Join(testdataDir(), "schemas", "allof", "document.json"))
	require.NoError(t, err)

	g, err := Build([]*parse.NamedSchema{ns}, BuildConfig{})
	require.NoError(t, err)

	sorted := TopoSort(g)
	require.Len(t, sorted, 3)

	indexOf := make(map[string]int)
	for i, typ := range sorted {
		indexOf[typ.Name] = i
	}

	assert.Less(t, indexOf["Entity"], indexOf["Document"])
	assert.Less(t, indexOf["Auditable"], indexOf["Document"])
}

func TestTopoSort_Integration_Organization(t *testing.T) {
	ns, err := parse.ParseFile(filepath.Join(testdataDir(), "schemas", "refs", "organization.json"))
	require.NoError(t, err)

	g, err := Build([]*parse.NamedSchema{ns}, BuildConfig{})
	require.NoError(t, err)

	sorted := TopoSort(g)
	require.Len(t, sorted, 4)

	indexOf := make(map[string]int)
	for i, typ := range sorted {
		indexOf[typ.Name] = i
	}

	assert.Less(t, indexOf["Country"], indexOf["Address"])
	assert.Less(t, indexOf["Address"], indexOf["Office"])
	assert.Less(t, indexOf["Office"], indexOf["Organization"])
}
