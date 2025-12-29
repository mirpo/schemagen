package typegraph

import "fmt"

// TypeRegistry manages type storage, ID generation, and unique naming.
type TypeRegistry struct {
	types  []*Type
	nextID int
}

// NewTypeRegistry creates a new type registry.
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		types: make([]*Type, 0),
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
}

// All returns all registered types.
func (r *TypeRegistry) All() []*Type {
	return r.types
}

// EnsureUniqueName ensures a type name is unique by appending numbers if needed.
func (r *TypeRegistry) EnsureUniqueName(baseName string) string {
	name := baseName
	counter := 1

	for {
		exists := false
		for _, typ := range r.types {
			if typ.Name == name {
				exists = true
				break
			}
		}
		if !exists {
			return name
		}
		counter++
		name = fmt.Sprintf("%s%d", baseName, counter)
	}
}
