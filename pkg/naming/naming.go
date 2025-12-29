package naming

import (
	"strings"
	"unicode"
)

// ToPascalCase converts a string to PascalCase.
// Examples: "user_name" -> "UserName", "api-key" -> "ApiKey"
func ToPascalCase(s string) string {
	if s == "" {
		return s
	}

	parts := splitOnDelimiters(s)
	result := ""
	for _, part := range parts {
		if len(part) > 0 {
			result += capitalizeFirst(part)
		}
	}

	return result
}

// ToCamelCase converts a string to camelCase.
// Examples: "user_name" -> "userName", "ApiKey" -> "apiKey"
func ToCamelCase(s string) string {
	if s == "" {
		return s
	}

	pascal := ToPascalCase(s)
	if len(pascal) == 0 {
		return pascal
	}

	// Lowercase the first character
	runes := []rune(pascal)
	runes[0] = toLower(runes[0])
	return string(runes)
}

// ToSnakeCase converts a camelCase or PascalCase string to snake_case.
// Handles acronyms properly: "APIKey" -> "api_key", "HTTPResponse" -> "http_response"
func ToSnakeCase(s string) string {
	if s == "" {
		return s
	}

	runes := []rune(s)
	var result strings.Builder

	for i := range runes {
		r := runes[i]

		// Check if this is an uppercase letter
		if isUpper(r) {
			// Add underscore before uppercase if:
			// 1. Not the first character AND
			// 2. Either:
			//    a) Previous char is lowercase (start of new word)
			//    b) Previous char is uppercase AND next char is lowercase (end of acronym, start of word)
			if i > 0 {
				prevIsLower := !isUpper(runes[i-1]) && runes[i-1] != '_'
				nextIsLower := i+1 < len(runes) && !isUpper(runes[i+1])

				if prevIsLower || (isUpper(runes[i-1]) && nextIsLower) {
					result.WriteRune('_')
				}
			}
		}

		result.WriteRune(toLower(r))
	}

	return result.String()
}

// ToConstantCase converts a string to CONSTANT_CASE.
// Examples: "userName" -> "USER_NAME", "api-key" -> "API_KEY"
func ToConstantCase(s string) string {
	if s == "" {
		return s
	}

	// Replace delimiters with underscores and insert underscores before capitals
	var result []rune
	for i, r := range s {
		// Replace hyphens, spaces, and dots with underscores
		if r == '-' || r == ' ' || r == '.' {
			result = append(result, '_')
		} else {
			// Insert underscore before capitals in camelCase
			if i > 0 && isUpper(r) && !isUpper(rune(s[i-1])) && s[i-1] != '_' && s[i-1] != '-' {
				result = append(result, '_')
			}
			result = append(result, toUpper(r))
		}
	}

	return string(result)
}

// IsPascalCase checks if a string is in PascalCase format.
func IsPascalCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Check if first character is uppercase
	return s[0] >= 'A' && s[0] <= 'Z'
}

// Helper functions

// splitOnDelimiters splits a string on underscores, hyphens, and spaces.
func splitOnDelimiters(s string) []string {
	var parts []string
	current := ""

	for _, r := range s {
		if r == '_' || r == '-' || r == ' ' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// capitalizeFirst capitalizes the first character of a string.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = toUpper(runes[0])
	return string(runes)
}

// isUpper checks if a rune is an uppercase letter.
func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

// toUpper converts a lowercase letter to uppercase.
func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 32
	}
	return r
}

// toLower converts an uppercase letter to lowercase.
func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}

// ToGoFieldName converts a property name to a valid Go field name.
// It first applies PascalCase conversion, then checks if the result is a valid Go identifier.
// If not, it sanitizes the name by:
// - Replacing dots with hyphens (so they're treated as word delimiters)
// - Removing special characters like $, @, #
// - Prefixing with 'F' if original starts with digit or non-letter
// - Applying PascalCase conversion
func ToGoFieldName(name string) string {
	if name == "" {
		return "Field"
	}

	// Step 1: Check if ORIGINAL name starts with invalid char (digit or non-letter)
	needsPrefix := false
	if len(name) > 0 {
		first := rune(name[0])
		if !unicode.IsLetter(first) && first != '_' {
			needsPrefix = true
		}
	}

	// Step 2: Replace dots with hyphens so they become word delimiters
	cleaned := strings.ReplaceAll(name, ".", "-")

	// Step 3: Remove special characters but keep delimiters
	var filtered strings.Builder
	for _, r := range cleaned {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == ' ' {
			filtered.WriteRune(r)
		}
		// Skip special chars like $, @, #
	}
	cleaned = filtered.String()

	if cleaned == "" {
		return "Field"
	}

	// Step 4: Capitalize first letter if it's lowercase
	// This ensures "123numeric" becomes "123Numeric" before adding prefix
	runes := []rune(cleaned)
	for i, r := range runes {
		if unicode.IsLetter(r) {
			runes[i] = unicode.ToUpper(r)
			break
		}
	}
	cleaned = string(runes)

	// Step 5: Apply PascalCase conversion
	pascalName := ToPascalCase(cleaned)

	// Step 6: Add prefix if needed
	if needsPrefix {
		pascalName = "F" + pascalName
	}

	// Step 7: Final validation
	if !isValidGoIdentifier(pascalName) {
		return "Field"
	}

	return pascalName
}

// isValidGoIdentifier checks if a string is a valid Go identifier.
// Go identifiers must:
// - Start with a letter or underscore
// - Contain only letters, digits, and underscores
func isValidGoIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}

	// Check each rune
	for i, r := range s {
		if i == 0 {
			// First character must be letter or underscore
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
		} else {
			// Subsequent characters must be letter, digit, or underscore
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
				return false
			}
		}
	}

	return true
}
