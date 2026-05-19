package graph

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/mirpo/schemagen/pkg/naming"
	"github.com/mirpo/schemagen/pkg/v2/parse"
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
			b.buildType(naming.ToPascalCase(d.Name), d.Schema, ns.Name)
		}

		rootName := naming.ToPascalCase(ns.Name)
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
	name := naming.ToPascalCase(ns.Name)
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

	for _, p := range props {
		t.Fields = append(t.Fields, b.buildField(name, p, rootName, node.IsRequired(p.Name)))
	}

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

func (b *builder) buildEnum(name string, node *parse.SchemaNode) {
	t := &Type{
		Name:        name,
		Kind:        KindEnum,
		Description: node.Description,
		SourceFile:  b.currentSourceFile,
		EnumType:    inferEnumType(node.Enum),
	}

	for _, v := range node.Enum {
		t.EnumValues = append(t.EnumValues, EnumValue{
			Name:  naming.ToConstantCase(enumValueName(v)),
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
			for _, p := range branch.Properties {
				if seen[p.Name] {
					continue
				}
				seen[p.Name] = true
				t.Fields = append(t.Fields, b.buildField(name, p, rootName, requiredMap[p.Name]))
			}
		}
	}

	for _, p := range node.Properties {
		if seen[p.Name] {
			continue
		}
		seen[p.Name] = true
		t.Fields = append(t.Fields, b.buildField(name, p, rootName, node.IsRequired(p.Name)))
	}

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
			enumName := naming.ToPascalCase(fieldName)
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
				baseName := naming.ToPascalCase(fieldName)
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
			objName := naming.ToPascalCase(fieldName)
			objName = b.ensureUniqueName(objName)
			b.buildStruct(objName, node, rootName, true)
			return &TypeRef{Kind: KindRef, TypeName: objName, Nullable: nullable}
		}
		sortedProps := sortedProperties(node.Properties)
		var fields []*Field
		for _, p := range sortedProps {
			fields = append(fields, b.buildField(parentName, p, rootName, node.IsRequired(p.Name)))
		}
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
		return naming.ToPascalCase(parts[len(parts)-1])
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
	return naming.ToPascalCase(name)
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

func extractConstraints(node *parse.SchemaNode, f *Field) {
	f.MinLength = node.MinLength
	f.MaxLength = node.MaxLength
	f.Pattern = node.Pattern
	f.Minimum = node.Minimum
	f.Maximum = node.Maximum
	f.ExclusiveMinimum = node.ExclusiveMinimum
	f.ExclusiveMaximum = node.ExclusiveMaximum
	f.MinItems = node.MinItems
	f.MaxItems = node.MaxItems
}

func mapPrimitive(typ, format string) PrimitiveKind {
	if format != "" {
		switch format {
		case FormatDateTime:
			return PrimDateTime
		case FormatDate:
			return PrimDate
		case FormatTime:
			return PrimTime
		case FormatUUID:
			return PrimUUID
		case FormatEmail:
			return PrimEmail
		case FormatURI, FormatURL:
			return PrimURI
		case FormatHostname:
			return PrimHostname
		case FormatIPv4:
			return PrimIPv4
		case FormatIPv6:
			return PrimIPv6
		case FormatInt32:
			return PrimInt32
		case FormatInt64:
			return PrimInt64
		case FormatFloat:
			return PrimFloat32
		case FormatDouble:
			return PrimFloat64
		}
	}

	switch typ {
	case parse.TypeString:
		return PrimString
	case parse.TypeInteger:
		return PrimInt
	case parse.TypeNumber:
		return PrimFloat64
	case parse.TypeBoolean:
		return PrimBool
	default:
		return PrimUnknown
	}
}

func sortedProperties(props []parse.NamedSchema) []parse.NamedSchema {
	sorted := make([]parse.NamedSchema, len(props))
	copy(sorted, props)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

func isNullType(node *parse.SchemaNode) bool {
	return len(node.Type) == 1 && node.Type[0] == parse.TypeNull
}

func inferEnumType(values []any) EnumKind {
	cat := analyzeRawValues(values)
	switch {
	case cat.AllStrings:
		return EnumKindString
	case cat.AllNumbers:
		return EnumKindInt
	default:
		return EnumKindMixed
	}
}

func rawToEnumValues(values []any) []EnumValue {
	if values == nil {
		return nil
	}
	result := make([]EnumValue, len(values))
	for i, v := range values {
		result[i] = EnumValue{Value: v}
	}
	return result
}

func enumValueName(v any) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

func uniqueMemberName(baseName string, index int) string {
	if index == 0 {
		return baseName
	}
	return fmt.Sprintf("%s%d", baseName, index)
}
