package tsutil

import "unicode"

var reservedKeywords = map[string]bool{
	"break": true, "case": true, "catch": true, "continue": true,
	"debugger": true, "default": true, "delete": true, "do": true,
	"else": true, "finally": true, "for": true, "function": true,
	"if": true, "in": true, "instanceof": true, "new": true,
	"return": true, "switch": true, "this": true, "throw": true,
	"try": true, "typeof": true, "var": true, "void": true,
	"while": true, "with": true, "class": true, "const": true,
	"enum": true, "export": true, "extends": true, "import": true,
	"super": true, "implements": true, "interface": true, "let": true,
	"package": true, "private": true, "protected": true, "public": true,
	"static": true, "yield": true, "await": true,
}

// NeedsQuoting returns true if the property name needs to be quoted in JS/TS.
// A valid JS identifier must:
// - Start with a letter, underscore, or dollar sign
// - Contain only letters, digits, underscores, or dollar signs
// - Not be a reserved keyword
func NeedsQuoting(name string) bool {
	if len(name) == 0 {
		return true
	}

	// Must start with letter, underscore, or dollar sign
	first := rune(name[0])
	if !unicode.IsLetter(first) && first != '_' && first != '$' {
		return true
	}

	// Rest must be alphanumeric, underscore, or dollar sign
	for _, r := range name[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '$' {
			return true
		}
	}

	// Check for reserved words
	return reservedKeywords[name]
}
