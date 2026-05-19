package py

import (
	"strings"
	"unicode"
)

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
