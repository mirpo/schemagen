package graph

import (
	"fmt"
	"path"
	"strings"

	"github.com/mirpo/schemagen/pkg/parse"
)

func Build(schemas []*parse.NamedSchema, cfg BuildConfig) (*Graph, error) {
	g := NewGraph()
	b := &builder{
		graph:          g,
		cfg:            cfg,
		globalDefs:     make(map[string]*parse.SchemaNode),
		processedTypes: make(map[string]bool),
		schemaNames:    make(map[string]string),
	}

	for _, ns := range schemas {
		for _, d := range ns.Schema.Defs {
			b.globalDefs[d.Name] = d.Schema
		}
		b.registerSchemaName(ns)
	}

	for _, ns := range schemas {
		b.currentSourceFile = ns.Path
		if b.currentSourceFile == "" {
			b.currentSourceFile = ns.Name
		}

		for _, d := range ns.Schema.Defs {
			b.buildType(ToPascalCase(d.Name), d.Schema, ns.Name)
		}

		rootName := ToPascalCase(ns.Name)
		b.buildType(rootName, ns.Schema, ns.Name)
	}

	g.Warnings = b.warnings
	return g, nil
}

type builder struct {
	graph             *Graph
	cfg               BuildConfig
	globalDefs        map[string]*parse.SchemaNode
	processedTypes    map[string]bool
	schemaNames       map[string]string
	warnings          []string
	currentSourceFile string
}

func (b *builder) warn(format string, args ...any) {
	b.warnings = append(b.warnings, fmt.Sprintf(format, args...))
}

func (b *builder) registerSchemaName(ns *parse.NamedSchema) {
	name := ToPascalCase(ns.Name)
	path := ns.Path
	b.schemaNames[path] = name

	for _, variant := range refPathVariants(path) {
		b.schemaNames[variant] = name
	}
}

func refPathVariants(p string) []string {
	var variants []string

	base := path.Base(p)
	if base != p {
		variants = append(variants, base)
	}

	cleaned := path.Clean(p)
	if cleaned != p {
		variants = append(variants, cleaned)
	}

	if strings.HasPrefix(p, "./") {
		variants = append(variants, strings.TrimPrefix(p, "./"))
	} else {
		variants = append(variants, "./"+p)
	}

	return variants
}

func (b *builder) buildType(name string, node *parse.SchemaNode, rootName string) {
	if b.processedTypes[name] {
		return
	}
	b.processedTypes[name] = true

	switch {
	case node.IsAllOf():
		b.buildAllOf(name, node, rootName)
	case node.IsEnum() && node.IsObject():
		b.buildStruct(name, node, rootName, false)
	case node.IsEnum():
		b.buildEnum(name, node)
	case node.IsUnion():
		b.buildUnion(name, node, rootName)
	case node.IsObject():
		b.buildStruct(name, node, rootName, false)
	case node.IsArray():
		b.graph.AddType(&Type{
			Name:       name,
			Kind:       KindPrimitive,
			Primitive:  PrimUnknown,
			SourceFile: b.currentSourceFile,
		})
	case node.IsPrimitive():
		b.graph.AddType(&Type{
			Name:       name,
			Kind:       KindPrimitive,
			Primitive:  mapPrimitive(node.Type.Single(), node.Format),
			SourceFile: b.currentSourceFile,
		})
	default:
		b.warn("schema %q did not produce a type (no recognizable structure)", name)
	}
}

func (b *builder) buildStruct(name string, node *parse.SchemaNode, rootName string, sortProps bool) {
	t := &Type{
		Name:        name,
		Kind:        KindStruct,
		Description: node.Description,
		SourceFile:  b.currentSourceFile,
	}

	if node.AdditionalProperties != nil {
		t.AdditionalProps = &AdditionalPropsConfig{
			Allowed: node.AdditionalProperties.Allowed,
		}
		if node.AdditionalProperties.Schema != nil {
			ref := b.buildTypeRef(name, "", node.AdditionalProperties.Schema, rootName)
			t.AdditionalProps.Type = ref
		}
	}

	props := node.Properties
	if sortProps {
		props = sortedProperties(props)
	}

	t.Fields = append(t.Fields, b.buildFields(name, props, rootName, node.IsRequired)...)

	b.graph.AddType(t)
}

func (b *builder) buildField(parentName string, p parse.NamedSchema, rootName string, required bool) *Field {
	f := &Field{
		JSONName:    p.Name,
		Type:        b.buildTypeRef(parentName, p.Name, p.Schema, rootName),
		Description: p.Schema.Description,
		Required:    required,
	}
	extractConstraints(p.Schema, f)
	return f
}

// buildFields builds a Field for each prop in order. isRequired resolves the
// required flag, absorbing the difference between a plain object (node.IsRequired)
// and allOf's merged requiredMap.
func (b *builder) buildFields(parentName string, props []parse.NamedSchema, rootName string, isRequired func(string) bool) []*Field {
	var fields []*Field
	for _, p := range props {
		fields = append(fields, b.buildField(parentName, p, rootName, isRequired(p.Name)))
	}
	return fields
}

// unseenProps returns the props whose names have not been seen yet, marking each
// returned name as seen. allOf uses this to dedup fields across branches with
// first-occurrence-wins semantics.
func unseenProps(props []parse.NamedSchema, seen map[string]bool) []parse.NamedSchema {
	var fresh []parse.NamedSchema
	for _, p := range props {
		if seen[p.Name] {
			continue
		}
		seen[p.Name] = true
		fresh = append(fresh, p)
	}
	return fresh
}

func (b *builder) buildEnum(name string, node *parse.SchemaNode) {
	t := &Type{
		Name:        name,
		Kind:        KindEnum,
		Description: node.Description,
		SourceFile:  b.currentSourceFile,
	}

	for _, v := range node.Enum {
		t.EnumValues = append(t.EnumValues, EnumValue{
			Name:  ToConstantCase(enumValueName(v)),
			Value: v,
		})
	}

	b.graph.AddType(t)
}

func (b *builder) buildUnion(name string, node *parse.SchemaNode, rootName string) {
	members := node.UnionMembers()

	t := &Type{
		Name:        name,
		Kind:        KindUnion,
		Description: node.Description,
		SourceFile:  b.currentSourceFile,
	}

	for _, m := range members {
		ref := b.buildTypeRef(name, "", m, rootName)
		t.UnionMembers = append(t.UnionMembers, ref)
	}

	b.graph.AddType(t)
}

func (b *builder) buildAllOf(name string, node *parse.SchemaNode, rootName string) {
	t := &Type{
		Name:        name,
		Kind:        KindStruct,
		Description: node.Description,
		SourceFile:  b.currentSourceFile,
	}

	seen := map[string]bool{}
	requiredMap := map[string]bool{}

	for _, req := range node.Required {
		requiredMap[req] = true
	}

	for _, branch := range node.AllOf {
		if branch.IsRef() {
			refName := b.resolveRefName(branch.Ref, rootName)
			t.Extends = append(t.Extends, refName)
			continue
		}

		if branch.IsObject() || len(branch.Properties) > 0 {
			for _, req := range branch.Required {
				requiredMap[req] = true
			}
			fresh := unseenProps(branch.Properties, seen)
			t.Fields = append(t.Fields, b.buildFields(name, fresh, rootName, func(n string) bool { return requiredMap[n] })...)
		}
	}

	fresh := unseenProps(node.Properties, seen)
	t.Fields = append(t.Fields, b.buildFields(name, fresh, rootName, node.IsRequired)...)

	b.graph.AddType(t)
}

func (b *builder) buildTypeRef(parentName, fieldName string, node *parse.SchemaNode, rootName string) *TypeRef {
	nullable := node.Type.IsNullable() || isNullType(node)

	if node.IsRef() {
		refName := b.resolveRefName(node.Ref, rootName)
		return &TypeRef{Kind: KindRef, TypeName: refName, Nullable: nullable}
	}

	if node.IsConst() {
		return b.buildConstTypeRef(node)
	}

	if node.IsEnum() {
		if b.cfg.ExtractInlined && fieldName != "" {
			enumName := ToPascalCase(fieldName)
			enumName = b.ensureUniqueName(enumName)
			b.buildEnum(enumName, node)
			return &TypeRef{Kind: KindRef, TypeName: enumName, Nullable: nullable}
		}
		prim := PrimUnknown
		if len(node.Enum) > 0 {
			prim = inferPrimitiveFromValue(node.Enum[0])
		}
		return &TypeRef{Kind: KindEnum, EnumValues: rawToEnumValues(node.Enum), Primitive: prim, Nullable: nullable}
	}

	if node.IsUnion() {
		members := node.UnionMembers()
		var refs []*TypeRef
		for i, m := range members {
			if m.IsObject() && len(m.Properties) > 0 {
				baseName := ToPascalCase(fieldName)
				if baseName == "" {
					baseName = parentName
				}
				memberName := uniqueMemberName(baseName, i)
				if b.cfg.ExtractInlined {
					memberName = b.ensureUniqueName(memberName)
				}
				b.buildStruct(memberName, m, rootName, true)
				refs = append(refs, &TypeRef{Kind: KindRef, TypeName: memberName})
			} else {
				refs = append(refs, b.buildTypeRef(parentName, fieldName, m, rootName))
			}
		}
		return &TypeRef{Kind: KindUnion, UnionMembers: refs, Nullable: nullable}
	}

	if node.IsArray() && node.Items != nil {
		itemRef := b.buildTypeRef(parentName, fieldName+"Item", node.Items, rootName)
		return &TypeRef{Kind: KindArray, ItemType: itemRef, Nullable: nullable}
	}

	if node.IsObject() && len(node.Properties) > 0 {
		if b.cfg.ExtractInlined && fieldName != "" {
			objName := ToPascalCase(fieldName)
			objName = b.ensureUniqueName(objName)
			b.buildStruct(objName, node, rootName, true)
			return &TypeRef{Kind: KindRef, TypeName: objName, Nullable: nullable}
		}
		sortedProps := sortedProperties(node.Properties)
		fields := b.buildFields(parentName, sortedProps, rootName, node.IsRequired)
		return &TypeRef{Kind: KindInterface, ObjectFields: fields, Nullable: nullable}
	}

	if node.IsObject() && node.AdditionalProperties != nil && node.AdditionalProperties.Schema != nil {
		valRef := b.buildTypeRef(parentName, "", node.AdditionalProperties.Schema, rootName)
		return &TypeRef{Kind: KindMap, ValueType: valRef, Nullable: nullable}
	}

	if node.IsObject() {
		return &TypeRef{
			Kind:      KindMap,
			ValueType: &TypeRef{Kind: KindPrimitive, Primitive: PrimUnknown},
			Nullable:  nullable,
		}
	}

	prim := mapPrimitive(node.Type.Single(), node.Format)
	return &TypeRef{
		Kind:      KindPrimitive,
		Primitive: prim,
		Format:    node.Format,
		Nullable:  nullable,
	}
}

func (b *builder) resolveRefName(ref string, rootName string) string {
	if ref == "#" {
		return rootName
	}
	if strings.HasPrefix(ref, RefPrefixDefs) || strings.HasPrefix(ref, RefPrefixDefinitions) {
		parts := strings.Split(ref, "/")
		return ToPascalCase(parts[len(parts)-1])
	}

	if name, ok := b.schemaNames[ref]; ok {
		return name
	}
	for _, variant := range refPathVariants(ref) {
		if name, ok := b.schemaNames[variant]; ok {
			return name
		}
	}

	b.warn("$ref %q could not be resolved to a known schema; using filename-derived name", ref)
	base := path.Base(ref)
	name := strings.TrimSuffix(base, path.Ext(base))
	return ToPascalCase(name)
}

func (b *builder) buildConstTypeRef(node *parse.SchemaNode) *TypeRef {
	return &TypeRef{
		Kind:       KindEnum,
		EnumValues: []EnumValue{{Value: node.Const}},
		Primitive:  inferPrimitiveFromValue(node.Const),
		Nullable:   node.Type.IsNullable(),
	}
}

func inferPrimitiveFromValue(val any) PrimitiveKind {
	switch val.(type) {
	case string:
		return PrimString
	case float64, int, int64:
		return PrimInt
	default:
		return PrimUnknown
	}
}

func (b *builder) ensureUniqueName(name string) string {
	if b.graph.GetType(name) == nil && !b.processedTypes[name] {
		return name
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s%d", name, i)
		if b.graph.GetType(candidate) == nil && !b.processedTypes[candidate] {
			return candidate
		}
	}
}
