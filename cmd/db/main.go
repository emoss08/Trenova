package main

import (
	"os"

	"github.com/trenova-app/transport/cmd/db/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
