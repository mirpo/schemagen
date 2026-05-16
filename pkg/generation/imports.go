package generation

import (
	"github.com/mirpo/schemagen/pkg/typegraph"
)

// ImportConverter transforms ImportSpec for language-specific needs.
type ImportConverter interface {
	Convert(imports []typegraph.ImportSpec) []typegraph.ImportSpec
}

// PassthroughConverter copies imports without transformation (used by TypeScript).
type PassthroughConverter struct{}

func (c *PassthroughConverter) Convert(imports []typegraph.ImportSpec) []typegraph.ImportSpec {
	return imports
}
