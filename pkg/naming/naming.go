package naming

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ToPascalCase converts a string to PascalCase.
// Examples: "user_name" -> "UserName", "api-key" -> "ApiKey"
func ToPascalCase(s string) string {
	if s == "" {
		return s
	}

	parts := splitOnDelimiters(s)
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(capitalizeFirst(p))
	}
	return b.String()
}

// ToCamelCase converts a string to camelCase.
// Examples: "user_name" -> "userName", "ApiKey" -> "apiKey"
func ToCamelCase(s string) string {
	if s == "" {
		return s
	}

	pascal := ToPascalCase(s)
	if pascal == "" {
		return pascal
	}

	r := []rune(pascal)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// ToSnakeCase converts a camelCase or PascalCase string to snake_case.
// Handles acronyms properly: "APIKey" -> "api_key", "HTTPResponse" -> "http_response"
func ToSnakeCase(s string) string {
	if s == "" {
		return s
	}

	runes := []rune(s)
	var b strings.Builder
	var prevWritten rune

	for i := range runes {
		r := runes[i]

		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				prevIsLower := unicode.IsLower(prev) || unicode.IsDigit(prev)
				nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])

				// Boundary cases:
				// - aB   => a_b
				// - ABc  => a_bc (acronym break before last cap if next is lower)
				if prevIsLower || (unicode.IsUpper(prev) && nextIsLower) {
					if prevWritten != 0 && prevWritten != '_' {
						b.WriteRune('_')
					}
				}
			}
			lower := unicode.ToLower(r)
			b.WriteRune(lower)
			prevWritten = lower
			continue
		}

		// Keep underscores as-is; lower everything else if it has case
		if r == '_' {
			b.WriteRune('_')
			prevWritten = '_'
			continue
		}
		lower := unicode.ToLower(r)
		b.WriteRune(lower)
		prevWritten = lower
	}

	return b.String()
}

// ToConstantCase converts a string to CONSTANT_CASE.
// Examples: "userName" -> "USER_NAME", "api-key" -> "API_KEY"
func ToConstantCase(s string) string {
	if s == "" {
		return s
	}

	normalized := strings.Join(splitOnDelimiters(s), "_")
	snake := ToSnakeCase(normalized)
	return strings.ToUpper(snake)
}

func IsPascalCase(s string) bool {
	if s == "" {
		return false
	}
	first, size := utf8.DecodeRuneInString(s)
	if !unicode.IsUpper(first) {
		return false
	}
	for _, r := range s[size:] {
		if isDelimiter(r) {
			return false
		}
	}
	return true
}

func isDelimiter(r rune) bool {
	return r == '_' || r == '-' || r == ' ' || r == '.'
}

// splitOnDelimiters splits a string on underscores, hyphens, spaces, and dots.
func splitOnDelimiters(s string) []string {
	var parts []string
	var cur strings.Builder

	flush := func() {
		if cur.Len() == 0 {
			return
		}
		parts = append(parts, cur.String())
		cur.Reset()
	}

	for _, r := range s {
		if isDelimiter(r) {
			flush()
		} else {
			cur.WriteRune(r)
		}
	}
	flush()

	if len(parts) == 0 {
		return nil
	}
	return parts
}

// capitalizeFirst uppercases the first rune of a string (unicode-safe).
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// ToGoFieldName converts a property name to a valid Go field name.
func ToGoFieldName(name string) string {
	if name == "" {
		return "Field"
	}

	needsPrefix := false
	first := rune(name[0])
	if !unicode.IsLetter(first) && first != '_' {
		needsPrefix = true
	}

	// Replace dots with hyphens so they become word delimiters
	cleaned := strings.ReplaceAll(name, ".", "-")

	// Remove special characters but keep delimiters
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

	// Uppercase first letter rune we find (keeps digits in front)
	rs := []rune(cleaned)
	for i := range rs {
		if unicode.IsLetter(rs[i]) {
			rs[i] = unicode.ToUpper(rs[i])
			break
		}
	}
	cleaned = string(rs)

	pascal := ToPascalCase(cleaned)
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

// isValidGoIdentifier checks if a string is a valid Go identifier.
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
