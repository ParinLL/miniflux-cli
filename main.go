package main

import (
	"errors"
	"log"
	"os"

	"github.com/parinll/miniflux-cli/internal/cli"
)

func main() {
	log.SetFlags(0)

	err := cli.Run(os.Args[1:], os.Stdout, os.Stderr)
	if err == nil {
		return
	}
	if errors.Is(err, cli.ErrUsage) {
		os.Exit(2)
	}

	log.Fatal(err)
}
