package graph

func TopoSort(g *Graph) []*Type {
	deps := make(map[string]map[string]bool)
	for _, t := range g.Types {
		deps[t.Name] = collectDeps(t)
	}

	var result []*Type
	visited := make(map[string]bool)
	visiting := make(map[string]bool)

	var visit func(name string)
	visit = func(name string) {
		if visited[name] || visiting[name] {
			return
		}
		visiting[name] = true

		for dep := range deps[name] {
			if g.GetType(dep) != nil {
				visit(dep)
			}
		}

		visiting[name] = false
		visited[name] = true
		if t := g.GetType(name); t != nil {
			result = append(result, t)
		}
	}

	for _, t := range g.Types {
		visit(t.Name)
	}

	return result
}

func collectDeps(t *Type) map[string]bool {
	deps := make(map[string]bool)

	for _, ext := range t.Extends {
		deps[ext] = true
	}

	for _, f := range t.Fields {
		collectRefDeps(f.Type, deps)
	}

	for _, m := range t.UnionMembers {
		collectRefDeps(m, deps)
	}

	if t.AdditionalProps != nil && t.AdditionalProps.Type != nil {
		collectRefDeps(t.AdditionalProps.Type, deps)
	}

	return deps
}

func collectRefDeps(ref *TypeRef, deps map[string]bool) {
	if ref == nil {
		return
	}
	if ref.Kind == KindRef && ref.TypeName != "" {
		deps[ref.TypeName] = true
	}
	collectRefDeps(ref.ItemType, deps)
	collectRefDeps(ref.ValueType, deps)
	for _, m := range ref.UnionMembers {
		collectRefDeps(m, deps)
	}
	for _, f := range ref.ObjectFields {
		collectRefDeps(f.Type, deps)
	}
}
