package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeRefWalk_CyclicDoesNotStackOverflow(t *testing.T) {
	// Build a cyclic TypeRef: A -> B -> A
	a := &TypeRef{Kind: KindRef, TypeName: "A"}
	b := &TypeRef{Kind: KindRef, TypeName: "B"}
	a.ItemType = b
	b.ItemType = a

	var visited []string
	a.Walk(func(ref *TypeRef) {
		visited = append(visited, ref.TypeName)
	})

	assert.Contains(t, visited, "A")
	assert.Contains(t, visited, "B")
}

func TestTypeRefWalk_SelfReferenceDoesNotStackOverflow(t *testing.T) {
	self := &TypeRef{Kind: KindRef, TypeName: "Self"}
	self.ItemType = self

	var count int
	self.Walk(func(ref *TypeRef) {
		count++
	})

	assert.Equal(t, 1, count)
}

func TestTypeRefWalk_NormalTraversal(t *testing.T) {
	ref := &TypeRef{
		Kind:     KindArray,
		ItemType: &TypeRef{Kind: KindRef, TypeName: "Item"},
	}

	var visited []string
	ref.Walk(func(r *TypeRef) {
		visited = append(visited, r.TypeName)
	})

	assert.Equal(t, []string{"", "Item"}, visited)
}

func TestTypeRefWalk_Nil(t *testing.T) {
	var ref *TypeRef
	ref.Walk(func(r *TypeRef) {
		t.Fatal("should not be called")
	})
}
