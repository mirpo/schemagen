package typegraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeRegistry_NextID(t *testing.T) {
	r := NewTypeRegistry()

	assert.Equal(t, "type_1", r.NextID())
	assert.Equal(t, "type_2", r.NextID())
	assert.Equal(t, "type_3", r.NextID())
}

func TestTypeRegistry_Add(t *testing.T) {
	r := NewTypeRegistry()

	typ1 := &Type{ID: "type_1", Name: "User"}
	typ2 := &Type{ID: "type_2", Name: "Order"}

	r.Add(typ1)
	r.Add(typ2)

	all := r.All()
	assert.Len(t, all, 2)
	assert.Equal(t, typ1, all[0])
	assert.Equal(t, typ2, all[1])
}

func TestTypeRegistry_All_Empty(t *testing.T) {
	r := NewTypeRegistry()

	all := r.All()
	assert.NotNil(t, all)
	assert.Len(t, all, 0)
}

func TestTypeRegistry_EnsureUniqueName_NoConflict(t *testing.T) {
	r := NewTypeRegistry()

	name := r.EnsureUniqueName("User")
	assert.Equal(t, "User", name)
}

func TestTypeRegistry_EnsureUniqueName_WithConflict(t *testing.T) {
	r := NewTypeRegistry()

	r.Add(&Type{ID: "type_1", Name: "User"})

	name := r.EnsureUniqueName("User")
	assert.Equal(t, "User2", name)
}

func TestTypeRegistry_EnsureUniqueName_MultipleConflicts(t *testing.T) {
	r := NewTypeRegistry()

	r.Add(&Type{ID: "type_1", Name: "User"})
	r.Add(&Type{ID: "type_2", Name: "User2"})
	r.Add(&Type{ID: "type_3", Name: "User3"})

	name := r.EnsureUniqueName("User")
	assert.Equal(t, "User4", name)
}

func TestTypeRegistry_EnsureUniqueName_DifferentBase(t *testing.T) {
	r := NewTypeRegistry()

	r.Add(&Type{ID: "type_1", Name: "User"})

	name := r.EnsureUniqueName("Order")
	assert.Equal(t, "Order", name)
}
