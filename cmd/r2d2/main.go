package main

import (
	"os"

	"github.com/ryangerardwilson/r2d2/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args, os.Stdout, os.Stderr))
}
