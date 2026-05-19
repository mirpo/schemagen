package golang

import (
	"strings"
	"unicode"

	"github.com/mirpo/schemagen/pkg/graph"
)

func toGoFieldName(name string) string {
	if name == "" {
		return "Field"
	}

	needsPrefix := false
	first := rune(name[0])
	if !unicode.IsLetter(first) && first != '_' {
		needsPrefix = true
	}

	cleaned := strings.ReplaceAll(name, ".", "-")

	var filtered strings.Builder
	for _, r := range cleaned {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == ' ' {
			filtered.WriteRune(r)
		}
	}
	cleaned = filtered.String()
	if cleaned == "" {
		return "Field"
	}

	rs := []rune(cleaned)
	for i := range rs {
		if unicode.IsLetter(rs[i]) {
			rs[i] = unicode.ToUpper(rs[i])
			break
		}
	}
	cleaned = string(rs)

	pascal := graph.ToPascalCase(cleaned)
	if pascal == "" {
		return "Field"
	}

	if needsPrefix {
		pascal = "F" + pascal
	}

	if !isValidGoIdentifier(pascal) {
		return "Field"
	}
	return pascal
}

func isValidGoIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
		} else {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
				return false
			}
		}
	}
	return true
}
