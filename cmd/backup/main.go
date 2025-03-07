package main

import (
	"os"

	"github.com/emoss08/trenova/cmd/backup/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
