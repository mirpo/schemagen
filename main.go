package main

import (
	"os"

	"github.com/mirpo/schemagen/cmd"
	"github.com/mirpo/schemagen/pkg/errors"
)

func main() {
	if err := cmd.Execute(); err != nil {
		if exitErr, ok := err.(*errors.ExitCodeError); ok {
			os.Exit(exitErr.Code)
		}
		os.Exit(1)
	}
}
