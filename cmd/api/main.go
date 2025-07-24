/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package main

import (
	"log"
	"os"

	"github.com/emoss08/trenova/cmd/api/commands"
)

func main() {
	root := commands.NewRootCmd()
	if err := root.Execute(); err != nil {
		log.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
