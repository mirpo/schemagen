//go:build ignore

package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: parse_go.go <file>...")
		os.Exit(1)
	}

	fset := token.NewFileSet()
	failed := false

	for _, path := range os.Args[1:] {
		_, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
}
