package typegraph

import "fmt"

// TypeRegistry manages type storage, ID generation, and unique naming.
type TypeRegistry struct {
	types   []*Type
	nameSet map[string]bool // O(1) name lookup for uniqueness checks
	nextID  int
}

// NewTypeRegistry creates a new type registry.
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		types:   make([]*Type, 0),
		nameSet: make(map[string]bool),
	}
}

// NextID generates the next unique type ID.
func (r *TypeRegistry) NextID() string {
	r.nextID++
	return fmt.Sprintf("type_%d", r.nextID)
}

// Add adds a type to the registry.
func (r *TypeRegistry) Add(t *Type) {
	r.types = append(r.types, t)
	r.nameSet[t.Name] = true
}

// All returns all registered types.
func (r *TypeRegistry) All() []*Type {
	return r.types
}

// EnsureUniqueName ensures a type name is unique by appending numbers if needed.
// Uses O(1) lookup via nameSet instead of O(n) linear scan.
func (r *TypeRegistry) EnsureUniqueName(baseName string) string {
	name := baseName
	counter := 1

	for r.nameSet[name] {
		counter++
		name = fmt.Sprintf("%s%d", baseName, counter)
	}
	return name
}
