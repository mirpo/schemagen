package typegraph

import "fmt"

type typeRegistry struct {
	types   []*Type
	nameSet map[string]bool // O(1) name lookup for uniqueness checks
}

func newTypeRegistry() *typeRegistry {
	return &typeRegistry{
		types:   make([]*Type, 0),
		nameSet: make(map[string]bool),
	}
}

func (r *typeRegistry) add(t *Type) {
	r.types = append(r.types, t)
	r.nameSet[t.Name] = true
}

func (r *typeRegistry) all() []*Type {
	return r.types
}

func (r *typeRegistry) ensureUniqueName(baseName string) string {
	name := baseName
	counter := 1

	for r.nameSet[name] {
		counter++
		name = fmt.Sprintf("%s%d", baseName, counter)
	}
	return name
}
