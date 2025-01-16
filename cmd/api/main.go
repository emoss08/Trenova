package main

import (
	"log"
	"os"

	"github.com/trenova-app/transport/cmd/api/commands"
)

func main() {
	root := commands.NewRootCmd()
	if err := root.Execute(); err != nil {
		log.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
