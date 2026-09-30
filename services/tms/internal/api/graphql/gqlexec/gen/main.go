package main

import (
	"flag"
	"fmt"
	"os"
)

var configPath = flag.String("config", "gqlgen.yml", "Path to gqlgen.yml")

func main() {
	flag.Parse()

	if err := run(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "gqlexec: %v\n", err)
		os.Exit(1)
	}
}
