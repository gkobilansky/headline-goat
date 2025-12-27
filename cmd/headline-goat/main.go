package main

import (
	"os"

	"github.com/headline-goat/headline-goat/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
