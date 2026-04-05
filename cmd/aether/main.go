// Package main is the CLI entry point for the aether colony system.
package main

import (
	"github.com/calcosmic/Aether/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		cmd.ExitWithError(err)
	}
}
