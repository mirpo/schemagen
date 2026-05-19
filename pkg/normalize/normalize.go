package normalize

import (
	"strings"

	"github.com/mirpo/schemagen/pkg/parse"
)

type Draft string

const (
	DraftUnknown Draft = ""
	Draft04      Draft = "draft-04"
	Draft06      Draft = "draft-06"
	Draft07      Draft = "draft-07"
	Draft201909  Draft = "2019-09"
	Draft202012  Draft = "2020-12"
)

func DetectDraft(node *parse.SchemaNode) Draft {
	s := node.Schema
	switch {
	case strings.Contains(s, "draft-04"):
		return Draft04
	case strings.Contains(s, "draft-06"):
		return Draft06
	case strings.Contains(s, "draft-07"):
		return Draft07
	case strings.Contains(s, "2019-09"):
		return Draft201909
	case strings.Contains(s, "2020-12"):
		return Draft202012
	default:
		return DraftUnknown
	}
}

func Normalize(node *parse.SchemaNode) {
	if node == nil {
		return
	}

	for _, d := range node.Defs {
		Normalize(d.Schema)
	}
	for _, p := range node.Properties {
		Normalize(p.Schema)
	}
	for _, s := range node.AllOf {
		Normalize(s)
	}
	for _, s := range node.AnyOf {
		Normalize(s)
	}
	for _, s := range node.OneOf {
		Normalize(s)
	}
	if node.Items != nil {
		Normalize(node.Items)
	}
}
