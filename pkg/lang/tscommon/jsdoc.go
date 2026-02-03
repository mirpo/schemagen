package tscommon

import (
	"fmt"
	"strings"
)

// WriteJSDoc writes a JSDoc comment block to the builder.
// If description is empty, nothing is written.
func WriteJSDoc(sb *strings.Builder, indent, description string) {
	if description == "" {
		return
	}
	sb.WriteString(indent)
	sb.WriteString("/**\n")
	sb.WriteString(indent)
	fmt.Fprintf(sb, " * %s\n", description)
	sb.WriteString(indent)
	sb.WriteString(" */\n")
}

// WriteJSDocSingleLine writes a single-line JSDoc comment.
// If description is empty, nothing is written.
func WriteJSDocSingleLine(sb *strings.Builder, indent, description string) {
	if description == "" {
		return
	}
	sb.WriteString(indent)
	fmt.Fprintf(sb, "/** %s */\n", description)
}

// WriteJSDocWithFormat writes a JSDoc comment with optional description and format annotation.
// Writes nothing if both description and format are empty.
func WriteJSDocWithFormat(sb *strings.Builder, indent, description, format string) {
	if description == "" && format == "" {
		return
	}

	// Single-line if only description, no format
	if description != "" && format == "" {
		WriteJSDocSingleLine(sb, indent, description)
		return
	}

	// Multi-line for format annotation
	sb.WriteString(indent)
	sb.WriteString("/**\n")
	if description != "" {
		sb.WriteString(indent)
		fmt.Fprintf(sb, " * %s\n", description)
	}
	if format != "" {
		sb.WriteString(indent)
		fmt.Fprintf(sb, " * @format %s\n", format)
	}
	sb.WriteString(indent)
	sb.WriteString(" */\n")
}
