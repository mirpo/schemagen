package graph

import (
	"strings"
	"unicode"
)

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

func ToConstantCase(s string) string {
	if s == "" {
		return s
	}

	normalized := strings.Join(splitOnDelimiters(s), "_")
	snake := toSnakeCase(normalized)
	return strings.ToUpper(snake)
}

func isDelimiter(r rune) bool {
	return r == '_' || r == '-' || r == ' ' || r == '.'
}

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

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func toSnakeCase(s string) string {
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
